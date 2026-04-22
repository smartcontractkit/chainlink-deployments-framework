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

// TestGenerateLinkToken is an end-to-end test that runs the generator against the
// real LinkToken ABI/bytecode and verifies that the generated output matches golden.
func TestGenerateLinkToken(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_config.yaml", "link_token.golden.go")
}

// TestGenerateManyChainMultiSig verifies generation against an MCMS-like ABI fixture.
func TestGenerateManyChainMultiSig(t *testing.T) {
	t.Parallel()
	runGoldenGenerationTest(t, "operations_gen_mcms_config.yaml", "many_chain_multi_sig.golden.go")
}

func runGoldenGenerationTest(t *testing.T, configFileName string, goldenFileName string) {
	t.Helper()

	evmTestdataDir, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "evm"))
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

	// Override paths: inputs point to fixture dirs, output to a temp dir.
	cfg.Input = mustYAMLNode(t, evm.EvmInputConfig{
		ABIBasePath:      filepath.Join(evmTestdataDir, "abi"),
		BytecodeBasePath: filepath.Join(evmTestdataDir, "bytecode"),
	})
	tmpDir := t.TempDir()
	cfg.Output = mustYAMLNode(t, evm.EvmOutputConfig{BasePath: tmpDir})
	cfg.ConfigDir = ""

	handler := evm.Handler{}
	tmpl, err := loadTemplateForTest()
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}

	if err = handler.Generate(cfg, tmpl); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Derive the output path from the first contract in the config, mirroring extractContractInfo.
	var contractCfgs []evm.EvmContractConfig
	if err = cfg.Contracts.Decode(&contractCfgs); err != nil || len(contractCfgs) == 0 {
		t.Fatalf("decoding contract configs: %v", err)
	}
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
	if err != nil {
		t.Fatalf("reading generated file %s: %v", outputPath, err)
	}

	goldenPath := filepath.Join(evmTestdataDir, goldenFileName)

	if *update {
		if err = os.WriteFile(goldenPath, got, 0o600); err != nil {
			t.Fatalf("writing golden file: %v", err)
		}

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s: %v (run with -update to create it)", goldenPath, err)
	}

	if string(got) != string(want) {
		t.Errorf("generated output does not match golden file %s\n\nrun: go test ./... -run %s -update", goldenPath, t.Name())
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
