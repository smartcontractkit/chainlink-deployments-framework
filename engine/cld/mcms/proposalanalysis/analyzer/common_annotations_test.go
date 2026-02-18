package analyzer

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

func TestValueTypeAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("uses correct name and type", func(t *testing.T) {
		t.Parallel()

		for _, vtype := range []string{"ethereum.address", "ethereum.uint256", "hex", "truncate:20"} {
			ann := ValueTypeAnnotation(vtype)
			assert.Equal(t, AnnotationValueTypeName, ann.Name())
			assert.Equal(t, AnnotationValueTypeType, ann.Type())
			assert.Equal(t, vtype, ann.Value())
		}
	})
}

func TestAnnotationConstructors_WorkWithBaseAnnotated(t *testing.T) {
	t.Parallel()

	a := &BaseAnnotated{}
	a.AddAnnotations(
		SeverityAnnotation(SeverityWarning),
		RiskAnnotation(RiskHigh),
		ValueTypeAnnotation("ethereum.address"),
	)

	assert.Len(t, a.GetAnnotationsByName(AnnotationSeverityName), 1)
	assert.Len(t, a.GetAnnotationsByName(AnnotationRiskName), 1)
	assert.Len(t, a.GetAnnotationsByName(AnnotationValueTypeName), 1)
}
