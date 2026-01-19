package canton

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticJWTProvider(t *testing.T) {
	t.Parallel()

	provider := NewStaticJWTProvider("test-jwt-token")

	assert.Equal(t, provider.Name(), "StaticJWTProvider")

	token, err := provider.Token(nil)
	assert.NoError(t, err)
	assert.Equal(t, token, "test-jwt-token")
}
