package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerate(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	config := fmt.Sprintf(`version: "1.0.0"
	chain_family: evm

	output:
	  base_path: %q

	contracts:
	  - contract_name: LinkToken
		version: "1.0.0"
		gobindings_package: "github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/testdata/evm/gobindings/v1_0_0/link_token"
		functions:
		  - name: transfer
			access: public
	`, outputDir)

	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte(config), &cfg))

	require.NoError(t, Generate(cfg))

	outputPath := filepath.Join(outputDir, "v1_0_0", "operations", "link_token", "link_token.go")
	_, err := os.Stat(outputPath)
	require.NoError(t, err)
}

func TestGenerateUnsupportedFamily(t *testing.T) {
	t.Parallel()

	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte(`chain_family: solana`), &cfg))

	err := Generate(cfg)
	require.ErrorContains(t, err, `unsupported chain_family "solana"`)
}
