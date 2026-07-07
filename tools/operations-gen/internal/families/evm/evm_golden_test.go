package evm_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/generate"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

var update = flag.Bool("update", false, "update golden files")

// TestGenerateLinkToken is an end-to-end test that runs the generator against the
// LinkToken gobindings fixture and verifies that the generated output matches golden.
func TestGenerateLinkToken(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_config.yaml", "link_token.golden.go")
}

// TestGenerateManyChainMultiSig verifies generation against an MCMS-like ABI fixture.
func TestGenerateManyChainMultiSig(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_mcms_config.yaml", "many_chain_multi_sig.golden.go")
}

func TestGenerateRBACTimelock(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_rbac_timelock_config.yaml", "rbac_timelock.golden.go")
}

func TestGenerateFeeQuoter(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_fee_quoter.yaml", "fee_quoter.golden.go")
}

func TestGenerateWorkflowRegistry(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "opertions_gen_workflow_router.yaml", "workflow_registry.golden.go")
}

// TestGenerateLinkTokenWithZkSyncBindingsPackage verifies generation when
// input.zksync_bindings_package and contract zksync_bytecode are both set.
func TestGenerateLinkTokenWithZkSyncBindingsPackage(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_link_token_zksync_config.yaml", "link_token_zksync.golden.go")
}

func runGoldenGenerationTest(t *testing.T, configFileName string, goldenFileName string) {
	t.Helper()

	evmTestdataDir, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "evm"))
	require.NoError(t, err)

	configData, err := os.ReadFile(filepath.Join(evmTestdataDir, configFileName))
	require.NoError(t, err, "reading config")

	var cfg core.Config
	require.NoError(t, yaml.Unmarshal(configData, &cfg), "parsing config")

	// Override output path to an isolated temp dir.
	tmpDir := t.TempDir()
	cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: tmpDir})
	cfg.ConfigDir = ""

	handler := evm.Handler{}
	tmpl, err := generate.LoadTemplate("evm")
	require.NoError(t, err, "loadTemplate")

	require.NoError(t, handler.Generate(cfg, tmpl), "Generate")

	// Derive the output path from the first contract in the config, mirroring extractContractInfo.
	var contractCfgs []evm.EvmContractConfig
	require.NoError(t, cfg.Contracts.Decode(&contractCfgs), "decoding contract configs")
	require.NotEmpty(t, contractCfgs, "decoding contract configs")
	first := contractCfgs[0]
	pkgName := first.PackageName
	if pkgName == "" {
		pkgName = evm.ToSnakeCase(first.Name)
	}
	vPath := core.VersionToPath(first.Version)
	if first.VersionPath != "" {
		vPath = first.VersionPath
	}
	outputPath := core.ContractOutputPath(tmpDir, vPath, pkgName)

	got, err := os.ReadFile(outputPath)
	require.NoError(t, err, "reading generated file %s", outputPath)

	goldenPath := filepath.Join(evmTestdataDir, goldenFileName)

	if *update {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o600), "writing golden file") //nolint:gosec // G703: goldenPath is the in-repo testdata file, only written under -update by the developer

		return
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "reading golden file %s (run with -update to create it)", goldenPath)

	require.Equal(t, string(want), string(got), "generated output does not match golden file %s\n\nrun: go test ./... -run %s -update", goldenPath, t.Name())
}

func mustYAMLNode(t *testing.T, value any) yaml.Node {
	t.Helper()
	b, err := yaml.Marshal(value)
	require.NoError(t, err, "marshal yaml node")
	var n yaml.Node
	require.NoError(t, yaml.Unmarshal(b, &n), "unmarshal yaml node")

	return n
}
