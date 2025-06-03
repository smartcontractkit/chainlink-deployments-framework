package datastore

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestEnvMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := EnvMetadata[DefaultMetadata]{
		Metadata: DefaultMetadata{Data: "test-value"},
	}

	cloned, err := original.Clone()
	require.NoError(t, err, "Clone should not return an error")

	require.Equal(t, original.Metadata, cloned.Metadata)
	require.NotSame(t, &original.Metadata, &cloned.Metadata) // Ensure Metadata is a deep copy
}
