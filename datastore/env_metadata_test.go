package datastore

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestEnvMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := EnvMetadata{
		Metadata: TestMetadata{Data: "test-value"},
	}

	cloned, err := original.Clone()
	require.NoError(t, err)

	typed, err := As[TestMetadata](cloned.Metadata)

	require.Equal(t, original.Metadata, typed)
	require.NotSame(t, &original.Metadata, &typed) // Ensure Metadata is a deep copy
}
