package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"

	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// --- Structure & Metadata Tests ---

func TestNewEvmCmds_Structure(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	root := c.NewEvmCmds(domain)

	require.Equal(t, "evm", root.Use)

	subs := root.Commands()
	require.Len(t, subs, 4)

	uses := []string{subs[0].Use, subs[1].Use, subs[2].Use, subs[3].Use}
	require.ElementsMatch(t, []string{"gas", "nonce", "nodes", "contract"}, uses)

	require.NotNil(t, root.PersistentFlags().Lookup("environment"))
	require.NotNil(t, root.PersistentFlags().Lookup("selector"))

	// nonce has clear
	n := subs[indexOf(subs, "nonce")]
	nonceSubs := n.Commands()
	require.Len(t, nonceSubs, 1)
	require.Equal(t, "clear", nonceSubs[0].Use)
}

func TestEvmClearCommand_Metadata(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	root := c.NewEvmCmds(domain)

	tcData := []struct {
		path        []string
		wantUse     string
		wantShort   string
		wantLong    string
		wantExample string
		wantFlags   []string
	}{
		{
			path:        []string{"nonce", "clear"},
			wantUse:     "clear",
			wantShort:   "Clear any stuck txes",
			wantLong:    "Clear any stuck transactions for the deployer key on an EVM chain.",
			wantExample: "--1559",
			wantFlags:   []string{"environment", "selector", "1559"},
		},
	}

	for _, tc := range tcData {
		t.Run(strings.Join(tc.path, " "), func(t *testing.T) {
			t.Parallel()
			cmd, _, err := root.Find(tc.path)
			require.NoError(t, err)
			require.Equal(t, tc.wantUse, cmd.Use)
			require.Contains(t, cmd.Short, tc.wantShort)
			require.Contains(t, cmd.Long, tc.wantLong)
			require.Contains(t, cmd.Example, tc.wantExample)

			for _, f := range tc.wantFlags {
				var flag *pflag.Flag
				if f == "environment" || f == "selector" {
					flag = root.PersistentFlags().Lookup(f)
				} else {
					flag = cmd.Flags().Lookup(f)
				}
				require.NotNil(t, flag)
			}
		})
	}
}

func TestNewEvmNonceClear_Metadata(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	cmd := c.newEvmNonceClear(domain)

	require.Equal(t, "clear", cmd.Use)
	require.Contains(t, cmd.Short, "Clear any stuck txes for the deployer key")
	require.Contains(t, cmd.Long, "Clear any stuck transactions for the deployer key on an EVM chain.")
	require.Contains(t, cmd.Example, "--1559")

	// Local flag
	require.NotNil(t, cmd.Flags().Lookup("1559"))

	// Persistent flags live on the parent
	parent := &cobra.Command{}
	parent.AddCommand(cmd)
	parent.PersistentFlags().StringP("environment", "e", "", "")
	parent.PersistentFlags().Uint64P("selector", "s", 0, "")
	require.NotNil(t, parent.PersistentFlags().Lookup("environment"))
	require.NotNil(t, parent.PersistentFlags().Lookup("selector"))
}

func TestNewEvmNodesFund_Metadata(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	cmd := c.newEvmNodesFund(domain)

	// Basic use/description/examples
	require.Equal(t, "fund", cmd.Use)
	require.Contains(t, cmd.Short, "Ensure all nodes have a certain amount of gas")
	require.Contains(t, cmd.Long, "Ensure all OCR2 nodes have a target amount of gas in their account on an EVM chain.")
	require.Contains(t, cmd.Example, "--amount")

	// Local flags
	require.NotNil(t, cmd.Flags().Lookup("amount"), "amount flag should exist")
	require.NotNil(t, cmd.Flags().Lookup("1559"), "1559 flag should exist")

	// The local 'amount' flag should be required
	_, err := cmd.Flags().GetString("amount")
	require.NoError(t, err)
	// Persistent flags live on the parent
	parent := &cobra.Command{}
	parent.PersistentFlags().StringP("environment", "e", "", "")
	parent.PersistentFlags().Uint64P("selector", "s", 0, "")
	parent.AddCommand(cmd)
	_ = parent.MarkPersistentFlagRequired("environment")
	_ = parent.MarkPersistentFlagRequired("selector")

	require.NotNil(t, parent.PersistentFlags().Lookup("environment"), "environment flag should exist on parent")
	require.NotNil(t, parent.PersistentFlags().Lookup("selector"), "selector flag should exist on parent")
}

// --- newEvmGasSend Metadata Tests ---

func TestNewEvmGasSend_Metadata(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	cmd := c.newEvmGasSend(domain)

	require.Equal(t, "send", cmd.Use)
	require.Contains(t, cmd.Short, "Send gas token to an address")
	require.Contains(t, cmd.Long, "Send a specified amount of gas tokens to an address on an EVM chain.")
	require.Contains(t, cmd.Example, "exemplar evm gas send")

	// Local flags
	require.NotNil(t, cmd.Flags().Lookup("amount"))
	require.NotNil(t, cmd.Flags().Lookup("to"))
	require.NotNil(t, cmd.Flags().Lookup("1559"))

	// 'amount' and 'to' are required flags
	err := cmd.MarkFlagRequired("amount")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("to")
	require.NoError(t, err)

	// Persistent flags live on the parent
	parent := &cobra.Command{}
	parent.AddCommand(cmd)
	parent.PersistentFlags().StringP("environment", "e", "", "env")
	parent.PersistentFlags().Uint64P("selector", "s", 0, "sel")
	_ = parent.MarkPersistentFlagRequired("environment")
	_ = parent.MarkPersistentFlagRequired("selector")

	require.NotNil(t, parent.PersistentFlags().Lookup("environment"))
	require.NotNil(t, parent.PersistentFlags().Lookup("selector"))
}

