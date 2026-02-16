package authentication

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Provider provides authentication credentials for connecting to a Canton participant's API endpoints.
// The Provider acts as both a raw token-source for HTTP API authentication, and a gRPC credentials provider for gRPC endpoint authentication.
//
// Implementations of this interface can implement different means of fetching and refreshing authentication tokens,
// as well as enforcing different levels of transport security. The specific implementation of the Provider
// should be chosen based on the environment being connected to (e.g. LocalNet vs. production, i.e. CI/OIDC).
type Provider interface {
	// TokenSource returns an oauth2.TokenSource that can be used to retrieve access tokens for authenticating with the participant's API endpoints.
	TokenSource() oauth2.TokenSource
	// TransportCredentials returns gRPC transport credentials to be used when connecting to the participant's RPC endpoints.
	TransportCredentials() credentials.TransportCredentials
	// PerRPCCredentials returns gRPC per-RPC credentials to be used when connecting to the participant's gRPC endpoints.
	PerRPCCredentials() credentials.PerRPCCredentials
}

// InsecureStaticProvider is an insecure implementation of Provider that always
// returns the same static access token and does not provide/enforce transport security.
// This provider is only suitable for testing against LocalNet or other non-production environments.
type InsecureStaticProvider struct {
	AccessToken string
}

var _ Provider = InsecureStaticProvider{}

func (i InsecureStaticProvider) TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: i.AccessToken,
	})
}

func (i InsecureStaticProvider) TransportCredentials() credentials.TransportCredentials {
	return insecure.NewCredentials()
}

func (i InsecureStaticProvider) PerRPCCredentials() credentials.PerRPCCredentials {
	return insecureTokenSource{
		TokenSource: i.TokenSource(),
	}
}

// insecureTokenSource is an insecure OAuth2 PerRPCCredentials implementation that retrieves tokens from an underlying oauth2.TokenSource.
// It does not enforce transport security, making it only suitable for testing against LocalNet.
type insecureTokenSource struct {
	oauth2.TokenSource
}

var _ credentials.PerRPCCredentials = insecureTokenSource{}

func (ts insecureTokenSource) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	token, err := ts.Token()
	if err != nil {
		return nil, err
	}
	if token == nil {
		//nolint:nilnil // nothing to do here, just retuning no metadata and no error
		return nil, nil
	}

	return map[string]string{
		"authorization": "Bearer " + token.AccessToken,
	}, nil
}

func (ts insecureTokenSource) RequireTransportSecurity() bool {
	return false
}
