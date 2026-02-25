package authentication

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"
)

func TestInsecureStaticProvider(t *testing.T) {
	t.Parallel()

	testToken := "test-token-123"
	provider := NewInsecureStaticProvider(testToken)

	// Test that TokenSource returns the correct token
	token, err := provider.TokenSource().Token()
	require.NoError(t, err)
	assert.Equal(t, testToken, token.AccessToken)

	// Test that the provider returns the correct transport credentials
	transportCredentials := provider.TransportCredentials()
	assert.Equal(t, insecure.NewCredentials(), transportCredentials)

	// Test that the provider returns the correct per RPC credentials
	perRPCCredentials := provider.PerRPCCredentials()
	require.NotNil(t, perRPCCredentials)

	// Test that the RPC credentials return the correct metadata
	metadata, err := perRPCCredentials.GetRequestMetadata(t.Context())
	require.NoError(t, err)
	header, ok := metadata["authorization"]
	require.True(t, ok, "PerRPCCredentials didn't return authorization header")
	assert.Equal(t, "Bearer "+testToken, header)

	// Test that the RPC credentials do not require transport security
	requireTransportSecurity := perRPCCredentials.RequireTransportSecurity()
	assert.False(t, requireTransportSecurity, "PerRPCCredentials must not require transport security")
}

func TestStaticProvider(t *testing.T) {
	t.Parallel()

	testToken := "test-token-123"
	provider := NewStaticProvider(testToken)

	token, err := provider.TokenSource().Token()
	require.NoError(t, err)
	assert.Equal(t, testToken, token.AccessToken)

	transportCredentials := provider.TransportCredentials()
	require.NotNil(t, transportCredentials)
	assert.NotEqual(t, insecure.NewCredentials(), transportCredentials)

	perRPCCredentials := provider.PerRPCCredentials()
	require.NotNil(t, perRPCCredentials)

	metadata, err := perRPCCredentials.GetRequestMetadata(t.Context())
	require.NoError(t, err)
	header, ok := metadata["authorization"]
	require.True(t, ok, "PerRPCCredentials didn't return authorization header")
	assert.Equal(t, "Bearer "+testToken, header)

	requireTransportSecurity := perRPCCredentials.RequireTransportSecurity()
	assert.True(t, requireTransportSecurity, "PerRPCCredentials must require transport security")
}
