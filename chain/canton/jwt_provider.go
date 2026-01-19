package canton

import (
	"context"
)

type JWTProvider interface {
	Name() string
	Token(ctx context.Context) (string, error)
}

// StaticJWTProvider is a simple implementation of JWTProvider that always returns the same JWT.
type StaticJWTProvider struct {
	jwt string
}

func NewStaticJWTProvider(jwt string) *StaticJWTProvider {
	return &StaticJWTProvider{jwt: jwt}
}

func (s StaticJWTProvider) Name() string {
	return "StaticJWTProvider"
}

func (s StaticJWTProvider) Token(_ context.Context) (string, error) {
	return s.jwt, nil
}
