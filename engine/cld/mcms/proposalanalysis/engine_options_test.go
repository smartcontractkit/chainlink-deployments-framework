package proposalanalysis

import (
	"testing"
	"time"

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
		)

		assert.NotNil(t, cfg.GetLogger())
	})

	t.Run("WithAnalyzerTimeout option sets timeout", func(t *testing.T) {
		customTimeout := 2 * time.Minute
		cfg := ApplyEngineOptions(WithAnalyzerTimeout(customTimeout))

		assert.Equal(t, customTimeout, cfg.GetAnalyzerTimeout())
	})

	t.Run("GetAnalyzerTimeout returns default when not set", func(t *testing.T) {
		cfg := ApplyEngineOptions()

		timeout := cfg.GetAnalyzerTimeout()
		assert.Equal(t, DefaultAnalyzerTimeout, timeout)
		assert.Equal(t, 5*time.Minute, timeout)
	})

	t.Run("all options can be combined including timeout", func(t *testing.T) {
		lggr := logger.Test(t)
		customTimeout := 1 * time.Minute
		cfg := ApplyEngineOptions(
			WithLogger(lggr),
			WithAnalyzerTimeout(customTimeout),
		)

		assert.NotNil(t, cfg.GetLogger())
		assert.Equal(t, customTimeout, cfg.GetAnalyzerTimeout())
	})
}
