package vault

import (
	"context"
	"fmt"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

// DatabaseCredentials are dynamic credentials issued by vault's database
// secrets engine.
type DatabaseCredentials struct {
	Username string
	Password string
}

// GetDatabaseCredentials reads dynamic DB credentials from path (e.g.
// "database/creds/shutterbase") and starts a background renewer for their
// lease. The LifetimeWatcher renews at ~2/3 of the lease TTL (with jitter) and
// retries transient failures with backoff, so the lease stays alive until its
// max_ttl — or until the auth token itself expires, which revokes child leases.
// ponytail: at that point the process exits and the restart is the credential
// rotation; reopening the pool with re-read creds in-flight is the upgrade if
// restarts ever hurt.
func (c *Client) GetDatabaseCredentials(ctx context.Context, path string) (*DatabaseCredentials, error) {
	secret, err := c.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read database credentials from vault at %q: %w", path, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("no database credentials found in vault at %q", path)
	}
	username, _ := secret.Data["username"].(string)
	password, _ := secret.Data["password"].(string)
	if username == "" || password == "" {
		return nil, fmt.Errorf("vault secret at %q is missing username/password", path)
	}

	watcher, err := c.NewLifetimeWatcher(&vaultapi.LifetimeWatcherInput{Secret: secret})
	if err != nil {
		return nil, fmt.Errorf("failed to create database lease watcher: %w", err)
	}
	go watcher.Start()
	go func() {
		for {
			select {
			case <-ctx.Done():
				watcher.Stop()
				return
			case renewal := <-watcher.RenewCh():
				log.Debug().Int("lease_duration", renewal.Secret.LeaseDuration).Msg("renewed database credential lease")
			case err := <-watcher.DoneCh():
				log.Fatal().Err(err).Msg("database credential lease expired and cannot be renewed; exiting for fresh credentials on restart")
			}
		}
	}()

	return &DatabaseCredentials{Username: username, Password: password}, nil
}
