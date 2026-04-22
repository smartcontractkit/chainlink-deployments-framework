package evm_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

var update = flag.Bool("update", false, "update golden files")

// goldenFixtureConfig is the shared YAML used by TestGenerate.
const goldenFixtureConfig = "operations_gen_config.yaml"

// TestGenerate runs Generate once for goldenFixtureConfig, then each subtest checks
// one contract's output against testdata/evm/<package>.golden.go.
func TestGenerate(t *testing.T) {
	t.Parallel()

	evmTestdataDir, tmpDir, contractCfgs := goldenTestRun(t, goldenFixtureConfig)

	tests := []struct {
		name         string
		contractName string // matches YAML contract_name
	}{
		{"generate many chain multisig", "ManyChainMultiSig"},
		{"generate rbac timelock", "RBACTimelock"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertContractGolden(t, evmTestdataDir, tmpDir, contractCfgs, tc.contractName)
		})
	}
}

// assertContractGolden compares or updates the golden for one contract after a shared
// goldenTestRun (tmpDir must already hold generated files for all contracts in contractCfgs).
func assertContractGolden(t *testing.T, evmTestdataDir, tmpDir string, contractCfgs []evm.ContractConfig, contractName string) {
	t.Helper()

	cc, ok := findContractConfig(contractCfgs, contractName)
	if !ok {
		t.Fatalf("no contract_name %q in config", contractName)
	}

	pkgName := contractPackageName(cc)
	outputPath := contractGeneratedPath(tmpDir, cc)
	goldenPath := filepath.Join(evmTestdataDir, pkgName+".golden.go")
	compareOrUpdateGolden(t, outputPath, goldenPath, cc.Name)
}

func findContractConfig(cfgs []evm.ContractConfig, contractName string) (evm.ContractConfig, bool) {
	for _, c := range cfgs {
		if c.Name == contractName {
			return c, true
		}
	}

	return evm.ContractConfig{}, false
}

// goldenTestRun loads config from testdata/evm, runs Generate into a temp dir, and
// returns the testdata directory, temp output root, and decoded contract configs.
func goldenTestRun(t *testing.T, configFileName string) (evmTestdataDir, tmpDir string, contractCfgs []evm.ContractConfig) {
	t.Helper()

	var err error
	evmTestdataDir, err = filepath.Abs(filepath.Join("..", "..", "..", "testdata", "evm"))
	if err != nil {
		t.Fatal(err)
	}

	configData, err := os.ReadFile(filepath.Join(evmTestdataDir, configFileName))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	var cfg core.Config
	if err = yaml.Unmarshal(configData, &cfg); err != nil {
		t.Fatalf("parsing config: %v", err)
	}

	tmpDir = t.TempDir()
	cfg.Output = mustYAMLNode(t, evm.OutputConfig{BasePath: tmpDir})
	cfg.ConfigDir = evmTestdataDir

	handler := evm.Handler{}
	tmpl, err := loadTemplateForTest()
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}

	if err = handler.Generate(cfg, tmpl); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if err = cfg.Contracts.Decode(&contractCfgs); err != nil || len(contractCfgs) == 0 {
		t.Fatalf("decoding contract configs: %v", err)
	}

	return evmTestdataDir, tmpDir, contractCfgs
}

func contractPackageName(cc evm.ContractConfig) string {
	if cc.PackageName != "" {
		return cc.PackageName
	}

	return evm.ToSnakeCase(cc.Name)
}

func contractVersionPath(cc evm.ContractConfig) string {
	vPath := core.VersionToPath(cc.Version)
	if cc.OutputVersionPath != "" {
		vPath = cc.OutputVersionPath
	}

	return vPath
}

func contractGeneratedPath(tmpDir string, cc evm.ContractConfig) string {
	return core.ContractOutputPath(tmpDir, contractVersionPath(cc), contractPackageName(cc))
}

// compareOrUpdateGolden reads generated output at gotPath and compares or writes
// goldenPath. If contractName is non-empty, mismatch errors mention that contract.
func compareOrUpdateGolden(t *testing.T, gotPath, goldenPath, contractName string) {
	t.Helper()

	got, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("reading generated file %s: %v", gotPath, err)
	}

	if *update {
		if err = os.WriteFile(goldenPath, got, 0o600); err != nil {
			t.Fatalf("writing golden file %s: %v", goldenPath, err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s: %v (run with -update to create it)", goldenPath, err)
	}

	if string(got) != string(want) {
		if contractName != "" {
			t.Errorf("generated output for %s does not match golden file %s\n\n",
				contractName, goldenPath)
		} else {
			t.Errorf("generated output does not match golden file %s\n\n", goldenPath)
		}
	}
}

func mustYAMLNode(t *testing.T, value any) yaml.Node {
	t.Helper()
	b, err := yaml.Marshal(value)
	if err != nil {
		t.Fatalf("marshal yaml node: %v", err)
	}
	var n yaml.Node
	if err = yaml.Unmarshal(b, &n); err != nil {
		t.Fatalf("unmarshal yaml node: %v", err)
	}

	return n
}

func loadTemplateForTest() (*template.Template, error) {
	path := filepath.Join("..", "..", "..", "templates", "evm", "operations.tmpl")
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return template.New("operations").Parse(string(content))
}
