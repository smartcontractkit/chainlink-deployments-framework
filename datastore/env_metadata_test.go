package datastore

import (
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestEnvMetadata_Clone(t *testing.T) {
	t.Parallel()

	var (
		metaOne  = testMetadata{Field: "test-value-one", ChainSelector: 0}
		metaTwo  = testMetadata{Field: "test-value-two", ChainSelector: 0}
		original = EnvMetadata{
			Metadata: metaOne,
		}
	)

	cloned, err := original.Clone()
	require.NoError(t, err, "Clone should not return an error")

	concrete, err := As[testMetadata](cloned.Metadata)
	require.NoError(t, err, "As should not return an error for CustomMetadata")
	require.Equal(t, metaOne, concrete)

	original.Metadata = metaTwo

	concreteTwo, err := As[testMetadata](original.Metadata)
	require.NoError(t, err, "As should not return an error for CustomMetadata after modification")
	require.NotEqual(t, concrete, concreteTwo, "Cloned metadata should not be equal to modified original metadata")
}
