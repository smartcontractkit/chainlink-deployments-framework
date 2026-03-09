package evm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSolidityContractMetadata_SourceCode(t *testing.T) {
	t.Parallel()

	t.Run("valid metadata", func(t *testing.T) {
		t.Parallel()

		meta := SolidityContractMetadata{
			Version:  "0.8.19",
			Language: "Solidity",
			Settings: map[string]any{"optimizer": map[string]any{"enabled": true}},
			Sources:  map[string]any{"Contract.sol": map[string]any{"content": "pragma solidity 0.8.19;"}},
			Name:     "MyContract",
		}

		sourceCode, err := meta.SourceCode()
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(sourceCode), &decoded))
		require.Equal(t, "Solidity", decoded["language"])
		require.Equal(t, "0.8.19", meta.Version)
	})

	t.Run("empty metadata produces valid JSON", func(t *testing.T) {
		t.Parallel()

		meta := SolidityContractMetadata{}
		sourceCode, err := meta.SourceCode()
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal([]byte(sourceCode), &decoded))
		require.Empty(t, decoded["language"])
	})
}
