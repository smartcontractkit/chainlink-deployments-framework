package datastore

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestEnvMetadata_Clone(t *testing.T) {
	t.Parallel()

	var (
		metaOne  = DefaultMetadata{Data: "test-value-one"}
		metaTwo  = DefaultMetadata{Data: "test-value-two"}
		original = EnvMetadata{
			Metadata: metaOne,
		}
	)

	cloned, err := original.Clone()
	require.NoError(t, err, "Clone should not return an error")

	concrete, err := As[DefaultMetadata](cloned.Metadata)
	require.NoError(t, err, "As should not return an error for DefaultMetadata")
	require.Equal(t, metaOne, concrete)

	original.Metadata = metaTwo

	concreteTwo, err := As[DefaultMetadata](original.Metadata)
	require.NoError(t, err, "As should not return an error for DefaultMetadata after modification")
	require.NotEqual(t, concrete, concreteTwo, "Cloned metadata should not be equal to modified original metadata")
}
