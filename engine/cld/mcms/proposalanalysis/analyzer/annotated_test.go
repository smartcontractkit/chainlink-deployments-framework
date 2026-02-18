package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with name, type, and value", func(t *testing.T) {
		t.Parallel()

		ann := NewAnnotation("cld.severity", "enum", "warning")

		assert.Equal(t, "cld.severity", ann.Name())
		assert.Equal(t, "enum", ann.Type())
		assert.Equal(t, "warning", ann.Value())
		assert.Empty(t, ann.AnalyzerID())
	})

	t.Run("supports arbitrary value types", func(t *testing.T) {
		t.Parallel()

		ann := NewAnnotation("count", "int", 42)
		assert.Equal(t, 42, ann.Value())

		ann = NewAnnotation("flag", "boolean", true)
		assert.Equal(t, true, ann.Value())
	})

	t.Run("supports nil value", func(t *testing.T) {
		t.Parallel()

		ann := NewAnnotation("empty", "string", nil)
		assert.Nil(t, ann.Value())
	})
}

func TestNewAnnotationWithAnalyzer(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with analyzer tracking", func(t *testing.T) {
		t.Parallel()

		ann := NewAnnotationWithAnalyzer("cld.risk", "enum", "high", "my-analyzer")

		assert.Equal(t, "cld.risk", ann.Name())
		assert.Equal(t, "enum", ann.Type())
		assert.Equal(t, "high", ann.Value())
		assert.Equal(t, "my-analyzer", ann.AnalyzerID())

		a := &BaseAnnotated{}
		a.AddAnnotations(ann)

		result := a.GetAnnotationsByAnalyzer("my-analyzer")
		require.Len(t, result, 1)
		assert.Equal(t, "cld.risk", result[0].Name())
	})
}

func TestBaseAnnotated(t *testing.T) {
	t.Parallel()

	t.Run("empty annotations", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		assert.Empty(t, a.Annotations())
	})

	t.Run("add multiple annotations at once", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			NewAnnotation("name1", "type1", "value1"),
			NewAnnotation("name2", "type2", "value2"),
		)

		require.Len(t, a.Annotations(), 2)
		assert.Equal(t, "name1", a.Annotations()[0].Name())
		assert.Equal(t, "name2", a.Annotations()[1].Name())
	})

	t.Run("add annotations incrementally", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(NewAnnotation("first", "string", "a"))
		a.AddAnnotations(NewAnnotation("second", "string", "b"))

		require.Len(t, a.Annotations(), 2)
		assert.Equal(t, "first", a.Annotations()[0].Name())
		assert.Equal(t, "second", a.Annotations()[1].Name())
	})

	t.Run("GetAnnotationsByName filters correctly", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			NewAnnotation("severity", "enum", "warning"),
			NewAnnotation("risk", "enum", "high"),
			NewAnnotation("severity", "enum", "error"),
		)

		result := a.GetAnnotationsByName("severity")
		require.Len(t, result, 2)
		assert.Equal(t, "warning", result[0].Value())
		assert.Equal(t, "error", result[1].Value())

		result = a.GetAnnotationsByName("risk")
		require.Len(t, result, 1)

		result = a.GetAnnotationsByName("nonexistent")
		assert.Empty(t, result)
	})

	t.Run("GetAnnotationsByType filters correctly", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			NewAnnotation("severity", "enum", "warning"),
			NewAnnotation("note", "string", "hello"),
			NewAnnotation("risk", "enum", "high"),
		)

		result := a.GetAnnotationsByType("enum")
		require.Len(t, result, 2)

		result = a.GetAnnotationsByType("string")
		require.Len(t, result, 1)

		result = a.GetAnnotationsByType("nonexistent")
		assert.Empty(t, result)
	})

	t.Run("GetAnnotationsByAnalyzer filters by analyzer ID", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			NewAnnotationWithAnalyzer("severity", "enum", "warning", "analyzer-a"),
			NewAnnotationWithAnalyzer("risk", "enum", "high", "analyzer-b"),
			NewAnnotationWithAnalyzer("note", "string", "details", "analyzer-a"),
			NewAnnotation("plain", "string", "no-analyzer"),
		)

		result := a.GetAnnotationsByAnalyzer("analyzer-a")
		require.Len(t, result, 2)
		assert.Equal(t, "severity", result[0].Name())
		assert.Equal(t, "note", result[1].Name())

		result = a.GetAnnotationsByAnalyzer("analyzer-b")
		require.Len(t, result, 1)

		result = a.GetAnnotationsByAnalyzer("")
		require.Len(t, result, 1)
		assert.Equal(t, "plain", result[0].Name())

		result = a.GetAnnotationsByAnalyzer("nonexistent")
		assert.Empty(t, result)
	})

	t.Run("Filter with custom predicate", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			NewAnnotation("severity", "enum", "warning"),
			NewAnnotation("note", "string", "hello"),
			NewAnnotation("risk", "enum", "high"),
			NewAnnotation("flag", "boolean", true),
		)

		result := a.Filter(func(ann Annotation) bool {
			s, ok := ann.Value().(string)
			return ok && len(s) > 0 && s[0] == 'h'
		})
		require.Len(t, result, 2)
		assert.Equal(t, "hello", result[0].Value())
		assert.Equal(t, "high", result[1].Value())

		result = a.Filter(func(ann Annotation) bool { return false })
		assert.Empty(t, result)
	})

	t.Run("Filter with predicate builders", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			NewAnnotationWithAnalyzer("severity", "enum", "warning", "analyzer-a"),
			NewAnnotation("note", "string", "hello"),
			NewAnnotationWithAnalyzer("risk", "enum", "high", "analyzer-a"),
		)

		result := a.Filter(ByName("severity"))
		require.Len(t, result, 1)
		assert.Equal(t, "warning", result[0].Value())

		result = a.Filter(ByType("enum"))
		require.Len(t, result, 2)

		result = a.Filter(ByAnalyzer("analyzer-a"))
		require.Len(t, result, 2)
	})
}
