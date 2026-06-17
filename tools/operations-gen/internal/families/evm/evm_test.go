package evm_test

import (
	"fmt"
	"os"
	"path/filepath"
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

	require.NoError(t, evm.Handler{}.Generate(cfg, tmpl), "Generate")

	outputPath := core.ContractOutputPath(tmpDir, core.VersionToPath("1.0.0"), "link_token")
	got, err := os.ReadFile(outputPath)
	require.NoError(t, err, "reading generated file %s", outputPath)

	require.Contains(t, string(got), `gobindings "`+overrideGobindingsPackage+`"`)
	require.NotContains(t, string(got), "unused_parent")
}

func TestGenerateResolvesRelativeInputGobindingsPackage(t *testing.T) {
	t.Parallel()

	const wantGobindingsPackage = "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/v1_0_0/link_token"
	config := `version: "1.0.0"
chain_family: evm

input:
  gobindings_package: "./testdata/evm/gobindings"

output:
  base_path: "."

contracts:
  - contract_name: LinkToken
    version: "1.0.0"
    functions:
      - name: transfer
        access: public
`

	var cfg core.Config
	require.NoError(t, yaml.Unmarshal([]byte(config), &cfg), "parsing config")

	moduleDir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	tmpDir := t.TempDir()
	outputBasePath, err := filepath.Rel(moduleDir, tmpDir)
	require.NoError(t, err)
	cfg.ConfigDir = moduleDir
	cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: outputBasePath})

	tmpl, err := generate.LoadTemplate("evm")
	require.NoError(t, err, "loadTemplate")

	require.NoError(t, evm.Handler{}.Generate(cfg, tmpl), "Generate")

	outputPath := core.ContractOutputPath(tmpDir, core.VersionToPath("1.0.0"), "link_token")
	got, err := os.ReadFile(outputPath)
	require.NoError(t, err, "reading generated file %s", outputPath)

	require.Contains(t, string(got), `gobindings "`+wantGobindingsPackage+`"`)
	require.NotContains(t, string(got), `gobindings "./testdata/evm/gobindings`)
}

func TestDeployContractTypesGeneratesExtraVarsAndBytecodeEntries(t *testing.T) {
	t.Parallel()

	config := `version: "1.0.0"
chain_family: evm

input:
  gobindings_package: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings"

output:
  base_path: "."

contracts:
  - contract_name: LinkToken
    version: "1.0.0"
    deploy_contract_types:
      - AliasLinkToken
      - AnotherLinkToken
    functions:
      - name: transfer
        access: public
`

	var cfg core.Config
	require.NoError(t, yaml.Unmarshal([]byte(config), &cfg), "parsing config")

	tmpDir := t.TempDir()
	cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: tmpDir})

	tmpl, err := generate.LoadTemplate("evm")
	require.NoError(t, err)

	require.NoError(t, evm.Handler{}.Generate(cfg, tmpl))

	outputPath := core.ContractOutputPath(tmpDir, core.VersionToPath("1.0.0"), "link_token")
	got, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	src := string(got)
	require.Contains(t, src, `var AliasLinkTokenContractType cldf_deployment.ContractType = "AliasLinkToken"`)
	require.Contains(t, src, `var AliasLinkTokenTypeAndVersion = cldf_deployment.NewTypeAndVersion(AliasLinkTokenContractType, *Version)`)
	require.Contains(t, src, `var AnotherLinkTokenContractType cldf_deployment.ContractType = "AnotherLinkToken"`)
	require.Contains(t, src, `var AnotherLinkTokenTypeAndVersion = cldf_deployment.NewTypeAndVersion(AnotherLinkTokenContractType, *Version)`)
	require.Contains(t, src, `AliasLinkTokenTypeAndVersion.String()`)
	require.Contains(t, src, `AnotherLinkTokenTypeAndVersion.String()`)
	// Base TypeAndVersion must not be emitted when deploy_contract_types is set.
	require.NotContains(t, src, `var TypeAndVersion = cldf_deployment.NewTypeAndVersion(ContractType, *Version)`)
	require.NotContains(t, src, `cldf_deployment.NewTypeAndVersion(ContractType, *Version).String()`)
}

