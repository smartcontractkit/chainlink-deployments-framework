package canton

import (
	"context"
)

// JWTProvider defines an interface for obtaining JWT tokens.
// Implementations can provide tokens from various sources such as static configuration,
// OAuth flows, or other authentication mechanisms.
type JWTProvider interface {
	// Name returns the identifier of this JWT provider.
	Name() string

	// Token retrieves a JWT token from the provider's source.
	Token(ctx context.Context) (string, error)
}

// StaticJWTProvider is a simple implementation of JWTProvider that always returns the same JWT.
type StaticJWTProvider struct {
	jwt string
}

// NewStaticJWTProvider creates a new StaticJWTProvider with the given JWT token.
func NewStaticJWTProvider(jwt string) *StaticJWTProvider {
	return &StaticJWTProvider{jwt: jwt}
}

func (s StaticJWTProvider) Name() string {
	return "StaticJWTProvider"
}

func (s StaticJWTProvider) Token(_ context.Context) (string, error) {
	return s.jwt, nil
}
