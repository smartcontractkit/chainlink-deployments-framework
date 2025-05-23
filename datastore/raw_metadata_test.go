package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRawMetadata_Clone(t *testing.T) {
	original := RawMetadata{raw: json.RawMessage(`{"foo": "bar"}`)}
	cloned := original.Clone()

	require.Equal(t, original, cloned)
	// Ensure it's a deep copy
	clonedRaw := cloned.(RawMetadata)
	clonedRaw.raw[0] = '}'
	require.NotEqual(t, original.raw, clonedRaw.raw)
}

func TestRawMetadata_MarshalJSON(t *testing.T) {
	meta := RawMetadata{raw: json.RawMessage(`{"foo": "bar"}`)}
	b, err := meta.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `{"foo": "bar"}`, string(b))
}

func TestRawMetadata_Raw(t *testing.T) {
	msg := json.RawMessage(`{"foo": "bar"}`)
	meta := RawMetadata{raw: msg}
	require.Equal(t, msg, meta.Raw())
}
