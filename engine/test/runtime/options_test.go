package runtime

import (
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/environment"
)

func TestWithEnvOpts(t *testing.T) {
	t.Parallel()

	cfg := &runtimeConfig{}
	opts := []environment.LoadOpt{
		environment.WithLogger(logger.Test(t)),
	}

	WithEnvOpts(opts...)(cfg)

	require.Equal(t, opts, cfg.envOpts)
}