func TestDeployContractTypesValidationErrors(t *testing.T) {
	t.Parallel()

	const baseConfig = `version: "1.0.0"
chain_family: evm

input:
  gobindings_package: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings"

output:
  base_path: "."

contracts:
  - contract_name: LinkToken
    version: "1.0.0"
    %s
    functions:
      - name: transfer
        access: public
`

	cases := []struct {
		name    string
		snippet string
		wantErr string
	}{
		{
			name: "omit_deploy with deploy_contract_types",
			snippet: `omit_deploy: true
    deploy_contract_types:
      - AliasLinkToken`,
			wantErr: "deploy_contract_types cannot be set when omit_deploy is true",
		},
		{
			name: "empty entry",
			snippet: `deploy_contract_types:
      - ""`,
			wantErr: "deploy_contract_types entries must not be empty",
		},
		{
			name: "duplicate entry",
			snippet: `deploy_contract_types:
      - AliasLinkToken
      - AliasLinkToken`,
			wantErr: `duplicate deploy_contract_types entry "AliasLinkToken"`,
		},
		{
			name: "base contract name as entry",
			snippet: `deploy_contract_types:
      - LinkToken`,
			wantErr: `deploy_contract_types must not contain the base contract name "LinkToken"`,
		},
		{
			name:    "empty list",
			snippet: `deploy_contract_types: []`,
			wantErr: "deploy_contract_types must contain at least one entry",
		},
		{
			name: "invalid identifier lowercase",
			snippet: `deploy_contract_types:
      - proposerLinkToken`,
			wantErr: `deploy_contract_types entry "proposerLinkToken" must be a valid Go exported identifier`,
		},
		{
			name: "invalid identifier with space",
			snippet: `deploy_contract_types:
      - "Proposer LinkToken"`,
			wantErr: `deploy_contract_types entry "Proposer LinkToken" must be a valid Go exported identifier`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfgYAML := fmt.Sprintf(baseConfig, tc.snippet)
			var cfg core.Config
			require.NoError(t, yaml.Unmarshal([]byte(cfgYAML), &cfg))

			cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: t.TempDir()})

			tmpl, err := generate.LoadTemplate("evm")
			require.NoError(t, err)

			err = evm.Handler{}.Generate(cfg, tmpl)
			require.Error(t, err)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestGenerateResolvesRelativeZkSyncBindingsPackage(t *testing.T) {
	t.Parallel()

	const wantZkSyncPackage = "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/zksync_bindings"
	config := `version: "1.0.0"
chain_family: evm

input:
  gobindings_package: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings"
  zksync_bindings_package: "./testdata/evm/zksync_bindings"

output:
  base_path: "."

contracts:
  - contract_name: ManyChainMultiSig
    version: "1.0.0"
    package_name: many_chain_multi_sig
    zksync_bytecode: ManyChainMultiSigZkBytecode
    functions:
      - name: owner
        access: public
`

	var cfg core.Config
	require.NoError(t, yaml.Unmarshal([]byte(config), &cfg), "parsing config")

	moduleDir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	tmpDir := t.TempDir()
	outputBasePath, err := filepath.Rel(moduleDir, tmpDir)
	require.NoError(t, err)
	cfg.ConfigDir = moduleDir
	cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: outputBasePath})

	tmpl, err := generate.LoadTemplate("evm")
	require.NoError(t, err, "loadTemplate")

	require.NoError(t, evm.Handler{}.Generate(cfg, tmpl), "Generate")

	outputPath := core.ContractOutputPath(tmpDir, core.VersionToPath("1.0.0"), "many_chain_multi_sig")
	got, err := os.ReadFile(outputPath)
	require.NoError(t, err, "reading generated file %s", outputPath)

	require.Contains(t, string(got), `zkbindings "`+wantZkSyncPackage+`"`)
	require.Contains(t, string(got), "ZkSyncVM: zkbindings.ManyChainMultiSigZkBytecode")
}
