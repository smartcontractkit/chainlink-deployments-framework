package proposalanalysis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEngineOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithAnalyzerTimeout option sets timeout", func(t *testing.T) {
		t.Parallel()

		customTimeout := 2 * time.Minute
		cfg := ApplyEngineOptions(WithAnalyzerTimeout(customTimeout))

		assert.Equal(t, customTimeout, cfg.GetAnalyzerTimeout())
	})

	t.Run("GetAnalyzerTimeout returns default when not set", func(t *testing.T) {
		t.Parallel()

		cfg := ApplyEngineOptions()

		timeout := cfg.GetAnalyzerTimeout()
		assert.Equal(t, DefaultAnalyzerTimeout, timeout)
	})
}