func TestNewEvmContractVerify_Metadata(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	cmd := c.newEvmContractVerify(domain)

	require.Equal(t, "verify", cmd.Use)
	require.Contains(t, cmd.Short, "Verify evm contract")
	require.Contains(t, cmd.Long, "Verify a contract on Etherscan using forge")
	require.Contains(t, cmd.Example, "--contract-address")

	// Local flags
	require.NotNil(t, cmd.Flags().Lookup("optimizer-runs"))
	require.NotNil(t, cmd.Flags().Lookup("compiler-version"))
	require.NotNil(t, cmd.Flags().Lookup("contract-address"))
	require.NotNil(t, cmd.Flags().Lookup("name"))
	require.NotNil(t, cmd.Flags().Lookup("dir"))
	require.NotNil(t, cmd.Flags().Lookup("commit"))

	// Required flags
	err := cmd.MarkFlagRequired("contract-address")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("optimizer-runs")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("name")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("dir")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("commit")
	require.NoError(t, err)

	// Persistent flags live on the parent
	parent := &cobra.Command{}
	parent.PersistentFlags().StringP("environment", "e", "", "")
	parent.PersistentFlags().Uint64P("selector", "s", 0, "")
	parent.AddCommand(cmd)
	_ = parent.MarkPersistentFlagRequired("environment")
	_ = parent.MarkPersistentFlagRequired("selector")

	require.NotNil(t, parent.PersistentFlags().Lookup("environment"))
	require.NotNil(t, parent.PersistentFlags().Lookup("selector"))
}

func TestNewEvmContractBatchVerify_Metadata(t *testing.T) {
	t.Parallel()
	c := Commands{}
	var domain cldf_domain.Domain
	cmd := c.newEvmContractBatchVerify(domain)

	require.Equal(t, "verify-batch", cmd.Use)
	require.Contains(t, cmd.Short, "Verify batch evm contracts")
	require.Contains(t, cmd.Long, "Verify a list of contracts on Etherscan compatible chain viewers using forge")
	require.Contains(t, cmd.Example, "--contracts")

	// Local flags
	require.NotNil(t, cmd.Flags().Lookup("contracts"))
	require.NotNil(t, cmd.Flags().Lookup("dir"))
	require.NotNil(t, cmd.Flags().Lookup("commit"))

	// Required flags
	err := cmd.MarkFlagRequired("contracts")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("dir")
	require.NoError(t, err)
	err = cmd.MarkFlagRequired("commit")
	require.NoError(t, err)

	// Persistent flags live on the parent
	parent := &cobra.Command{}
	parent.PersistentFlags().StringP("environment", "e", "", "")
	parent.PersistentFlags().Uint64P("selector", "s", 0, "")
	parent.AddCommand(cmd)
	_ = parent.MarkPersistentFlagRequired("environment")
	_ = parent.MarkPersistentFlagRequired("selector")

	require.NotNil(t, parent.PersistentFlags().Lookup("environment"))
	require.NotNil(t, parent.PersistentFlags().Lookup("selector"))
}

func TestAppendEtherscanInfoToFoundryToml(t *testing.T) {
	t.Parallel()

	var testSourceCfg = `[profile.default]
auto_detect_solc = true
optimizer = true
optimizer_runs = 1_000_000
`

	var testResultCfg = `[profile.default]
auto_detect_solc = true
optimizer = true
optimizer_runs = 1_000_000
[etherscan]
chain = { key = "api-key", url = "https://test-url.com", chain = "100" }`

	// create a temporary directory
	tmpDir, err := os.MkdirTemp("", "dir")
	require.NoError(t, err)
	defer func() {
		err = os.RemoveAll(tmpDir)
		require.NoError(t, err)
	}()

	foundryToml := filepath.Join(tmpDir, "foundry.toml")

	err = os.WriteFile(foundryToml, []byte(testSourceCfg), 0600)
	require.NoError(t, err)

	closeFn, err := appendEtherscanInfoToFoundryToml(tmpDir, "100", cldf_config_network.BlockExplorer{
		Type:   "Etherscan",
		APIKey: "api-key",
		URL:    "https://test-url.com",
	})
	require.NoError(t, err)

	// result file is correct
	dat, err := os.ReadFile(foundryToml)
	require.NoError(t, err)
	require.Equal(t, testResultCfg, string(dat))

	require.NoError(t, closeFn())

	// the changes were reverted
	dat, err = os.ReadFile(foundryToml)
	require.NoError(t, err)
	require.Equal(t, testSourceCfg, string(dat))
}

func TestReadContractsList(t *testing.T) {
	t.Parallel()

	example, err := readContractsList(filepath.Join(cldf_domain.ProjectRoot, "contracts.example.toml"))
	require.NoError(t, err)
	require.Len(t, example.Contracts, 2)
	require.Equal(t, "0xF389104dFaD66cED6B77b879FF76b572a8cC3590", example.Contracts[0].ContractAddress)
	require.Equal(t, "testnet", example.Contracts[0].Environment)
	require.Equal(t, "EVM2EVMOffRamp", example.Contracts[0].ContractName)
	require.Equal(t, uint64(6955638871347136141), example.Contracts[0].Selector)
	require.Equal(t, uint64(26000), example.Contracts[0].OptimizerRuns)
	require.Equal(t, "v0.8.24", example.Contracts[0].CompilerVersion)
}
