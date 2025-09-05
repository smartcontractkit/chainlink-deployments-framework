package commands

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	chainsel "github.com/smartcontractkit/chain-selectors"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/spf13/cobra"

	fclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/chains"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
)

// NewEvmCmds creates EVM-related commands for managing gas, nonces, and node funding.
func (c Commands) NewEvmCmds(domain domain.Domain) *cobra.Command {
	evmCmd := &cobra.Command{
		Use:   "evm",
		Short: "EVM commands",
	}

	gasCmd := &cobra.Command{
		Use:   "gas",
		Short: "Manage gas tokens",
	}
	gasCmd.AddCommand(c.newEvmGasSend(domain))

	nonceCmd := &cobra.Command{
		Use:   "nonce",
		Short: "Nonce management for EVM chains",
	}
	nonceCmd.AddCommand(c.newEvmNonceClear(domain))

	nodesCmd := &cobra.Command{
		Use:   "nodes",
		Short: "Manage gas on OCR2 nodes",
	}
	nodesCmd.AddCommand(c.newEvmNodesFund(domain))

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Manage evm contracts",
	}
	contractCmd.AddCommand(c.newEvmContractVerify(domain))
	contractCmd.AddCommand(c.newEvmContractBatchVerify(domain))

	evmCmd.AddCommand(gasCmd, nonceCmd, nodesCmd, contractCmd)

	evmCmd.PersistentFlags().StringP("environment", "e", "", "Deployment environment (required)")
	_ = evmCmd.MarkPersistentFlagRequired("environment")
	evmCmd.PersistentFlags().Uint64P("selector", "s", 0, "Chain selector ID (required)")
	_ = evmCmd.MarkPersistentFlagRequired("selector")

	return evmCmd
}

var (
	evmNonceClearLong = cli.LongDesc(`
	Clear any stuck transactions for the deployer key on an EVM chain.
`)
	evmNonceClearExample = cli.Examples(`
	exemplar evm nonce clear --environment staging --selector 1 --1559
`)
)

