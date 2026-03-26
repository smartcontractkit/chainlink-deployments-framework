package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeverityAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("uses correct name and type for all levels", func(t *testing.T) {
		t.Parallel()

		for _, level := range []Severity{SeverityError, SeverityWarning, SeverityInfo, SeverityDebug} {
			ann := SeverityAnnotation(level)
			assert.Equal(t, AnnotationSeverityName, ann.Name())
			assert.Equal(t, AnnotationSeverityType, ann.Type())
			assert.Equal(t, string(level), ann.Value())
		}
	})
}

func TestRiskAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("uses correct name and type for all levels", func(t *testing.T) {
		t.Parallel()

		for _, level := range []Risk{RiskHigh, RiskMedium, RiskLow} {
			ann := RiskAnnotation(level)
			assert.Equal(t, AnnotationRiskName, ann.Name())
			assert.Equal(t, AnnotationRiskType, ann.Type())
			assert.Equal(t, string(level), ann.Value())
		}
	})
}
