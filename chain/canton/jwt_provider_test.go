package canton

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticJWTProvider(t *testing.T) {
	t.Parallel()

	provider := NewStaticJWTProvider("test-jwt-token")

	assert.Equal(t, "StaticJWTProvider", provider.Name())

	token, err := provider.Token(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, "test-jwt-token", token)
}
