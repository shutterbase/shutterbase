package vault

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"net"
	"net/http"
	"os/exec"
	"path"
	"runtime"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

// loginOIDC performs vault's CLI-style OIDC browser dance: request an auth URL,
// open the browser, and catch the provider redirect on a local callback server.
// Interactive by design — the local-dev fallback when neither a kubernetes role
// nor a token is configured.
func (c *Client) loginOIDC(ctx context.Context) (*vaultapi.Secret, error) {
	mount := c.options.OIDCMount
	if mount == "" {
		mount = "oidc"
	}
	port := c.options.OIDCCallbackPort
	if port == 0 {
		port = 8250
	}

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, err
	}
	clientNonce := hex.EncodeToString(nonceBytes)

	redirectURI := fmt.Sprintf("http://localhost:%d/oidc/callback", port)
	authURLSecret, err := c.Logical().WriteWithContext(ctx, fmt.Sprintf("auth/%s/oidc/auth_url", mount), map[string]interface{}{
		"redirect_uri": redirectURI,
		"nonce":        clientNonce,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get OIDC auth URL: %w", err)
	}
	authURL, _ := authURLSecret.Data["auth_url"].(string)
	if authURL == "" {
		return nil, fmt.Errorf("vault returned no OIDC auth URL for mount %q", mount)
	}

	type loginResult struct {
		secret *vaultapi.Secret
		err    error
	}
	doneCh := make(chan loginResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oidc/callback", func(w http.ResponseWriter, req *http.Request) {
		data := map[string][]string{
			"state":        {req.FormValue("state")},
			"code":         {req.FormValue("code")},
			"id_token":     {req.FormValue("id_token")},
			"client_nonce": {clientNonce},
		}
		// form_post response mode: relay the POST body to vault first, then
		// complete the flow with the regular GET callback below.
		if req.Method == http.MethodPost {
			postURL := c.Address() + path.Join("/v1/auth", mount, "oidc/callback")
			resp, err := http.PostForm(postURL, data)
			if err != nil {
				fmt.Fprint(w, "<h3>Vault login failed.</h3>")
				doneCh <- loginResult{nil, err}
				return
			}
			resp.Body.Close()
			delete(data, "id_token")
		}
		secret, err := c.Logical().ReadWithData(fmt.Sprintf("auth/%s/oidc/callback", mount), data)
		if err != nil {
			fmt.Fprintf(w, "<h3>Vault login failed.</h3><pre>%s</pre>", html.EscapeString(err.Error()))
		} else {
			fmt.Fprint(w, "<h3>Vault login successful.</h3>You can close this window.")
		}
		doneCh <- loginResult{secret, err}
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to open OIDC callback listener: %w", err)
	}
	defer listener.Close()
	go http.Serve(listener, mux) //nolint:errcheck // listener closes when the login completes

	openBrowser(authURL)
	log.Info().Str("url", authURL).Msg("complete the vault OIDC login in your browser")

	select {
	case result := <-doneCh:
		return result.secret, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Minute):
		return nil, fmt.Errorf("timed out waiting for OIDC login")
	}
}

// openBrowser is best-effort: on failure the auth URL is already logged.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Debug().Err(err).Msg("could not open browser automatically")
	}
}
