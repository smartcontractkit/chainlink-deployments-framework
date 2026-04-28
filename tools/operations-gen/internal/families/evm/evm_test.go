package evm_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/generate"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

func TestGenerateContractGobindingsPackageOverridesInputDefault(t *testing.T) {
	t.Parallel()

	const overrideGobindingsPackage = "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/v1_0_0/link_token"
	config := `version: "1.0.0"
chain_family: evm

input:
  gobindings_package: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/unused_parent"

output:
  base_path: "."

contracts:
  - contract_name: LinkToken
    version: "1.0.0"
    gobindings_package: "` + overrideGobindingsPackage + `"
    functions:
      - name: transfer
        access: public
`

	var cfg core.Config
	require.NoError(t, yaml.Unmarshal([]byte(config), &cfg), "parsing config")

	tmpDir := t.TempDir()
	cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: tmpDir})

	tmpl, err := generate.LoadTemplate("evm")
	require.NoError(t, err, "loadTemplate")

	require.NoError(t, evm.Handler{}.Generate(cfg, tmpl, nil), "Generate")

	outputPath := core.ContractOutputPath(tmpDir, core.VersionToPath("1.0.0"), "link_token")
	got, err := os.ReadFile(outputPath)
	require.NoError(t, err, "reading generated file %s", outputPath)

	require.Contains(t, string(got), `gobindings "`+overrideGobindingsPackage+`"`)
	require.NotContains(t, string(got), "unused_parent")
}
