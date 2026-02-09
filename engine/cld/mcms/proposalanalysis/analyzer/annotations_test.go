package analyzer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnnotations(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("NewAnnotation", func(t *testing.T) {
		ann := NewAnnotation("test", "INFO", "value")
		assert.Equal(t, "test", ann.Name())
		assert.Equal(t, "INFO", ann.Type())
		assert.Equal(t, "value", ann.Value())
	})

	t.Run("NewAnnotationWithAnalyzer", func(t *testing.T) {
		ann := NewAnnotationWithAnalyzer("test", "WARN", "warning", "analyzer-1")
		assert.Equal(t, "test", ann.Name())
		assert.Equal(t, "WARN", ann.Type())
		assert.Equal(t, "warning", ann.Value())
	})

	t.Run("AddAnnotations", func(t *testing.T) {
		a := &annotated{}
		ann1 := NewAnnotation("ann1", "INFO", "v1")
		ann2 := NewAnnotation("ann2", "WARN", "v2")

		a.AddAnnotations(ann1)
		assert.Len(t, a.Annotations(), 1)

		a.AddAnnotations(ann2)
		assert.Len(t, a.Annotations(), 2)
	})

	t.Run("GetAnnotationsByName", func(t *testing.T) {
		a := &annotated{}
		ann1 := NewAnnotation("gas-estimate", "INFO", 100)
		ann2 := NewAnnotation("security-check", "WARN", "vulnerable")
		ann3 := NewAnnotation("gas-estimate", "INFO", 200)

		a.AddAnnotations(ann1, ann2, ann3)

		results := a.GetAnnotationsByName("gas-estimate")
		assert.Len(t, results, 2)
		assert.Equal(t, "gas-estimate", results[0].Name())
		assert.Equal(t, "gas-estimate", results[1].Name())

		results = a.GetAnnotationsByName("security-check")
		assert.Len(t, results, 1)
		assert.Equal(t, "security-check", results[0].Name())

		results = a.GetAnnotationsByName("nonexistent")
		assert.Len(t, results, 0)
	})

	t.Run("GetAnnotationsByType", func(t *testing.T) {
		a := &annotated{}
		ann1 := NewAnnotation("ann1", "INFO", "v1")
		ann2 := NewAnnotation("ann2", "WARN", "v2")
		ann3 := NewAnnotation("ann3", "INFO", "v3")
		ann4 := NewAnnotation("ann4", "ERROR", "v4")

		a.AddAnnotations(ann1, ann2, ann3, ann4)

		results := a.GetAnnotationsByType("INFO")
		assert.Len(t, results, 2)

		results = a.GetAnnotationsByType("WARN")
		assert.Len(t, results, 1)

		results = a.GetAnnotationsByType("ERROR")
		assert.Len(t, results, 1)

		results = a.GetAnnotationsByType("DIFF")
		assert.Len(t, results, 0)
	})

	t.Run("GetAnnotationsByAnalyzer", func(t *testing.T) {
		a := &annotated{}
		ann1 := NewAnnotationWithAnalyzer("ann1", "INFO", "v1", "analyzer-1")
		ann2 := NewAnnotationWithAnalyzer("ann2", "WARN", "v2", "analyzer-2")
		ann3 := NewAnnotationWithAnalyzer("ann3", "INFO", "v3", "analyzer-1")
		ann4 := NewAnnotation("ann4", "ERROR", "v4") // No analyzer ID

		a.AddAnnotations(ann1, ann2, ann3, ann4)

		results := a.GetAnnotationsByAnalyzer("analyzer-1")
		assert.Len(t, results, 2)

		results = a.GetAnnotationsByAnalyzer("analyzer-2")
		assert.Len(t, results, 1)

		results = a.GetAnnotationsByAnalyzer("analyzer-3")
		assert.Len(t, results, 0)
	})

	t.Run("Combined queries", func(t *testing.T) {
		a := &annotated{}
		ann1 := NewAnnotationWithAnalyzer("gas-estimate", "INFO", 100, "gas-analyzer")
		ann2 := NewAnnotationWithAnalyzer("gas-estimate", "WARN", 500, "gas-analyzer")
		ann3 := NewAnnotationWithAnalyzer("security", "WARN", "issue", "security-analyzer")

		a.AddAnnotations(ann1, ann2, ann3)

		// Get all gas-estimate annotations
		gasAnnotations := a.GetAnnotationsByName("gas-estimate")
		assert.Len(t, gasAnnotations, 2)

		// Get all WARN annotations
		warnings := a.GetAnnotationsByType("WARN")
		assert.Len(t, warnings, 2)

		// Get all annotations from gas-analyzer
		gasAnalyzerAnnotations := a.GetAnnotationsByAnalyzer("gas-analyzer")
		assert.Len(t, gasAnalyzerAnnotations, 2)
	})
}

func TestAnnotationsImplementInterfaces(t *testing.T) {
	t.Run("annotation implements Annotation", func(t *testing.T) {
		var _ analyzer.Annotation = &annotation{}
	})

	t.Run("annotated implements Annotated", func(t *testing.T) {
		var _ analyzer.Annotated = &annotated{}
	})
}
