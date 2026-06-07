// Package authorizationcode provides OAuth2 authorization code flow authentication for Canton gRPC connections.
// This flow is intended for local development where a browser-based login is available; it is not suitable for CI.
package authorizationcode

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	cantonauth "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton/provider/authentication"
)

var _ cantonauth.Provider = Provider{}

// Provider implements authentication.Provider using the OAuth2 authorization code flow with PKCE (S256).
type Provider struct {
	tokenSource          oauth.TokenSource
	transportCredentials credentials.TransportCredentials
}

type authorizationCodeProviderConfig struct {
	scopes               []string
	transportCredentials credentials.TransportCredentials
	callbackURL          string
	openBrowser          bool
	timeout              time.Duration
}

func defaultAuthorizationCodeProviderConfig() *authorizationCodeProviderConfig {
	return &authorizationCodeProviderConfig{
		scopes: []string{"openid", "daml_ledger_api"},
		transportCredentials: credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}),
		callbackURL: "http://127.0.0.1:8400/callback",
		openBrowser: true,
	}
}

// ProviderOption configures the authorization code Provider.
type ProviderOption func(*authorizationCodeProviderConfig)

// WithScopes configures the scopes requested from the authorization server.
func WithScopes(scopes ...string) ProviderOption {
	return func(config *authorizationCodeProviderConfig) {
		config.scopes = scopes
	}
}

// WithTransportCredentials configures transport credentials for gRPC connections.
func WithTransportCredentials(creds credentials.TransportCredentials) ProviderOption {
	return func(config *authorizationCodeProviderConfig) {
		config.transportCredentials = creds
	}
}

// WithCallbackURL configures the local redirect URI used by the authorization server.
func WithCallbackURL(callbackURL string) ProviderOption {
	return func(config *authorizationCodeProviderConfig) {
		config.callbackURL = callbackURL
	}
}

// WithOpenBrowser controls whether the default browser is opened automatically.
func WithOpenBrowser(openBrowser bool) ProviderOption {
	return func(config *authorizationCodeProviderConfig) {
		config.openBrowser = openBrowser
	}
}

// WithTimeout configures a timeout for the overall authorization flow.
func WithTimeout(timeout time.Duration) ProviderOption {
	return func(config *authorizationCodeProviderConfig) {
		config.timeout = timeout
	}
}

// NewDiscoveryProvider creates a provider using OAuth2 Authorization Server Metadata discovery (RFC 8414).
// PKCE with the S256 challenge method is required.
func NewDiscoveryProvider(
	ctx context.Context,
	authorizationServerURL, clientID string,
	options ...ProviderOption,
) (*Provider, error) {
	metadata, err := cantonauth.GetAuthorizationServerMetadata(ctx, authorizationServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization server metadata: %w", err)
	}

	if !slices.Contains(metadata.CodeChallengeMethodsSupported, "S256") {
		return nil, errors.New("authorization server does not support S256 PKCE challenges")
	}

	return NewProvider(ctx, metadata.AuthorizationEndpoint, metadata.TokenEndpoint, clientID, options...)
}

// NewProvider creates a provider that performs the OAuth2 authorization code flow with PKCE (S256).
func NewProvider(
	ctx context.Context,
	authURL, tokenURL, clientID string,
	options ...ProviderOption,
) (*Provider, error) {
	cfg := defaultAuthorizationCodeProviderConfig()
	for _, option := range options {
		option(cfg)
	}

	if authURL == "" {
		return nil, errors.New("authURL cannot be empty")
	}
	if tokenURL == "" {
		return nil, errors.New("tokenURL cannot be empty")
	}
	if clientID == "" {
		return nil, errors.New("clientID cannot be empty")
	}

	flowCtx := ctx
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		flowCtx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	callbackURL, err := url.Parse(cfg.callbackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse callback URL: %w", err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: callbackURL.String(),
		Scopes:      cfg.scopes,
		Endpoint:    oauth2.Endpoint{AuthURL: authURL, TokenURL: tokenURL},
	}

	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()
	authCodeURL := oauthCfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	callbackChan := make(chan *oauth2.Token, 1)
	var deliverOnce sync.Once

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(callbackURL.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := q.Get("code")
		receivedState := q.Get("state")

		if receivedState != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}
		if code == "" {
			http.Error(w, "No code parameter received", http.StatusBadRequest)
			return
		}

		token, exchangeErr := oauthCfg.Exchange(flowCtx, code, oauth2.VerifierOption(verifier))
		if exchangeErr != nil {
			fmt.Fprintf(os.Stderr, "authorization code token exchange failed: %v\n", exchangeErr)
			http.Error(w, fmt.Sprintf("Token exchange failed: %v", exchangeErr), http.StatusInternalServerError)
			return
		}

		deliverOnce.Do(func() {
			callbackChan <- token
		})

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
		Addr:              callbackURL.Host,
		Handler:           serveMux,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	listener, err := new(net.ListenConfig).Listen(flowCtx, "tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("creating listener: %w", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(listener)
	}()

	if cfg.openBrowser {
		fmt.Println("Attempting to open your default browser.")
		fmt.Println("If the browser does not open, visit the following URL:")
		fmt.Println(authCodeURL)
		openBrowser(flowCtx, authCodeURL)
	} else {
		fmt.Println("Visit the following URL:")
		fmt.Println(authCodeURL)
	}

	shutdown := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}

	select {
	case err := <-serverErr:
		shutdown()
		return nil, fmt.Errorf("callback server error: %w", err)
	case token := <-callbackChan:
		shutdown()
		tokenSource := oauthCfg.TokenSource(flowCtx, token)

		return &Provider{
			tokenSource:          oauth.TokenSource{TokenSource: tokenSource},
			transportCredentials: cfg.transportCredentials,
		}, nil
	case <-flowCtx.Done():
		shutdown()
		return nil, flowCtx.Err()
	}
}

func (p Provider) TokenSource() oauth2.TokenSource {
	return p.tokenSource.TokenSource
}

func (p Provider) TransportCredentials() credentials.TransportCredentials {
	return p.transportCredentials
}

func (p Provider) PerRPCCredentials() credentials.PerRPCCredentials {
	return p.tokenSource
}

func openBrowser(ctx context.Context, targetURL string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.CommandContext(ctx, "open", targetURL).Start()
	case "linux":
		_ = exec.CommandContext(ctx, "xdg-open", targetURL).Start()
	case "windows":
		_ = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", targetURL).Start()
	}
}
