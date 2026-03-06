package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewEnvironment_DefaultDomainConfigGetter(t *testing.T) {
	t.Parallel()

	env := NewNoopEnvironment(t)
	require.NotNil(t, env.DomainConfigGetter)
}
