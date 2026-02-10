package proposalanalysis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestEngineWithLogger(t *testing.T) {
	t.Run("engine accepts logger from options", func(t *testing.T) {
		lggr := logger.Test(t)
		engine := NewAnalyzerEngine(WithLogger(lggr))

		assert.NotNil(t, engine)
		// Verify the logger is set by checking the concrete type
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)
		assert.NotNil(t, concreteEngine.logger)
		assert.Equal(t, "TestEngineWithLogger/engine_accepts_logger_from_options", concreteEngine.logger.Name())
	})

	t.Run("engine uses nop logger when not provided", func(t *testing.T) {
		engine := NewAnalyzerEngine()

		assert.NotNil(t, engine)
		// Verify the logger is set (will be Nop logger)
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)
		assert.NotNil(t, concreteEngine.logger)
	})
}
