// Package clientcredentials provides OAuth2 client credentials flow authentication for Canton gRPC connections.
// This flow is designed for machine-to-machine authentication and is intended for CI/CD environments.
package clientcredentials

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"

	cantonauth "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton/provider/authentication"
)

var _ cantonauth.Provider = Provider{}

// Provider implements authentication.Provider using the OAuth2 client credentials flow.
type Provider struct {
	tokenSource          oauth.TokenSource
	transportCredentials credentials.TransportCredentials
}

type clientCredentialsProviderConfig struct {
	scopes               []string
	transportCredentials credentials.TransportCredentials
}

func defaultClientCredentialsProviderConfig() *clientCredentialsProviderConfig {
	return &clientCredentialsProviderConfig{
		scopes: []string{"daml_ledger_api"},
		transportCredentials: credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		}),
	}
}

// ProviderOption configures the client credentials Provider.
type ProviderOption func(*clientCredentialsProviderConfig)

// WithScopes configures the scopes requested from the authorization server.
func WithScopes(scopes ...string) ProviderOption {
	return func(config *clientCredentialsProviderConfig) {
		config.scopes = scopes
	}
}

// WithTransportCredentials configures transport credentials for gRPC connections.
func WithTransportCredentials(creds credentials.TransportCredentials) ProviderOption {
	return func(config *clientCredentialsProviderConfig) {
		config.transportCredentials = creds
	}
}

// NewDiscoveryProvider creates a provider using OAuth2 Authorization Server Metadata discovery (RFC 8414).
func NewDiscoveryProvider(
	ctx context.Context,
	authorizationServerURL, clientID, clientSecret string,
	options ...ProviderOption,
) (*Provider, error) {
	metadata, err := cantonauth.GetAuthorizationServerMetadata(ctx, authorizationServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization server metadata: %w", err)
	}

	return NewProvider(ctx, metadata.TokenEndpoint, clientID, clientSecret, options...)
}

// NewProvider creates a provider that fetches tokens using the OAuth2 client credentials flow.
func NewProvider(
	ctx context.Context,
	tokenURL, clientID, clientSecret string,
	options ...ProviderOption,
) (*Provider, error) {
	cfg := defaultClientCredentialsProviderConfig()
	for _, option := range options {
		option(cfg)
	}

	if tokenURL == "" {
		return nil, errors.New("tokenURL cannot be empty")
	}
	if clientID == "" {
		return nil, errors.New("clientID cannot be empty")
	}
	if clientSecret == "" {
		return nil, errors.New("clientSecret cannot be empty")
	}

	oauthCfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
		Scopes:       cfg.scopes,
	}

	refreshCtx := context.WithoutCancel(ctx)
	tokenSource := oauthCfg.TokenSource(refreshCtx)

	return &Provider{
		tokenSource:          oauth.TokenSource{TokenSource: tokenSource},
		transportCredentials: cfg.transportCredentials,
	}, nil
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
