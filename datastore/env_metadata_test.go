package datastore

import (
	"testing"
	"time"

	require "github.com/stretchr/testify/require"
)

func TestEnvMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := EnvMetadata{
		Metadata: TestEnvMetadata{
			EnvName:   "test-env",
			EnvID:     "env-123",
			CreatedAt: time.Date(2024, 5, 28, 12, 0, 0, 0, time.UTC),
		},
	}

	cloned, err := original.Clone()
	require.NoError(t, err)

	typed, err := As[TestEnvMetadata](cloned.Metadata)
	require.NoError(t, err)

	require.Equal(t, original.Metadata, typed)
	require.NotSame(t, &original.Metadata, &typed) // Ensure Metadata is a deep copy
}
