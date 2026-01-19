package canton

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticJWTProvider(t *testing.T) {
	t.Parallel()

	provider := NewStaticJWTProvider("test-jwt-token")

	assert.Equal(t, "StaticJWTProvider", provider.Name())

	token, err := provider.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "test-jwt-token", token)
}