func (c Commands) newEvmNonceClear(domain domain.Domain) *cobra.Command {
	var (
		use1559 bool
	)

	cmd := cobra.Command{
		Use:     "clear",
		Short:   "Clear any stuck txes for the deployer key",
		Long:    evmNonceClearLong,
		Example: evmNonceClearExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			chainselector, _ := cmd.Flags().GetUint64("selector")

			_, ok := chainsel.ChainBySelector(chainselector)
			if !ok {
				return fmt.Errorf("EVM chain not found for selector %d", chainselector)
			}

			config, err := config.Load(domain, envKey, c.lggr)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			chains, err := chains.LoadChains(
				cmd.Context(), c.lggr, config, []uint64{chainselector},
			)
			if err != nil {
				return fmt.Errorf("failed to load chains from env: %w", err)
			}

			evmChains := chains.EVMChains()

			chain := evmChains[chainselector]
			fromAddr := chain.DeployerKey.From

			pendingNonce, err := chain.Client.PendingNonceAt(cmd.Context(), fromAddr)
			if err != nil {
				return err
			}
			nonce, err := chain.Client.(*fclient.MultiClient).Client.NonceAt(cmd.Context(), fromAddr, nil)
			if err != nil {
				return err
			}
			if nonce == pendingNonce {
				fmt.Printf("No pending transactions for %s", fromAddr)
				return nil
			}

			// Otherwise we need to cancel each one with a self send of 0 value eth.
			for i := nonce; i < pendingNonce; i++ {
				var tx *types.Transaction
				if use1559 {
					// Do a heavy 50% boost on the self send.
					tx, err = build1559Tx(cmd.Context(), c.lggr,
						chain,
						big.NewInt(0),
						fromAddr,
						i, big.NewInt(30), big.NewInt(10))
				} else {
					// TODO
					return errors.New("legacy txes not supported for nonce clearing")
				}
				if err != nil {
					return err
				}
				signedTx, err1 := chain.DeployerKey.Signer(fromAddr, tx)
				if err1 != nil {
					return err1
				}
				err1 = chain.Client.SendTransaction(cmd.Context(), signedTx)
				if err1 != nil {
					return err1
				}
				fmt.Printf("Confirming cancellation transaction %s", signedTx.Hash().Hex())
				_, err1 = chain.Confirm(signedTx)
				if err1 != nil {
					return err1
				}
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&use1559, "1559", false, "Use 1559")

	return &cmd
}

var (
	evmGasSendLong = cli.LongDesc(`
	Send a specified amount of gas tokens to an address on an EVM chain.
`)
	evmGasSendExample = cli.Examples(`
	# Send 1 ETH (in wei) to an address
	exemplar evm gas send --environment staging --selector 1 --amount 1000000000000000000 --to 0xABC... --1559
`)
)

func (c Commands) newEvmGasSend(domain domain.Domain) *cobra.Command {
	var (
		amountStr string
		toStr     string
		use1559   bool
	)

	cmd := cobra.Command{
		Use:     "send",
		Short:   "Send gas token to an address",
		Long:    evmGasSendLong,
		Example: evmGasSendExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			chainselector, _ := cmd.Flags().GetUint64("selector")

			_, ok := chainsel.ChainBySelector(chainselector)
			if !ok {
				return fmt.Errorf("EVM chain not found for selector %d", chainselector)
			}

			config, err := config.Load(domain, envKey, c.lggr)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			chains, err := chains.LoadChains(
				cmd.Context(), c.lggr, config, []uint64{chainselector},
			)
			if err != nil {
				return fmt.Errorf("failed to load chains from env: %w", err)
			}

			evmChains := chains.EVMChains()

			chain := evmChains[chainselector]
			err = sendGas(cmd.Context(), c.lggr, chain, amountStr, toStr, use1559)
			if err != nil {
				return fmt.Errorf("failed to send gas: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&amountStr, "amount", "a", "", "Amount to send")
	cmd.Flags().StringVar(&toStr, "to", "", "Destination address to send to")
	cmd.Flags().BoolVar(&use1559, "1559", false, "Use EIP-1559 transaction")

	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("to")

	return &cmd
}

var (
	evmNodesFundLong = cli.LongDesc(`
	Ensure all OCR2 nodes have a target amount of gas in their account on an EVM chain.
`)

	evmNodesFundExample = cli.Examples(`
	# Fund all nodes with at least 0.5 ETH (in wei) on chain 1 in staging
	exemplar evm nodes fund --environment staging --selector 1 --amount 500000000000000000 --1559
`)
)

func (c Commands) newEvmNodesFund(domain domain.Domain) *cobra.Command {
	var (
		amountStr string
		use1559   bool
	)

	cmd := cobra.Command{
		Use:     "fund",
		Short:   "Ensure all nodes have a certain amount of gas in the env for a specific evm chain",
		Long:    evmNodesFundLong,
		Example: evmNodesFundExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			chainselector, _ := cmd.Flags().GetUint64("selector")

			env, err := environment.Load(cmd.Context, c.lggr, envKey, domain, true /*useRealBackends*/)
			if err != nil {
				return fmt.Errorf("failed to load command environment: %w", err)
			}
			cs, exists := chainsel.ChainBySelector(chainselector)
			if !exists {
				return fmt.Errorf("chain not found for selector %d", chainselector)
			}
			chain := env.BlockChains.EVMChains()[cs.Selector]
			targetAmount, success := big.NewInt(0).SetString(amountStr, 10)
			if !success {
				return errors.New("invalid amount")
			}
			for _, node := range env.NodeIDs {
				chainConfigs, err := env.Offchain.ListNodeChainConfigs(cmd.Context(),
					&nodev1.ListNodeChainConfigsRequest{
						Filter: &nodev1.ListNodeChainConfigsRequest_Filter{
							NodeIds: []string{node},
						},
					})
				if err != nil {
					return fmt.Errorf("failed to list node chain configs: %w", err)
				}
				for _, chainConfig := range chainConfigs.ChainConfigs {
					if strconv.FormatUint(cs.EvmChainID, 10) != chainConfig.Chain.Id {
						continue
					}
					if chainConfig.Ocr2Config == nil {
						cmd.Println("Skipping node", "node", node, "chain", chainConfig.Chain.Id, "reason", "no ocr2 config")
						continue
					}
					if chainConfig.Ocr2Config.IsBootstrap {
						cmd.Println("Skipping bootstrap node", node)
						continue
					}
					b, err := chain.Client.BalanceAt(cmd.Context(), common.HexToAddress(chainConfig.AccountAddress), nil)
					if err != nil {
						return fmt.Errorf("failed to get balance: %w", err)
					}
					cmd.Printf("Current balance %d for %s on node %s", b, chainConfig.AccountAddress, node)
					// Let's fund the difference.
					if b.Cmp(targetAmount) < 0 {
						amount := big.NewInt(0).Sub(targetAmount, b)
						cmd.Printf("Current balance insufficient, funding node %s's address %s with %d", node, chainConfig.AccountAddress, amount)
						err = sendGas(cmd.Context(), c.lggr, chain, amount.String(), chainConfig.AccountAddress, use1559)
						if err != nil {
							return fmt.Errorf("failed to send gas: %w", err)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&amountStr, "amount", "a", "", "Target amount of gas to ensure for each node")
	cmd.Flags().BoolVar(&use1559, "1559", false, "Use EIP-1559 transaction")

	return &cmd
}

const (
	// BytecodeOffset takes a length-128 hex string sample at the end of the contract bytecode to compare in transaction data.
	// 128 hex (64 bytes) provides both minimum collision risk with encoded constructor args, and minimum mismatch risk with transaction init data.
	BytecodeOffset         = 128
	defaultCompilerVersion = "v0.8.24"
)

type ContractToVerify struct {
	Selector        uint64 `toml:"selector"`
	Environment     string `toml:"environment"`
	ContractAddress string `toml:"contract_address"`
	ContractName    string `toml:"contract_name"`
	OptimizerRuns   uint64 `toml:"optimizer_runs"`
	CompilerVersion string `toml:"compiler_version"`
}

type ContractsToVerify struct {
	Contracts []ContractToVerify `toml:"contracts"`
}

var (
	evmContractVerifyLong = cli.LongDesc(`
		Verify a contract on Etherscan using forge.
		Main benefit is this will determine the constructor args automatically" +
		"Required dependencies: forge and git`)

	evmContractVerifyExample = cli.Examples(`
	    # Verify a contract on Etherscan using forge for exmemplar binary.
		exemplar contract verify -e testnet --selector 6955638871347136141 \
			--contract-address 0xF389104dFaD66cED6B77b879FF76b572a8cC3590 \
			--dir ../../../ccip/contracts --name EVM2EVMOffRamp \
			--commit dee09c782c17de1e37e3a1cf625d430330532c6d \
			--optimizer-runs 26000 \
			--compiler-version v0.8.24`)
)

func (c Commands) newEvmContractVerify(domain domain.Domain) *cobra.Command {
	var (
		contractDirectory string
		commit            string
		contractAddress   string
		compilerVersion   string
		optimizerRuns     uint64
		contractName      string
	)
	cmd := cobra.Command{
		Use:     "verify",
		Short:   "Verify evm contract",
		Long:    evmContractVerifyLong,
		Example: evmContractVerifyExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			chainselector, _ := cmd.Flags().GetUint64("selector")

			return verifyContract(
				cmd.Context(),
				c.lggr,
				domain,
				chainselector,
				envKey,
				contractDirectory,
				commit,
				contractAddress,
				compilerVersion,
				optimizerRuns,
				contractName,
			)
		},
	}
	cmd.Flags().Uint64Var(&optimizerRuns, "optimizer-runs", 0,
		"Optimizer runs used to compile the contract, take a look at foundry-toml to find this value")
	cmd.Flags().StringVar(&compilerVersion, "compiler-version", defaultCompilerVersion,
		"Compiler version")
	cmd.Flags().StringVarP(&contractAddress, "contract-address", "a", "",
		"Contract address to verify")
	cmd.Flags().StringVarP(&contractName, "name", "n", "",
		"Name of contract to verify. Eg. contract Offramp -> Offramp."+
			"You should see foundry-artifacts directory containing your contract in the directory specified. If not, run forge build")
	cmd.Flags().StringVarP(&contractDirectory, "dir", "d", "",
		"Path to the git directory containing the contract. This should normally be wherever the foundry.toml file is located")
	cmd.Flags().StringVar(&commit, "commit", "",
		"The commit to use in the directory")

	_ = cmd.MarkFlagRequired("contract-address")
	_ = cmd.MarkFlagRequired("optimizer-runs")
	_ = cmd.MarkFlagRequired("contract-name")
	_ = cmd.MarkFlagRequired("dir")
	_ = cmd.MarkFlagRequired("commit")

	return &cmd
}

var (
	evmContractVerifyBatchLong = cli.LongDesc(`
		Verify a list of contracts on Etherscan compatible chain viewers using forge.
        Main benefit is this will determine the constructor args automatically.
		Required dependencies: forge and git`)

	evmContractVerifyBatchExample = cli.Examples(`
           # Verify a contract batch on Etherscan using forge for exemplar binary.
			Provide contracts list via toml config (contracts.example.toml for reference) via contracts argument.

			exemplar contract verify-batch \
				--contracts ../../contracts.example.toml \
				--dir ../../../exemplar/contracts \
				--commit dee09c782c17de1e37e3a1cf625d430330532c6d`)
)

func (c Commands) newEvmContractBatchVerify(domain domain.Domain) *cobra.Command {
	var (
		contractDirectory string
		commit            string
		contracts         string
	)
	cmd := cobra.Command{
		Use:     "verify-batch",
		Short:   "Verify batch evm contracts",
		Long:    evmContractVerifyBatchLong,
		Example: evmContractVerifyBatchExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			contractsList, err := readContractsList(contracts)
			if err != nil {
				return fmt.Errorf("failed to load contracts from file: %w", err)
			}

			for i, contract := range contractsList.Contracts {
				cmd.Printf(
					"#%d Verifying contract %s %s (chain selector %d)",
					i,
					contract.ContractName,
					contract.ContractAddress,
					contract.Selector,
				)

				err = verifyContract(
					cmd.Context(),
					c.lggr,
					domain,
					contract.Selector,
					contract.Environment,
					contractDirectory,
					commit,
					contract.ContractAddress,
					contract.CompilerVersion,
					contract.OptimizerRuns,
					contract.ContractName,
				)
				if err != nil {
					return fmt.Errorf("failed to verify #%d contract %s: %w", i, contract.ContractAddress, err)
				}

				// May need to add a sleep in between calls, as etherscan API is rate limited.
				time.Sleep(time.Millisecond * 750)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&contracts, "contracts", "contracts.example.toml",
		"Сontract list for verification")
	cmd.Flags().StringVarP(&contractDirectory, "dir", "d", "",
		"Path to the git directory containing the contract. This should normally be wherever the foundry.toml file is located")
	cmd.Flags().StringVar(&commit, "commit", "",
		"The commit to use in the directory")

	_ = cmd.MarkFlagRequired("contracts")
	_ = cmd.MarkFlagRequired("dir")
	_ = cmd.MarkFlagRequired("commit")

	return &cmd
}
