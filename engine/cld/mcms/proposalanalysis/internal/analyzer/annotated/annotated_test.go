package annotated

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with name, type, and value", func(t *testing.T) {
		t.Parallel()

		ann := annotation.New("cld.severity", "enum", "warning")

		assert.Equal(t, "cld.severity", ann.Name())
		assert.Equal(t, "enum", ann.Type())
		assert.Equal(t, "warning", ann.Value())
		assert.Empty(t, ann.AnalyzerID())
	})

	t.Run("supports arbitrary value types", func(t *testing.T) {
		t.Parallel()

		ann := annotation.New("count", "int", 42)
		assert.Equal(t, 42, ann.Value())

		ann = annotation.New("flag", "boolean", true)
		assert.Equal(t, true, ann.Value())
	})

	t.Run("supports nil value", func(t *testing.T) {
		t.Parallel()

		ann := annotation.New("empty", "string", nil)
		assert.Nil(t, ann.Value())
	})
}

func TestNewWithAnalyzer(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with analyzer tracking", func(t *testing.T) {
		t.Parallel()

		ann := annotation.NewWithAnalyzer("cld.risk", "enum", "high", "my-analyzer")

		assert.Equal(t, "cld.risk", ann.Name())
		assert.Equal(t, "enum", ann.Type())
		assert.Equal(t, "high", ann.Value())
		assert.Equal(t, "my-analyzer", ann.AnalyzerID())

		a := &BaseAnnotated{}
		a.AddAnnotations(ann)

		result := a.Filter(ByAnalyzer("my-analyzer"))
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
			annotation.New("name1", "type1", "value1"),
			annotation.New("name2", "type2", "value2"),
		)

		require.Len(t, a.Annotations(), 2)
		assert.Equal(t, "name1", a.Annotations()[0].Name())
		assert.Equal(t, "name2", a.Annotations()[1].Name())
	})

	t.Run("add annotations incrementally", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(annotation.New("first", "string", "a"))
		a.AddAnnotations(annotation.New("second", "string", "b"))

		require.Len(t, a.Annotations(), 2)
		assert.Equal(t, "first", a.Annotations()[0].Name())
		assert.Equal(t, "second", a.Annotations()[1].Name())
	})

	t.Run("Filter by name works correctly", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.New("severity", "enum", "warning"),
			annotation.New("risk", "enum", "high"),
			annotation.New("severity", "enum", "error"),
		)

		result := a.Filter(ByName("severity"))
		require.Len(t, result, 2)
		assert.Equal(t, "warning", result[0].Value())
		assert.Equal(t, "error", result[1].Value())

		result = a.Filter(ByName("risk"))
		require.Len(t, result, 1)

		result = a.Filter(ByName("nonexistent"))
		assert.Empty(t, result)
	})

	t.Run("Filter by type works correctly", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.New("severity", "enum", "warning"),
			annotation.New("note", "string", "hello"),
			annotation.New("risk", "enum", "high"),
		)

		result := a.Filter(ByType("enum"))
		require.Len(t, result, 2)

		result = a.Filter(ByType("string"))
		require.Len(t, result, 1)

		result = a.Filter(ByType("nonexistent"))
		assert.Empty(t, result)
	})

	t.Run("Filter by analyzer works correctly", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.NewWithAnalyzer("severity", "enum", "warning", "analyzer-a"),
			annotation.NewWithAnalyzer("risk", "enum", "high", "analyzer-b"),
			annotation.NewWithAnalyzer("note", "string", "details", "analyzer-a"),
			annotation.New("plain", "string", "no-analyzer"),
		)

		result := a.Filter(ByAnalyzer("analyzer-a"))
		require.Len(t, result, 2)
		assert.Equal(t, "severity", result[0].Name())
		assert.Equal(t, "note", result[1].Name())

		result = a.Filter(ByAnalyzer("analyzer-b"))
		require.Len(t, result, 1)

		result = a.Filter(ByAnalyzer(""))
		require.Len(t, result, 1)
		assert.Equal(t, "plain", result[0].Name())

		result = a.Filter(ByAnalyzer("nonexistent"))
		assert.Empty(t, result)
	})

	t.Run("Filter with custom predicate", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.New("severity", "enum", "warning"),
			annotation.New("note", "string", "hello"),
			annotation.New("risk", "enum", "high"),
			annotation.New("flag", "boolean", true),
		)

		result := a.Filter(func(ann annotation.Annotation) bool {
			s, ok := ann.Value().(string)
			return ok && len(s) > 0 && s[0] == 'h'
		})
		require.Len(t, result, 2)
		assert.Equal(t, "hello", result[0].Value())
		assert.Equal(t, "high", result[1].Value())

		result = a.Filter(func(ann annotation.Annotation) bool { return false })
		assert.Empty(t, result)
	})

	t.Run("Filter with predicate builders", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.NewWithAnalyzer("severity", "enum", "warning", "analyzer-a"),
			annotation.New("note", "string", "hello"),
			annotation.NewWithAnalyzer("risk", "enum", "high", "analyzer-a"),
		)

		result := a.Filter(ByName("severity"))
		require.Len(t, result, 1)
		assert.Equal(t, "warning", result[0].Value())

		result = a.Filter(ByType("enum"))
		require.Len(t, result, 2)

		result = a.Filter(ByAnalyzer("analyzer-a"))
		require.Len(t, result, 2)
	})

	t.Run("Filter supports mixing predicates", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.NewWithAnalyzer("severity", "enum", "warning", "analyzer-a"),
			annotation.NewWithAnalyzer("severity", "string", "text", "analyzer-a"),
			annotation.NewWithAnalyzer("risk", "enum", "high", "analyzer-a"),
			annotation.NewWithAnalyzer("severity", "enum", "error", "analyzer-b"),
		)

		result := a.Filter(
			ByName("severity"),
			ByType("enum"),
			ByAnalyzer("analyzer-a"),
		)
		require.Len(t, result, 1)
		assert.Equal(t, "warning", result[0].Value())
	})

	t.Run("Filter with no predicates returns all annotations", func(t *testing.T) {
		t.Parallel()

		a := &BaseAnnotated{}
		a.AddAnnotations(
			annotation.New("severity", "enum", "warning"),
			annotation.New("risk", "enum", "high"),
		)

		result := a.Filter()
		require.Len(t, result, 2)
	})
}
