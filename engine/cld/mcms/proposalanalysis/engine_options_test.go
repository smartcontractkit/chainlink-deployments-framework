package proposalanalysis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestEngineOptions(t *testing.T) {
	t.Run("WithLogger option sets logger", func(t *testing.T) {
		lggr := logger.Test(t)
		cfg := ApplyEngineOptions(WithLogger(lggr))

		assert.NotNil(t, cfg.GetLogger())
	})

	t.Run("GetLogger returns nop logger when not set", func(t *testing.T) {
		cfg := ApplyEngineOptions()

		lggr := cfg.GetLogger()
		assert.NotNil(t, lggr)
		// Verify it's a nop logger by checking it doesn't panic when called
		lggr.Info("test message")
		lggr.Errorw("test error", "key", "value")
	})

	t.Run("multiple options can be combined", func(t *testing.T) {
		lggr := logger.Test(t)
		cfg := ApplyEngineOptions(
			WithLogger(lggr),
			WithEVMRegistry(nil),
			WithSolanaRegistry(nil),
		)

		assert.NotNil(t, cfg.GetLogger())
		assert.Nil(t, cfg.GetEVMRegistry())
		assert.Nil(t, cfg.GetSolanaRegistry())
	})
}
