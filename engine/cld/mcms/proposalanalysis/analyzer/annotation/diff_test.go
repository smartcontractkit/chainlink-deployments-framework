package annotation

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiffAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("creates annotation with correct name and type", func(t *testing.T) {
		t.Parallel()

		ann := DiffAnnotation("capacity", big.NewInt(0), big.NewInt(1000), "ethereum.uint256")
		assert.Equal(t, AnnotationDiffName, ann.Name())
		assert.Equal(t, AnnotationDiffType, ann.Type())
	})

	t.Run("value is a DiffValue struct", func(t *testing.T) {
		t.Parallel()

		ann := DiffAnnotation("rate", "100", "200", "")
		dv, ok := ann.Value().(DiffValue)
		assert.True(t, ok)
		assert.Equal(t, "rate", dv.Field)
		assert.Equal(t, "100", dv.Old)
		assert.Equal(t, "200", dv.New)
		assert.Empty(t, dv.ValueType)
	})

	t.Run("preserves all fields", func(t *testing.T) {
		t.Parallel()

		old := big.NewInt(100)
		newVal := big.NewInt(200)
		ann := DiffAnnotation("amount", old, newVal, "ethereum.uint256")
		dv := ann.Value().(DiffValue)
		assert.Equal(t, "amount", dv.Field)
		assert.Equal(t, old, dv.Old)
		assert.Equal(t, newVal, dv.New)
		assert.Equal(t, "ethereum.uint256", dv.ValueType)
	})
}
