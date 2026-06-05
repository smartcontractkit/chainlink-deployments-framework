package authentication

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthorizationServerMetadata represents a subset of the metadata provided by an OAuth 2.0 Authorization Server.
// See RFC 8414, Section 2 for the full specification:
// https://datatracker.ietf.org/doc/html/rfc8414#section-2
type AuthorizationServerMetadata struct {
	// Issuer is the authorization server's issuer identifier, which is a URL.
	Issuer string `json:"issuer"`
	// AuthorizationEndpoint is the URL of the authorization server's authorization endpoint.
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	// TokenEndpoint is the URL of the authorization server's token endpoint.
	TokenEndpoint string `json:"token_endpoint"`
	// CodeChallengeMethodsSupported lists PKCE code challenge methods supported by this authorization server.
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// GetAuthorizationServerMetadata retrieves OAuth 2.0 authorization server metadata from the well-known endpoint.
func GetAuthorizationServerMetadata(ctx context.Context, authorizationServerURL string) (*AuthorizationServerMetadata, error) {
	authorizationServerURL = strings.TrimSuffix(authorizationServerURL, "/")
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		authorizationServerURL+"/.well-known/oauth-authorization-server",
		nil,
	)
	if err != nil {
		return nil, err
	}

	client := http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	metadata := &AuthorizationServerMetadata{}
	if err := json.Unmarshal(body, metadata); err != nil {
		return nil, fmt.Errorf("unmarshalling response body: %w", err)
	}

	if metadata.Issuer != authorizationServerURL {
		return nil, fmt.Errorf("metadata: unexpected issuer: %s", metadata.Issuer)
	}

	return metadata, nil
}
