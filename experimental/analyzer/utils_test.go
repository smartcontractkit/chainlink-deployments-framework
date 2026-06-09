package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFieldValue_Nil(t *testing.T) {
	t.Parallel()

	field := getFieldValue(nil)
	assert.Equal(t, SimpleField{Value: "null"}, field)
}

// TestCantonFieldValue_MapWithNilValue verifies that cantonFieldValue (Canton-scoped) converts
// map[string]any to StructField, which getFieldValue does not do (to avoid affecting other chains).
func TestCantonFieldValue_MapWithNilValue(t *testing.T) {
	t.Parallel()

	field := cantonFieldValue(map[string]any{"optional": nil})
	structField, ok := field.(StructField)
	assert.True(t, ok)
	assert.Len(t, structField.Fields, 1)
	assert.Equal(t, "optional", structField.Fields[0].Name)
	assert.Equal(t, SimpleField{Value: "null"}, structField.Fields[0].Value)
}
