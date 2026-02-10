package proposalanalysis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

func TestTrackAnnotations(t *testing.T) {
	t.Run("tracks annotations with analyzer ID", func(t *testing.T) {
		// Create some test annotations
		ann1 := analyzer.NewAnnotation("test1", "INFO", "value1")
		ann2 := analyzer.NewAnnotation("test2", "WARN", "value2")
		annotations := types.Annotations{ann1, ann2}

		// Track them with an analyzer ID
		tracked := TrackAnnotations(annotations, "test-analyzer")

		// Verify we got the same number of annotations back
		assert.Len(t, tracked, 2)

		// Verify the annotations maintain their original properties
		assert.Equal(t, "test1", tracked[0].Name())
		assert.Equal(t, "INFO", tracked[0].Type())
		assert.Equal(t, "value1", tracked[0].Value())

		assert.Equal(t, "test2", tracked[1].Name())
		assert.Equal(t, "WARN", tracked[1].Type())
		assert.Equal(t, "value2", tracked[1].Value())

		// Verify they can be retrieved by analyzer ID
		annotated := &analyzer.Annotated{}
		annotated.AddAnnotations(tracked...)

		retrieved := annotated.GetAnnotationsByAnalyzer("test-analyzer")
		assert.Len(t, retrieved, 2)
		assert.Equal(t, "test1", retrieved[0].Name())
		assert.Equal(t, "test2", retrieved[1].Name())
	})

	t.Run("handles empty annotations slice", func(t *testing.T) {
		annotations := types.Annotations{}
		tracked := TrackAnnotations(annotations, "test-analyzer")

		assert.Len(t, tracked, 0)
		assert.NotNil(t, tracked)
	})

	t.Run("preserves annotation values of different types", func(t *testing.T) {
		ann1 := analyzer.NewAnnotation("int-value", "INFO", 42)
		ann2 := analyzer.NewAnnotation("bool-value", "INFO", true)
		ann3 := analyzer.NewAnnotation("slice-value", "INFO", []string{"a", "b", "c"})
		annotations := types.Annotations{ann1, ann2, ann3}

		tracked := TrackAnnotations(annotations, "multi-type-analyzer")

		assert.Equal(t, 42, tracked[0].Value())
		assert.Equal(t, true, tracked[1].Value())
		assert.Equal(t, []string{"a", "b", "c"}, tracked[2].Value())
	})
}
