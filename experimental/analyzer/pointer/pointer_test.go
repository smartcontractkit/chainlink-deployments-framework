package pointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDerefOrEmpty(t *testing.T) {
	t.Parallel()

	t.Run("int", func(t *testing.T) { //nolint:paralleltest
		intVar := 1
		intVarPtr := &intVar
		assert.Equal(t, 1, DerefOrEmpty(intVarPtr))
		intVarPtr = nil
		assert.Equal(t, 0, DerefOrEmpty(intVarPtr))
	})

	t.Run("string", func(t *testing.T) { //nolint:paralleltest
		stringVar := "want"
		stringVarPtr := &stringVar
		assert.Equal(t, "want", DerefOrEmpty(stringVarPtr))
		stringVarPtr = nil
		assert.Equal(t, "", DerefOrEmpty(stringVarPtr)) //nolint:testifylint
	})

	t.Run("slice", func(t *testing.T) { //nolint:paralleltest
		slicePointer := &[]int{1, 2, 3}
		assert.Equal(t, []int{1, 2, 3}, DerefOrEmpty(slicePointer))
		slicePointer = nil
		assert.Equal(t, []int(nil), DerefOrEmpty(slicePointer))
	})

	t.Run("struct", func(t *testing.T) { //nolint:paralleltest
		type TestStruct struct{ i int }
		structPointer := &TestStruct{1}
		assert.Equal(t, TestStruct{1}, DerefOrEmpty(structPointer))
		structPointer = nil
		assert.Equal(t, TestStruct{}, DerefOrEmpty(structPointer))
	})
}
