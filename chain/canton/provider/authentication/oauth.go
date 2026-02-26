package authentication

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
)

var _ Provider = (*OIDCProvider)(nil)

// OIDCProvider implements Provider using OAuth2/OIDC token flows (client credentials or authorization code).
type OIDCProvider struct {
	tokenSource oauth2.TokenSource
}

// NewClientCredentialsProvider creates a provider that fetches tokens using the OAuth2 client credentials flow.
// Use in CI where ClientID, ClientSecret and AuthURL are available; tokens are obtained automatically.
func NewClientCredentialsProvider(ctx context.Context, authURL, clientID, clientSecret string) (*OIDCProvider, error) {
	tokenURL := fmt.Sprintf("%s/v1/token", authURL)

	oauthCfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		Scopes:       []string{"daml_ledger_api"},
	}

	tokenSource := oauthCfg.TokenSource(ctx)

	return &OIDCProvider{
		tokenSource: tokenSource,
	}, nil
}

// NewAuthorizationCodeProvider creates a provider that uses the OAuth2 authorization code flow with PKCE.
// It starts a local callback server, opens the browser to the auth URL, and exchanges the code for a token.
// Use locally to skip canton-login; only ClientID and AuthURL are required.
func NewAuthorizationCodeProvider(ctx context.Context, authURL, clientID string) (*OIDCProvider, error) {
	verifier := oauth2.GenerateVerifier()

	port := 8400
	authEndpoint := fmt.Sprintf("%s/v1/authorize", authURL)
	tokenEndpoint := fmt.Sprintf("%s/v1/token", authURL)
	redirectURL := fmt.Sprintf("http://localhost:%d", port)

	oauthCfg := &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURL + "/callback",
		Scopes:      []string{"openid", "daml_ledger_api"},
		Endpoint:    oauth2.Endpoint{AuthURL: authEndpoint, TokenURL: tokenEndpoint},
	}

	state := generateState()
	authCodeURL := oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	callbackChan := make(chan *oauth2.Token)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := q.Get("code")
		receivedState := q.Get("state")

		if receivedState != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		token, err := oauthCfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		if err != nil {
			http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		callbackChan <- token

		html := `<!DOCTYPE html>
<html>
<head><title>Authentication Complete</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 40px;">
	<h1>Authentication complete!</h1>
	<p>You can safely close this window.</p>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	})

	server := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           serveMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("listening on port %d: %w", port, err)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	openBrowser(authCodeURL)

	select {
	case err := <-serverErr:
		_ = server.Shutdown(ctx)
		return nil, fmt.Errorf("callback server error: %w", err)
	case token := <-callbackChan:
		tokenSource := oauthCfg.TokenSource(ctx, token)
		_ = server.Shutdown(ctx)
		return &OIDCProvider{
			tokenSource: tokenSource,
		}, nil
	case <-ctx.Done():
		_ = server.Shutdown(ctx)
		return nil, ctx.Err()
	}
}

func (p *OIDCProvider) TokenSource() oauth2.TokenSource {
	return p.tokenSource
}

func (p *OIDCProvider) TransportCredentials() credentials.TransportCredentials {
	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
	})
}

func (p *OIDCProvider) PerRPCCredentials() credentials.PerRPCCredentials {
	return secureTokenSource{
		TokenSource: p.tokenSource,
	}
}

func generateState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// openBrowser opens the default browser to url on supported platforms; otherwise it is a no-op.
func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
