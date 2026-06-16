package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestZkSyncBytecodeRefUnmarshalYAML(t *testing.T) {
	t.Parallel()

	t.Run("shorthand symbol", func(t *testing.T) {
		t.Parallel()
		var ref ZkSyncBytecodeRef
		require.NoError(t, yaml.Unmarshal([]byte(`CallProxyZkBytecode`), &ref))
		require.Equal(t, "CallProxyZkBytecode", ref.Symbol)
		require.Empty(t, ref.Package)
	})

	t.Run("mapping", func(t *testing.T) {
		t.Parallel()
		var ref ZkSyncBytecodeRef
		require.NoError(t, yaml.Unmarshal([]byte(`
package: github.com/example/zk
symbol: FooZkBytecode
`), &ref))
		require.Equal(t, "github.com/example/zk", ref.Package)
		require.Equal(t, "FooZkBytecode", ref.Symbol)
	})

	t.Run("mapping symbol only", func(t *testing.T) {
		t.Parallel()
		var ref ZkSyncBytecodeRef
		require.NoError(t, yaml.Unmarshal([]byte(`
symbol: FooZkBytecode
`), &ref))
		require.Empty(t, ref.Package)
		require.Equal(t, "FooZkBytecode", ref.Symbol)
	})

	t.Run("mapping without symbol", func(t *testing.T) {
		t.Parallel()
		var ref ZkSyncBytecodeRef
		err := yaml.Unmarshal([]byte(`
package: github.com/example/zk
`), &ref)
		require.ErrorContains(t, err, "zksync_bytecode mapping requires symbol")
	})

	t.Run("scalar overwrites previous mapping", func(t *testing.T) {
		t.Parallel()
		var ref ZkSyncBytecodeRef
		require.NoError(t, yaml.Unmarshal([]byte(`
package: github.com/example/zk
symbol: FooZkBytecode
`), &ref))
		require.NoError(t, yaml.Unmarshal([]byte(`BarZkBytecode`), &ref))
		require.Equal(t, "BarZkBytecode", ref.Symbol)
		require.Empty(t, ref.Package)
	})

	t.Run("null is unset", func(t *testing.T) {
		t.Parallel()
		var cfg struct {
			ZkSyncBytecode ZkSyncBytecodeRef `yaml:"zksync_bytecode"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(`zksync_bytecode: null`), &cfg))
		require.True(t, cfg.ZkSyncBytecode.IsZero())
	})

	t.Run("tilde is unset", func(t *testing.T) {
		t.Parallel()
		var cfg struct {
			ZkSyncBytecode ZkSyncBytecodeRef `yaml:"zksync_bytecode"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(`zksync_bytecode: ~`), &cfg))
		require.True(t, cfg.ZkSyncBytecode.IsZero())
	})

	t.Run("empty string is unset", func(t *testing.T) {
		t.Parallel()
		var cfg struct {
			ZkSyncBytecode ZkSyncBytecodeRef `yaml:"zksync_bytecode"`
		}
		require.NoError(t, yaml.Unmarshal([]byte(`zksync_bytecode: ""`), &cfg))
		require.True(t, cfg.ZkSyncBytecode.IsZero())
	})
}

func TestResolveZkSyncBytecodeUsesInputDefaultPackage(t *testing.T) {
	t.Parallel()

	const zkPackage = "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/zksync_bindings"
	cfg := EvmContractConfig{
		Name:           "ManyChainMultiSig",
		ZkSyncBytecode: mustZkSyncBytecodeRef(t, "ManyChainMultiSigZkBytecode"),
	}
	input := EvmInputConfig{ZkSyncBindingsPackage: zkPackage}

	pkg, symbol, err := resolveZkSyncBytecode(cfg, input, "github.com/example/evm")
	require.NoError(t, err)
	require.Equal(t, zkPackage, pkg)
	require.Equal(t, "ManyChainMultiSigZkBytecode", symbol)
}

func TestResolveZkSyncBytecodeFallsBackToGobindingsPackage(t *testing.T) {
	t.Parallel()

	const gobindingsPackage = "github.com/example/gobindings/link_token"
	cfg := EvmContractConfig{
		Name:           "LinkToken",
		ZkSyncBytecode: mustZkSyncBytecodeRef(t, "ZkBytecode"),
	}

	pkg, symbol, err := resolveZkSyncBytecode(cfg, EvmInputConfig{}, gobindingsPackage)
	require.NoError(t, err)
	require.Equal(t, gobindingsPackage, pkg)
	require.Equal(t, "ZkBytecode", symbol)
}

func TestExtractContractInfoRejectsZkSyncBytecodeWithOmitDeploy(t *testing.T) {
	t.Parallel()

	cfg := EvmContractConfig{
		Name:              "LinkToken",
		Version:           "1.0.0",
		OmitDeploy:        true,
		GobindingsPackage: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/v1_0_0/link_token",
		ZkSyncBytecode:    mustZkSyncBytecodeRef(t, "ZkBytecode"),
		Functions: []EvmFunctionConfig{
			{Name: "transfer", Access: "public"},
		},
	}

	_, err := extractContractInfo(cfg, EvmInputConfig{}, EvmOutputConfig{BasePath: t.TempDir()})
	require.ErrorContains(t, err, "zksync_bytecode cannot be set when omit_deploy is true")
}

func mustZkSyncBytecodeRef(t *testing.T, symbol string) ZkSyncBytecodeRef {
	t.Helper()
	var ref ZkSyncBytecodeRef
	require.NoError(t, yaml.Unmarshal([]byte(symbol), &ref))

	return ref
}
