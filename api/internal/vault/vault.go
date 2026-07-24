// Package vault fetches credentials from HashiCorp Vault / OpenBao (the API is
// identical). Adapted from gitlab.fs-g.org/global/go-fsg-lib/pkg/vault, stripped
// of everything FSG-specific (CRDB JWT auth, AWS auth, transit, TLS certify).
package vault

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/rs/zerolog/log"
)

// Options configures the Vault client. Auth methods are tried in this order:
// kubernetes (when KubernetesRole is set), static token (when Token is set),
// interactive OIDC browser login (the local-dev fallback).
type Options struct {
	Address          string
	Token            string
	KubernetesRole   string
	OIDCMount        string
	OIDCCallbackPort int
}

type Client struct {
	*vaultapi.Client
	options *Options
}

func NewClient(ctx context.Context, options *Options) (*Client, error) {
	// DefaultConfig also honors the standard VAULT_* env vars (VAULT_CACERT etc.)
	apiClient, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}
	if options.Address != "" {
		if err := apiClient.SetAddress(options.Address); err != nil {
			return nil, fmt.Errorf("invalid vault address: %w", err)
		}
	}
	client := &Client{Client: apiClient, options: options}
	if err := client.authenticate(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Client) authenticate(ctx context.Context) error {
	if c.options.KubernetesRole != "" {
		auth, err := kubernetes.NewKubernetesAuth(c.options.KubernetesRole)
		if err != nil {
			return fmt.Errorf("failed to create kubernetes auth: %w", err)
		}
		secret, err := c.Auth().Login(ctx, auth)
		if err != nil {
			return fmt.Errorf("vault kubernetes login failed: %w", err)
		}
		log.Info().Msg("authenticated to vault via kubernetes auth")
		c.startTokenRenewal(ctx, secret, func() (*vaultapi.Secret, error) {
			return c.Auth().Login(ctx, auth)
		})
		return nil
	}

	if c.options.Token != "" {
		c.SetToken(c.options.Token)
		secret, err := c.tokenAuthSecret()
		if err != nil {
			return fmt.Errorf("vault token auth failed: %w", err)
		}
		log.Info().Msg("authenticated to vault via static token")
		if secret != nil {
			c.startTokenRenewal(ctx, secret, nil)
		}
		return nil
	}

	// Reuse an existing `vault login` CLI session (~/.vault-token) before
	// falling back to the interactive OIDC dance; an expired file token falls
	// through instead of failing.
	if token := readTokenHelperFile(); token != "" {
		c.SetToken(token)
		secret, err := c.tokenAuthSecret()
		if err == nil {
			log.Info().Msg("authenticated to vault via ~/.vault-token (vault CLI session)")
			if secret != nil {
				c.startTokenRenewal(ctx, secret, nil)
			}
			return nil
		}
		log.Debug().Err(err).Msg("~/.vault-token is invalid or expired; falling back to OIDC login")
		c.ClearToken()
	}

	secret, err := c.loginOIDC(ctx)
	if err != nil {
		return fmt.Errorf("vault OIDC login failed: %w", err)
	}
	c.SetToken(secret.Auth.ClientToken)
	log.Info().Msg("authenticated to vault via OIDC browser login")
	c.startTokenRenewal(ctx, secret, nil)
	return nil
}

// readTokenHelperFile returns the token stored by `vault login` in
// ~/.vault-token, or "" when absent. The api library does not read the CLI's
// token helper itself.
func readTokenHelperFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(home, ".vault-token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// tokenAuthSecret validates the static token and, when it is renewable, wraps
// it into an auth secret the LifetimeWatcher can renew. Non-expiring tokens
// (root/dev) return nil: nothing to renew.
func (c *Client) tokenAuthSecret() (*vaultapi.Secret, error) {
	self, err := c.Auth().Token().LookupSelf()
	if err != nil {
		return nil, err
	}
	renewable, err := self.TokenIsRenewable()
	if err != nil {
		return nil, err
	}
	ttl, err := self.TokenTTL()
	if err != nil {
		return nil, err
	}
	if !renewable || ttl <= 0 {
		return nil, nil
	}
	return &vaultapi.Secret{Auth: &vaultapi.SecretAuth{
		ClientToken:   c.Token(),
		Renewable:     true,
		LeaseDuration: int(ttl / time.Second),
	}}, nil
}

// startTokenRenewal keeps the vault auth token alive for the process lifetime.
// When the token hits its max TTL, reLogin (kubernetes) re-authenticates;
// without one the process exits so a restart performs a fresh login.
func (c *Client) startTokenRenewal(ctx context.Context, secret *vaultapi.Secret, reLogin func() (*vaultapi.Secret, error)) {
	go func() {
		for {
			watcher, err := c.NewLifetimeWatcher(&vaultapi.LifetimeWatcherInput{Secret: secret})
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create vault token watcher")
			}
			go watcher.Start()
			for renewing := true; renewing; {
				select {
				case <-ctx.Done():
					watcher.Stop()
					return
				case renewal := <-watcher.RenewCh():
					c.SetToken(renewal.Secret.Auth.ClientToken)
					log.Debug().Msg("renewed vault token")
				case err := <-watcher.DoneCh():
					if reLogin == nil {
						log.Fatal().Err(err).Msg("vault token expired and cannot be renewed; exiting for a fresh login on restart")
					}
					secret, err = reLogin()
					if err != nil {
						log.Fatal().Err(err).Msg("vault re-login failed")
					}
					log.Info().Msg("re-authenticated to vault")
					renewing = false
				}
			}
		}
	}()
}
