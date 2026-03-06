package deployment

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // Mutates process-global env vars via os.Setenv.
func TestEnvVarDomainConfigGetter_Get(t *testing.T) {
	const (
		key   = "CLDF_TEST_DOMAIN_CONFIG_GETTER_KEY"
		value = "domain-config-value"
	)

	prevValue, hadPrevValue := os.LookupEnv(key)
	require.NoError(t, os.Setenv(key, value))
	t.Cleanup(func() {
		if hadPrevValue {
			_ = os.Setenv(key, prevValue)
			return
		}
		_ = os.Unsetenv(key)
	})

	var getter DomainConfigGetter = envVarDomainConfigGetter{}
	got, found := getter.Get(key)
	require.True(t, found)
	require.Equal(t, value, got)
}
