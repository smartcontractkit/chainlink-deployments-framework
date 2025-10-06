package mcmsv2

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/rpc"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldf_chains "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/chains"
	cldf_config "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/upf"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli/mcmsv2/layout"
)

const (
	proposalKindFlag        = "proposalKind"
	indexFlag               = "index"
	forkFlag                = "fork"
	defaultAdvanceTime      = 36000 // In seconds - defaulting to 10 hours
	defaultProposalValidity = 72 * time.Hour
)

type commonFlagsv2 struct {
	proposalPath    string
	proposalKindStr string
	environmentStr  string
	chainSelector   uint64
	fork            bool
}

type cfgv2 struct {
	kind             types.ProposalKind
	proposal         mcms.Proposal
	timelockProposal *mcms.TimelockProposal // nil if not a timelock proposal
	chainSelector    uint64
	blockchains      chain.BlockChains
	envStr           string
	env              cldf.Environment
	forkedEnv        cldfenvironment.ForkedEnvironment
	fork             bool
	proposalCtx      analyzer.ProposalContext
}

func BuildMCMSv2Cmd(lggr logger.Logger, domain cldf_domain.Domain, proposalContextProvider analyzer.ProposalContextProvider) *cobra.Command {
	var (
		proposalPath       string
		proposalKindStr    string
		environmentStr     string
		chainSelector      uint64
		validProposalKinds = []string{string(types.KindProposal), string(types.KindTimelockProposal)}
	)

	cmd := cobra.Command{
		Use:   "mcmsv2",
		Short: "Manage MCMS proposals",
		Long:  ``,
	}
	stdErrLogger, err := newCLIStdErrLogger()
	if err != nil {
		fmt.Println("failed to create stdErr logger")
		os.Exit(1)
	}
	cmd.PersistentFlags().StringVarP(&proposalPath, proposalPathFlag, "p", "", "Absolute file path containing the proposal to be submitted")
	cmd.PersistentFlags().StringVarP(&proposalKindStr, proposalKindFlag, "k", string(types.KindTimelockProposal), fmt.Sprintf("The type of proposal being ingested '%v'", validProposalKinds))
	cmd.PersistentFlags().StringVarP(&environmentStr, environmentFlag, "e", "", "Deployment environment (required)")
	cmd.PersistentFlags().Uint64VarP(&chainSelector, chainSelectorFlag, "s", 0, "Chain selector used determine target chain")

	panicErr(cmd.MarkPersistentFlagRequired(proposalPathFlag))
	panicErr(cmd.MarkPersistentFlagRequired(environmentFlag))
	panicErr(cmd.MarkPersistentFlagRequired(chainSelectorFlag))

	cmd.AddCommand(buildMCMSCheckQuorumv2Cmd(lggr, domain))
	cmd.AddCommand(buildExecuteChainv2Cmd(lggr, domain, proposalContextProvider))
	cmd.AddCommand(buildExecuteOperationv2Cmd(lggr, domain, proposalContextProvider))
	cmd.AddCommand(buildSetRootv2Cmd(lggr, domain, proposalContextProvider))
	cmd.AddCommand(buildGetOpCountV2Cmd(lggr, domain))
	cmd.AddCommand(buildRunTimelockIsPendingV2Cmd(lggr, domain))
	cmd.AddCommand(buildRunTimelockIsReadyToExecuteV2Cmd(lggr, domain))
	cmd.AddCommand(buildRunTimelockIsDoneV2Cmd(lggr, domain))
	cmd.AddCommand(buildRunTimelockIsOperationPendingV2Cmd(lggr, domain))
	cmd.AddCommand(buildRunTimelockIsOperationReadyToExecuteV2Cmd(lggr, domain))
	cmd.AddCommand(buildRunTimelockIsOperationDoneV2Cmd(lggr, domain))
	cmd.AddCommand(buildTimelockExecuteChainV2Cmd(lggr, domain, proposalContextProvider))
	cmd.AddCommand(buildTimelockExecuteOperationV2Cmd(lggr, domain, proposalContextProvider))
	cmd.AddCommand(buildMCMSv2AnalyzeProposalCmd(stdErrLogger, domain, proposalContextProvider))
	cmd.AddCommand(buildMCMSv2ConvertUpf(stdErrLogger, domain, proposalContextProvider))
	cmd.AddCommand(buildMCMSv2ResetProposalCmd(stdErrLogger, domain, proposalContextProvider))

	// fork flag is only used internally by buildExecuteForkCommand
	cmd.PersistentFlags().BoolP(forkFlag, "f", false, "Run the command on forked environment (EVM)")
	cmd.AddCommand(buildExecuteForkCommand(lggr, domain, proposalContextProvider))

	return &cmd
}

func newCLIStdErrLogger() (logger.Logger, error) {
	lggr, err := logger.NewWith(func(cfg *zap.Config) {
		*cfg = zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(zapcore.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.OutputPaths = []string{"stderr"} // send logs to stderr
		cfg.ErrorOutputPaths = []string{"stderr"}
	})
	if err != nil {
		return nil, err
	}

	return lggr, nil
}

func buildMCMSCheckQuorumv2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	return &cobra.Command{
		Use:   "check-quorum",
		Short: "Determines whether the provided signatures meet the quorum to set the root",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newCfgv2(lggr, cmd, domain, nil)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			inspector, err := getInspectorFromChainSelector(*cfg)
			if err != nil {
				return fmt.Errorf("can't create inspector: %w", err)
			}

			signable, err := mcms.NewSignable(&cfg.proposal, map[types.ChainSelector]sdk.Inspector{
				types.ChainSelector(cfg.chainSelector): inspector,
			})
			if err != nil {
				return fmt.Errorf("error creating signable: %w", err)
			}

			quorumMet, err := signable.CheckQuorum(cmd.Context(), types.ChainSelector(cfg.chainSelector))
			if err != nil {
				return fmt.Errorf("error checking quorum: %w", err)
			}
			if quorumMet {
				lggr.Info("Signature Quorum met!")
			} else {
				lggr.Info("Signature Quorum not met!")
				return errors.New("signature Quorum not met")
			}

			return nil
		},
	}
}

func buildExecuteOperationv2Cmd(lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider) *cobra.Command {
	var index int

	cmd := cobra.Command{
		Use:   "execute-operation",
		Short: "Executes specified operation by the provided index for a given chain in an MCMS Proposal. Root must be set first.",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			executable, err := createExecutable(cfg)
			if err != nil {
				return fmt.Errorf("error converting proposal to executable: %w", err)
			}
			if cfg.fork {
				lggr.Info("Fork mode is on, all transactions will be executed on a forked chain")
			}

			if index >= len(cfg.proposal.Operations) {
				return fmt.Errorf("index %d is not found in operations", index)
			}

			op := cfg.proposal.Operations[index]
			if op.ChainSelector != types.ChainSelector(cfg.chainSelector) {
				return fmt.Errorf("operation %d is not for chain %d", index, cfg.chainSelector)
			}

			tx, err := executable.Execute(cmd.Context(), index)
			if err != nil {
				err = cldf.DecodeErr(bindings.ManyChainMultiSigABI, err)
				return fmt.Errorf("error executing chain op %d: %w", index, err)
			}
			lggr.Infof("Transaction sent: %s", tx.Hash)

			err = confirmTransaction(cmd.Context(), lggr, tx, cfg)
			if err != nil {
				return fmt.Errorf("unable to confirm execute(%d) transaction: %w", index, err)
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&index, "index", 0, "Index of the operation to execute")

	return &cmd
}

func buildSetRootv2Cmd(lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "set-root",
		Short: "Sets the Merkle Root on the MCM Contract",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			return setRootCommand(cmd.Context(), lggr, cfg)
		},
	}
}

func buildExecuteChainv2Cmd(lggr logger.Logger, domain cldf_domain.Domain, proposalContextProvider analyzer.ProposalContextProvider) *cobra.Command {
	var skipNonceErrors bool
	cmd := &cobra.Command{
		Use:   "execute-chain",
		Short: "Executes all operations for a given chain in an MCMS Proposal. Root must be set first.",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newCfgv2(lggr, cmd, domain, proposalContextProvider)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			return executeChainCommand(cmd.Context(), lggr, cfg, skipNonceErrors)
		},
	}
	cmd.Flags().BoolVar(&skipNonceErrors, "skip-nonce-errors", false, "Skip any incorrect nonce errors (useful when retrying a half executed proposal)")

	return cmd
}

func buildGetOpCountV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	return &cobra.Command{
		Use:   "get-op-count",
		Short: "Gets op count",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			inspector, err := getInspectorFromChainSelector(*cfg)
			if err != nil {
				return err
			}

			opCount, err := inspector.GetOpCount(cmd.Context(), cfg.proposal.ChainMetadata[types.ChainSelector(cfg.chainSelector)].MCMAddress)
			if err != nil {
				return err
			}

			cmd.Println(opCount)

			return nil
		},
	}
}

func buildRunTimelockIsPendingV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-timelock-pending",
		Short: "Checks if all operations in a timelock proposal are pending for the given chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfgv2, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			err = executable.IsChainPending(cmd.Context(), types.ChainSelector(cfgv2.chainSelector))
			if err != nil {
				return fmt.Errorf("operations from chain %v are not pending: %w", cfgv2.chainSelector, err)
			}

			lggr.Infof("All operations from chain %v are pending", cfgv2.chainSelector)

			return nil
		},
	}

	return cmd
}

func buildRunTimelockIsReadyToExecuteV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-timelock-ready",
		Short: "Checks if all operations in a timelock proposal are ready for execution for the given chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfgv2, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			err = executable.IsChainReady(cmd.Context(), types.ChainSelector(cfgv2.chainSelector))
			if err != nil {
				return fmt.Errorf("operations from chain %v are not ready for execution: %w", cfgv2.chainSelector, err)
			}

			lggr.Infof("All operations from chain %v are ready for execution", cfgv2.chainSelector)

			return nil
		},
	}

	return cmd
}

func buildRunTimelockIsDoneV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "is-timelock-done",
		Short: "Checks if all operations in a timelock proposal are done executing for the given chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfgv2, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			err = executable.IsChainDone(cmd.Context(), types.ChainSelector(cfgv2.chainSelector))
			if err != nil {
				return fmt.Errorf("operations from chain %v are not done: %w", cfgv2.chainSelector, err)
			}

			lggr.Infof("All operations from chain %v are done", cfgv2.chainSelector)

			return nil
		},
	}

	return cmd
}

func buildRunTimelockIsOperationPendingV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use:   "is-timelock-operation-pending",
		Short: "Checks if the operation with the given index in a timelock proposal is pending",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgv2, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}
			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}
			if index >= len(cfgv2.timelockProposal.Operations) {
				return fmt.Errorf("invalid index (# of operations: %v)", len(cfgv2.timelockProposal.Operations))
			}
			if uint64(cfgv2.timelockProposal.Operations[index].ChainSelector) != cfgv2.chainSelector {
				return fmt.Errorf("mismatching chain selector: %v vs %v)",
					cfgv2.timelockProposal.Operations[index].ChainSelector, cfgv2.chainSelector)
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			err = executable.IsOperationPending(cmd.Context(), index)
			if err != nil {
				return fmt.Errorf("operation %v is not pending: %w", index, err)
			}

			lggr.Infof("Operation %v is pending", index)

			return nil
		},
	}

	cmd.Flags().IntVar(&index, indexFlag, 0, "Index of the operation to execute")

	return cmd
}

func buildRunTimelockIsOperationReadyToExecuteV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use:   "is-timelock-operation-ready",
		Short: "Checks if an operation in a timelock proposal is ready for execution for the given index",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgv2, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}
			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}
			if index >= len(cfgv2.timelockProposal.Operations) {
				return fmt.Errorf("invalid index (# of operations: %v)", len(cfgv2.timelockProposal.Operations))
			}
			if uint64(cfgv2.timelockProposal.Operations[index].ChainSelector) != cfgv2.chainSelector {
				return fmt.Errorf("mismatching chain selector: %v vs %v)",
					cfgv2.timelockProposal.Operations[index].ChainSelector, cfgv2.chainSelector)
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			if err := executable.IsOperationReady(context.Background(), index); err != nil {
				return fmt.Errorf("operation %v is not ready: %w", index, err)
			}

			lggr.Infof("Operations %v is ready for execution", index)

			return nil
		},
	}

	cmd.Flags().IntVar(&index, indexFlag, 0, "Index of the operation to execute")

	return cmd
}

func buildRunTimelockIsOperationDoneV2Cmd(lggr logger.Logger, domain cldf_domain.Domain) *cobra.Command {
	var index int

	cmd := &cobra.Command{
		Use:   "is-timelock-operation-done",
		Short: "Checks if the operation with the given index in a timelock proposal is done executing",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgv2, err := newCfgv2(lggr, cmd, domain, nil, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}
			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}
			if index >= len(cfgv2.timelockProposal.Operations) {
				return fmt.Errorf("invalid index (# of operations: %v)", len(cfgv2.timelockProposal.Operations))
			}
			if uint64(cfgv2.timelockProposal.Operations[index].ChainSelector) != cfgv2.chainSelector {
				return fmt.Errorf("mismatching chain selector: %v vs %v)",
					cfgv2.timelockProposal.Operations[index].ChainSelector, cfgv2.chainSelector)
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			err = executable.IsOperationDone(cmd.Context(), index)
			if err != nil {
				return fmt.Errorf("operation %v is not done: %w", index, err)
			}

			lggr.Infof("Operation %v is done", index)

			return nil
		},
	}

	cmd.Flags().IntVar(&index, indexFlag, 0, "Index of the operation to execute")

	return cmd
}

func buildTimelockExecuteChainV2Cmd(lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "timelock-execute-chain",
		Short: "Executes all operations for a given chain in timelock proposal.",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfgv2, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			return timelockExecuteChainCommand(cmd.Context(), lggr, cfgv2, domain)
		},
	}
}

func buildTimelockExecuteOperationV2Cmd(lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider) *cobra.Command {
	var index int

	cmd := cobra.Command{
		Use:   "timelock-execute-operation",
		Short: "Executes specified operation by the provided index for a given chain in an MCMS Proposal. Root must be set first.",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfgv2, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}

			executable, err := createTimelockExecutable(cmd.Context(), cfgv2)
			if err != nil {
				return fmt.Errorf("failed to create TimelockExecutable: %w", err)
			}

			executeOptions, err := timelockExecuteOptions(cmd.Context(), lggr, domain, cfgv2)
			if err != nil {
				return fmt.Errorf("failed to get timelock execute options: %w", err)
			}

			result, err := executable.Execute(cmd.Context(), index, executeOptions...)
			if err != nil {
				return fmt.Errorf("failed to execute operation %d: %w", index, err)
			}

			lggr.Infof("Operation %d executed successfully: %s\n", index, result)

			return nil
		},
	}

	cmd.Flags().IntVar(&index, indexFlag, 0, "Index of the operation to execute")

	return &cmd
}

// buildExecuteForkCommand is a command that can be used only in forked environment
// it calls "set-root", "execute-chain", "execute-timelock-chain" to verify
// that a signed proposal can be applied on a taget network
// see rewind-blocks param, by default we are rewinding forked chains to speed up verification of Timelock proposals
func buildExecuteForkCommand(lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider) *cobra.Command {
	var testSigner bool

	cmd := &cobra.Command{
		Use:   "execute-fork",
		Short: "Executes set-root, execute-chain and execute-timelock-chain operations for forked environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmd.Flags().Set(forkFlag, "true"); err != nil {
				return fmt.Errorf("failed to set fork flag for buildExecuteForkCommand command: %w", err)
			}
			chainSelector, err := cmd.Flags().GetUint64(chainSelectorFlag)
			if err != nil {
				return fmt.Errorf("error getting selector flag: %w", err)
			}
			family, err := chainsel.GetSelectorFamily(chainSelector)
			if err != nil {
				return fmt.Errorf("failed to get selector family: %w", err)
			}
			if family != chainsel.FamilyEVM {
				lggr.Infof("Skipping fork execution: chain selector %d is not EVM. Family is %s", chainSelector, family)
				return nil // don’t fail, just exit cleanly
			}
			cfg, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			// get the chain URL, chain ID and MCM contract address
			url := cfg.forkedEnv.ChainConfigs[cfg.chainSelector].HTTPRPCs[0].External
			anvilClient := rpc.New(url, nil)
			chainID := cfg.forkedEnv.ChainConfigs[cfg.chainSelector].ChainID
			mcmsAddr := cfg.proposal.ChainMetadata[types.ChainSelector(cfg.chainSelector)].MCMAddress

			ctx, cancel := context.WithTimeout(cmd.Context(), 300*time.Second)
			defer cancel()
			if testSigner {
				if lerr := layout.SetMCMSigner(
					ctx,
					lggr,
					layout.MCMSLayout,
					blockchain.DefaultAnvilPrivateKey,
					blockchain.DefaultAnvilPublicKey,
					blockchain.DefaultAnvilPublicKey,
					url,
					chainID,
					mcmsAddr,
				); lerr != nil {
					return fmt.Errorf("failed to set signer: %w", lerr)
				}
			}
			// Override signatures for proposal
			privKey, err := crypto.HexToECDSA(blockchain.DefaultAnvilPrivateKey)
			if err != nil {
				return fmt.Errorf("failed to create private key: %w", err)
			}
			timelockAddress := common.HexToAddress(cfg.timelockProposal.TimelockAddresses[types.ChainSelector(cfg.chainSelector)])

			if err = overwriteProposalSignatureWithTestKey(cfg, privKey); err != nil {
				return fmt.Errorf("failed to overwrite proposal signature: %w", err)
			}

			// set root
			// TODO: improve error decoding on the mcms lib for set root.
			err = setRootCommand(ctx, lggr, cfg)
			if err != nil {
				return fmt.Errorf("failed to set root: %w", err)
			}
			lggr.Info("Root set successfully")
			// TODO: improve error decoding on the mcms lib for set root.
			err = executeChainCommand(ctx, lggr, cfg, true)
			if err != nil {
				return fmt.Errorf("failed to execute chain: %w", err)
			}
			lggr.Info("MCMs execute() success")
			lggr.Info("Waiting for the chain to be mined before executing timelock chain command")
			if err = anvilClient.EVMIncreaseTime(defaultAdvanceTime); err != nil {
				return fmt.Errorf("failed to increase time: %w", err)
			}
			if err = anvilClient.AnvilMine([]interface{}{1}); err != nil {
				return fmt.Errorf("failed to mine block: %w", err)
			}

			if cfg.timelockProposal.Action != types.TimelockActionSchedule {
				lggr.Infof("Proposal has type %s, skipping executing timelock chain command", cfg.timelockProposal.Action)
				return nil
			}

			lggr.Info("Executing timelock chain command")
			err = timelockExecuteChainCommand(ctx, lggr, cfg, domain)
			if err != nil {
				lggr.Warnw("Timelock execute failed, starting calling individual ops for debugging", "err", err)
				envdir := domain.EnvDir(cfg.envStr)
				ab, errAb := envdir.AddressBook()
				if errAb != nil {
					return fmt.Errorf("failed to load address book: %w", err)
				}
				if derr := diagnoseTimelockRevert(ctx, lggr, anvilClient.URL, cfg.chainSelector, cfg.timelockProposal.Operations, timelockAddress, ab, cfg.proposalCtx); derr != nil {
					lggr.Errorw("Diagnosis results", "err", derr)
					return fmt.Errorf("failed to timelock execute chain: %w", derr)
				}

				return fmt.Errorf("failed to timelock execute chain: %w", err)
			}
			lggr.Info("Timelock execute chain success")

			return nil
		},
	}
	cmd.Flags().BoolVar(&testSigner, "test-signer", false, "Use a test signer key")

	return cmd
}

func buildMCMSv2AnalyzeProposalCmd(
	lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider,
) *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "analyze-proposal",
		Short: "Analyze proposal and provide human readable output",
		Long:  ``,
		PreRun: func(command *cobra.Command, args []string) {
			// chainSelector is optional for AnalyzeProposal; trick cobra into thinking it's been set
			command.InheritedFlags().Lookup(chainSelectorFlag).Changed = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create config
			cfgv2, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be have non-nil *TimelockProposal")
			}

			var analyzedProposal string
			if cfgv2.timelockProposal != nil {
				analyzedProposal, err = analyzer.DescribeTimelockProposal(cfgv2.proposalCtx, cfgv2.timelockProposal)
			} else {
				analyzedProposal, err = analyzer.DescribeProposal(cfgv2.proposalCtx, &cfgv2.proposal)
			}
			if err != nil {
				return fmt.Errorf("failed to describe proposal: %w", err)
			}

			if outputFile == "" {
				fmt.Println(analyzedProposal)
			} else {
				err := os.WriteFile(outputFile, []byte(analyzedProposal), 0o600)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
	cmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		command.Flags().MarkHidden(chainSelectorFlag) //nolint:errcheck
		command.Parent().HelpFunc()(command, args)
	})

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file to write analyze result")

	return cmd
}

func buildMCMSv2ResetProposalCmd(
	lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider,
) *cobra.Command {
	var overrideRoot bool
	var proposalPath string
	cmd := &cobra.Command{
		Use:   "reset-proposal",
		Short: "Updates proposal with latest on-chain op counts and resets signatures",
		Long:  ``,
		PreRun: func(command *cobra.Command, args []string) {
			// chainSelector is optional for reset proposal; trick cobra into thinking it's been set
			command.InheritedFlags().Lookup(chainSelectorFlag).Changed = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgv2, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}
			overrideRoot, err = cmd.Flags().GetBool("override-root")
			if err != nil {
				return fmt.Errorf("error getting override-root flag: %w", err)
			}
			timelockProposal := cfgv2.timelockProposal
			if timelockProposal == nil {
				return errors.New("null TimelockProposal")
			}

			timelockProposal.ValidUntil = uint32(time.Now().Add(defaultProposalValidity).Unix()) //nolint:gosec // G404: time-based validity is acceptable for test signatures

			for selector := range cfgv2.proposal.ChainMetadata {
				cfgv2.chainSelector = uint64(selector)
				inspector, errInspect := getInspectorFromChainSelector(*cfgv2)
				if errInspect != nil {
					return fmt.Errorf("error getting inspector from chain selector: %w", errInspect)
				}
				opCount, errOpCount := inspector.GetOpCount(cmd.Context(), timelockProposal.ChainMetadata[types.ChainSelector(cfgv2.chainSelector)].MCMAddress)
				if errOpCount != nil {
					return errOpCount
				}
				metadata := timelockProposal.ChainMetadata[types.ChainSelector(cfgv2.chainSelector)]
				metadata.StartingOpCount = opCount
				timelockProposal.ChainMetadata[types.ChainSelector(cfgv2.chainSelector)] = metadata
			}

			timelockProposal.Signatures = nil
			if overrideRoot {
				timelockProposal.OverridePreviousRoot = true
			}

			// Write file to proposalPath
			pathFromFlag, err := cmd.Flags().GetString("proposal")
			if err == nil && pathFromFlag != "" {
				proposalPath = pathFromFlag
			}
			if proposalPath == "" {
				return errors.New("proposalPath flag is required (path to write the updated proposal)")
			}
			w, err := os.Create(proposalPath)
			if err != nil {
				return fmt.Errorf("error creating proposal file: %w", err)
			}

			err = mcms.WriteTimelockProposal(w, timelockProposal)
			if err != nil {
				return fmt.Errorf("error writing proposal to file: %w", err)
			}
			lggr.Infow("Successfully reset proposal", "path", proposalPath)

			return nil
		},
	}
	cmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		command.Flags().MarkHidden(chainSelectorFlag) //nolint:errcheck
		command.Parent().HelpFunc()(command, args)
	})

	cmd.Flags().Bool("override-root", overrideRoot, "Override the root of the MCMs contracts in the proposal")

	return cmd
}

func buildMCMSv2ConvertUpf(
	lggr logger.Logger, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider,
) *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "convert-upf",
		Short: "Convert proposal to UPF (universal proposal format)",
		Long:  ``,
		PreRun: func(command *cobra.Command, args []string) {
			// chainSelector is optional for Convert to UPF; trick cobra into thinking it's been set
			command.InheritedFlags().Lookup(chainSelectorFlag).Changed = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgv2, err := newCfgv2(lggr, cmd, domain, proposalCtxProvider, acceptExpiredProposal)
			if err != nil {
				return fmt.Errorf("error creating config: %w", err)
			}

			if cfgv2.timelockProposal == nil {
				return errors.New("expected proposal to be a TimelockProposal")
			}

			// Get signers for the proposal
			signers, err := getProposalSigners(*cfgv2, cmd.Context(), &cfgv2.proposal)
			if err != nil {
				return fmt.Errorf("failed to get proposal signers: %w", err)
			}

			var convertedProposal string

			if cfgv2.timelockProposal != nil {
				convertedProposal, err = upf.UpfConvertTimelockProposal(cfgv2.proposalCtx, cfgv2.timelockProposal, &cfgv2.proposal, signers)
			} else {
				convertedProposal, err = upf.UpfConvertProposal(cfgv2.proposalCtx, &cfgv2.proposal, signers)
			}
			if err != nil {
				return fmt.Errorf("failed to convert proposal to UPF format: %w", err)
			}

			if outputFile == "" {
				fmt.Println(convertedProposal)
			} else {
				err := os.WriteFile(outputFile, []byte(convertedProposal), 0o600)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
	cmd.SetHelpFunc(func(command *cobra.Command, args []string) {
		command.Flags().MarkHidden(chainSelectorFlag) //nolint:errcheck
		command.Parent().HelpFunc()(command, args)
	})

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "File path where the converted file will be saved")

	return cmd
}

// overwriteProposalSignatureWithTestKey overwrites the proposal's signature with a test key signature.
func overwriteProposalSignatureWithTestKey(cfg *cfgv2, testKey *ecdsa.PrivateKey) error {
	p := &cfg.proposal
	// Override the proposal fields that are used in the signing hash to ensure no errors occur related to those.
	p.ValidUntil = uint32(time.Now().Add(5 * time.Hour).Unix()) //nolint:gosec // G404: time-based validity is acceptable for test signatures
	p.Signatures = nil

	p.OverridePreviousRoot = true

	inspector, err := getInspectorFromChainSelector(*cfg)
	if err != nil {
		return fmt.Errorf("error getting inspector from chain selector: %w", err)
	}
	signable, errSignable := mcms.NewSignable(p, map[types.ChainSelector]sdk.Inspector{
		types.ChainSelector(cfg.chainSelector): inspector,
	})
	if errSignable != nil {
		return fmt.Errorf("error creating signable: %w", errSignable)
	}

	signature, err := signable.SignAndAppend(mcms.NewPrivateKeySigner(testKey))
	p.Signatures = []types.Signature{signature}
	if err != nil {
		return fmt.Errorf("error creating signable: %w", err)
	}

	return nil
}

func newRandomSalt() *common.Hash {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	h := common.BytesToHash(b[:])

	return &h
}

func parseCommonFlagsv2(cmdFlags *pflag.FlagSet) (commonFlagsv2, error) {
	var flags commonFlagsv2
	var err error
	flags.proposalPath, err = cmdFlags.GetString(proposalPathFlag)
	if err != nil {
		return flags, fmt.Errorf("error getting proposal flag: %w", err)
	}
	flags.proposalKindStr, err = cmdFlags.GetString(proposalKindFlag)
	if err != nil {
		return flags, fmt.Errorf("error getting proposalKind flag: %w", err)
	}
	flags.environmentStr, err = cmdFlags.GetString(environmentFlag)
	if err != nil {
		return flags, fmt.Errorf("error getting environment flag: %w", err)
	}
	flags.chainSelector, err = cmdFlags.GetUint64(chainSelectorFlag)
	if err != nil {
		return flags, fmt.Errorf("error getting selector flag: %w", err)
	}
	flags.fork, err = cmdFlags.GetBool(forkFlag)
	if err != nil {
		return flags, fmt.Errorf("error getting fork flag: %w", err)
	}

	// Validate proposal kind
	if _, exists := types.StringToProposalKind[flags.proposalKindStr]; !exists {
		return flags, fmt.Errorf("unknown proposal kind '%s'", flags.proposalKindStr)
	}

	return flags, nil
}

func newCfgv2(lggr logger.Logger, cmd *cobra.Command, domain cldf_domain.Domain, proposalCtxProvider analyzer.ProposalContextProvider, opts ...any) (*cfgv2, error) {
	flags, err := parseCommonFlagsv2(cmd.Flags())
	if err != nil {
		return nil, fmt.Errorf("error parsing common flags: %w", err)
	}

	proposalKind, exists := types.StringToProposalKind[flags.proposalKindStr]
	if !exists {
		return nil, fmt.Errorf("unknown proposal type '%s'", flags.proposalKindStr)
	}

	fileProposal, err := mcms.LoadProposal(proposalKind, flags.proposalPath)
	if err != nil {
		if !slices.Contains(opts, acceptExpiredProposal) || !isProposalExpiredError(err) {
			return nil, fmt.Errorf("error loading proposal: %w", err)
		}
	}

	var mcmsProposal *mcms.Proposal
	var timelockCastedProposal *mcms.TimelockProposal = nil
	if proposalKind == types.KindTimelockProposal {
		// convert proposal
		timelockCastedProposal = fileProposal.(*mcms.TimelockProposal)
		if flags.fork && timelockCastedProposal.Action == types.TimelockActionSchedule {
			timelockCastedProposal.SaltOverride = newRandomSalt()
		}

		// construct converters for each chain
		var fam string
		converters := make(map[types.ChainSelector]sdk.TimelockConverter)
		for chain := range timelockCastedProposal.ChainMetadata {
			fam, err = types.GetChainSelectorFamily(chain)
			if err != nil {
				return nil, fmt.Errorf("error getting chain family: %w", err)
			}

			var converter sdk.TimelockConverter
			switch fam {
			case chainsel.FamilyEVM:
				converter = &evm.TimelockConverter{}
			case chainsel.FamilySolana:
				converter = solana.TimelockConverter{}
			case chainsel.FamilyAptos:
				converter = aptos.NewTimelockConverter()
			case chainsel.FamilySui:
				converter, err = sui.NewTimelockConverter()
				if err != nil {
					return nil, fmt.Errorf("error creating Sui timelock converter: %w", err)
				}
			default:
				return nil, fmt.Errorf("unsupported chain family %s", fam)
			}

			converters[chain] = converter
		}

		var convertedProposal mcms.Proposal
		convertedProposal, _, err = timelockCastedProposal.Convert(cmd.Context(), converters)
		if err != nil {
			return nil, fmt.Errorf("error converting timelock proposal: %w", err)
		}

		mcmsProposal = &convertedProposal
	} else {
		mcmsProposal = fileProposal.(*mcms.Proposal)
	}

	cfg := &cfgv2{
		proposal:         *mcmsProposal,
		timelockProposal: timelockCastedProposal,
		kind:             proposalKind,
		chainSelector:    flags.chainSelector,
		envStr:           flags.environmentStr,
		fork:             flags.fork,
	}

	chainSelectors := make([]uint64, len(cfg.proposal.ChainSelectors()))
	if cfg.chainSelector != 0 {
		chainSelectors = []uint64{cfg.chainSelector}
	} else {
		for i, selector := range cfg.proposal.ChainSelectors() {
			chainSelectors[i] = uint64(selector)
		}
	}

	if proposalCtxProvider != nil {
		// Load Environment and proposal ctx (for error decoding and proposal analysis)
		env, err := cldfenvironment.Load(cmd.Context(), domain, cfg.envStr,
			cldfenvironment.WithLogger(lggr),
			cldfenvironment.OnlyLoadChainsFor(chainSelectors), cldfenvironment.WithoutJD())
		if err != nil {
			return nil, fmt.Errorf("error loading environment: %w", err)
		}
		cfg.env = env
		proposalCtx, err := proposalCtxProvider(env)
		if err != nil {
			return nil, fmt.Errorf("failed to provide proposal analysis context: %w", err)
		}
		cfg.proposalCtx = proposalCtx
	}

	if flags.fork {
		// we should load the environment to get proper forked chain URLs
		cfgSelectors := []uint64{cfg.chainSelector}
		forkedEnv, err := cldfenvironment.LoadFork(
			cmd.Context(),
			domain,
			flags.environmentStr,
			nil,
			cldfenvironment.WithLogger(lggr),
			cldfenvironment.OnlyLoadChainsFor(cfgSelectors),
			cldfenvironment.WithAnvilKeyAsDeployer(),
			cldfenvironment.WithoutJD(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load forked environment: %w", err)
		}
		cfg.forkedEnv = forkedEnv
		cfg.blockchains = forkedEnv.BlockChains
	} else {
		// FIXME: once DX-956 is done, restrict the list of chain selectors when
		// `flag.ChainSelector` is set so that we avoid the overhead of
		// loading _all_ chains
		var chainSelectors []uint64
		for chainSelector := range mcmsProposal.ChainMetadata {
			chainSelectors = append(chainSelectors, uint64(chainSelector))
		}

		config, err := cldf_config.Load(domain, flags.environmentStr, lggr)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}

		chains, err := cldf_chains.LoadChains(
			cmd.Context(),
			lggr,
			config,
			chainSelectors,
		)
		if err != nil {
			return nil, fmt.Errorf("error loading chains: %w", err)
		}
		cfg.blockchains = chains
	}

	return cfg, nil
}

var acceptExpiredProposal any

func isProposalExpiredError(err error) bool {
	var ivuerr *mcms.InvalidValidUntilError
	return errors.As(err, &ivuerr)
}

// isNonceError checks if the error is a nonce error for the given chain selector.
func isNonceError(rawErr error, selector uint64) (bool, error) {
	family, famErr := chainsel.GetSelectorFamily(selector)
	if famErr != nil {
		return false, famErr
	}

	switch family {
	case chainsel.FamilyEVM:
		decodedErr := cldf.DecodeErr(bindings.ManyChainMultiSigABI, rawErr)
		// Check if the error contains PostOpCountReached
		if strings.Contains(decodedErr.Error(), "PostOpCountReached") {
			return true, nil
		}

	case chainsel.FamilySolana:
		// Check if the error contains WrongNonce or PostOpCountReached
		if strings.Contains(rawErr.Error(), "WrongNonce") || strings.Contains(rawErr.Error(), "PostOpCountReached") {
			return true, nil
		}
	default:
		return false, nil
	}

	return false, nil
}

func executeChainCommand(ctx context.Context, lggr logger.Logger, cfg *cfgv2, skipNonceErrors bool) error {
	executable, err := createExecutable(cfg)
	if err != nil {
		return fmt.Errorf("error converting proposal to executable: %w", err)
	}
	if cfg.fork {
		lggr.Info("Fork mode is on, all transactions will be executed on a forked chain")
	}

	for i, op := range cfg.proposal.Operations {
		// TODO; consider multi-chain support
		if op.ChainSelector != types.ChainSelector(cfg.chainSelector) {
			continue
		}

		tx, err := executable.Execute(ctx, i)
		if err != nil {
			lggr.Errorf("error executing operation %d: %s", i, err)
			if skipNonceErrors {
				nonceErr, errNonceCheck := isNonceError(err, cfg.chainSelector)
				if errNonceCheck != nil {
					return fmt.Errorf("error checking nonce error: %w", err)
				}
				if nonceErr {
					lggr.Warnf("Skipping nonce error for operation %d", i)
					continue
				}
			}
			family, familyErr := chainsel.GetSelectorFamily(uint64(op.ChainSelector))
			if familyErr != nil {
				lggr.Errorf("error getting chain family: %w", familyErr)
			}
			switch family {
			case chainsel.FamilyEVM:
				err = cldf.DecodeErr(bindings.ManyChainMultiSigABI, err)

				return fmt.Errorf("error executing chain op %d: %w", i, err)
			}

			return err
		}
		lggr.Infof("Transaction sent: %s", tx.Hash)

		err = confirmTransaction(ctx, lggr, tx, cfg)
		if err != nil {
			return fmt.Errorf("unable to confirm execute(%d) transaction: %w", i, err)
		}
	}

	return nil
}

func setRootCommand(ctx context.Context, lggr logger.Logger, cfg *cfgv2) error {
	if cfg.fork {
		lggr.Info("Fork mode is on, all transactions will be executed on a forked chain")
	}

	executable, err := createExecutable(cfg)
	if err != nil {
		return fmt.Errorf("error converting proposal to executable: %w", err)
	}

	tx, err := executable.SetRoot(ctx, types.ChainSelector(cfg.chainSelector))
	if err != nil {
		err = cldf.DecodeErr(bindings.ManyChainMultiSigABI, err)
		return fmt.Errorf("error setting root: %w", err)
	}

	err = confirmTransaction(ctx, lggr, tx, cfg)
	if err != nil {
		return fmt.Errorf("failed to confirm set root transaction: %w", err)
	}

	return nil
}

func timelockExecuteChainCommand(ctx context.Context, lggr logger.Logger, cfg *cfgv2, domain cldf_domain.Domain) error {
	if cfg.timelockProposal == nil {
		return errors.New("expected proposal to be have non-nil *TimelockProposal")
	}

	executable, err := createTimelockExecutable(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create TimelockExecutable: %w", err)
	}

	executeOptions, err := timelockExecuteOptions(ctx, lggr, domain, cfg)
	if err != nil {
		return fmt.Errorf("failed to get timelock execute options: %w", err)
	}

	for i := range cfg.timelockProposal.Operations {
		if uint64(cfg.timelockProposal.Operations[i].ChainSelector) == cfg.chainSelector {
			// Check if operation is done, if so, skip it
			if err := executable.IsOperationDone(ctx, i); err == nil {
				lggr.Warnf("Operation %d is already done, skipping...\n", i)
				continue
			}

			if err := executable.IsOperationReady(ctx, i); err != nil {
				return fmt.Errorf("operation %d is not ready to be executed: %w", i, err)
			}

			result, err := executable.Execute(ctx, i, executeOptions...)
			if err != nil {
				return fmt.Errorf("failed to execute operation %d: %w", i, err)
			}

			err = confirmTransaction(ctx, lggr, result, cfg)
			if err != nil {
				return fmt.Errorf("failed to confirm execute transaction: %w", err)
			}

			lggr.Infof("Operation %d executed successfully: %s\n", i, result)
		}
	}

	lggr.Infof("All operations executed successfully")

	return nil
}

func getExecutorWithChainOverride(cfg *cfgv2, chainSelector types.ChainSelector) (sdk.Executor, error) {
	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	encoders, err := cfg.proposal.GetEncoders()
	if err != nil {
		return nil, fmt.Errorf("error getting encoders: %w", err)
	}
	encoder, ok := encoders[chainSelector]
	if !ok {
		return nil, fmt.Errorf("unable to get encoder from proposal for chain selector %v", chainSelector)
	}

	switch family {
	case chainsel.FamilyEVM:
		evmEncoder, ok := encoder.(*evm.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type: %T", encoder)
		}
		chain := cfg.blockchains.EVMChains()[uint64(chainSelector)]

		return evm.NewExecutor(evmEncoder, chain.Client, chain.DeployerKey), nil

	case chainsel.FamilySolana:
		solanaEncoder, ok := encoder.(*solana.Encoder)
		if !ok {
			return nil, fmt.Errorf("invalid encoder type: %T", encoder)
		}
		chain := cfg.blockchains.SolanaChains()[uint64(chainSelector)]

		return solana.NewExecutor(solanaEncoder, chain.Client, *chain.DeployerKey), nil

	case chainsel.FamilyAptos:
		encoder, ok := encoder.(*aptos.Encoder)
		if !ok {
			return nil, fmt.Errorf("error getting encoder for chain %d", cfg.chainSelector)
		}
		role, err := aptosRoleFromProposal(cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		chain := cfg.blockchains.AptosChains()[uint64(chainSelector)]

		return aptos.NewExecutor(chain.Client, chain.DeployerSigner, encoder, *role), nil

	case chainsel.FamilySui:
		encoder, ok := encoder.(*sui.Encoder)
		if !ok {
			return nil, fmt.Errorf("error getting encoder for chain %d", cfg.chainSelector)
		}
		metadata, err := suiMetadataFromProposal(chainSelector, cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		chain := cfg.blockchains.SuiChains()[uint64(chainSelector)]

		return sui.NewExecutor(chain.Client, chain.Signer, encoder, metadata.McmsPackageID, metadata.Role, cfg.timelockProposal.ChainMetadata[chainSelector].MCMAddress, metadata.AccountObj, metadata.RegistryObj, metadata.TimelockObj)
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}
}

func createExecutable(cfg *cfgv2) (*mcms.Executable, error) {
	executors := make(map[types.ChainSelector]sdk.Executor, len(cfg.proposal.ChainMetadata))
	for chainSelector := range cfg.proposal.ChainMetadata {
		if cfg.chainSelector == 0 || cfg.chainSelector == uint64(chainSelector) {
			executor, err := getExecutorWithChainOverride(cfg, chainSelector)
			if err != nil {
				return &mcms.Executable{}, fmt.Errorf("unable to get executor with chain override: %w", err)
			}
			executors[chainSelector] = executor
		}
	}

	return mcms.NewExecutable(&cfg.proposal, executors)
}

func getTimelockExecutorWithChainOverride(cfg *cfgv2, chainSelector types.ChainSelector) (sdk.TimelockExecutor, error) {
	family, err := types.GetChainSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	var executor sdk.TimelockExecutor
	switch family {
	case chainsel.FamilyEVM:
		chain := cfg.blockchains.EVMChains()[uint64(chainSelector)]

		executor = evm.NewTimelockExecutor(chain.Client, chain.DeployerKey)
	case chainsel.FamilySolana:
		chain := cfg.blockchains.SolanaChains()[uint64(chainSelector)]
		executor = solana.NewTimelockExecutor(chain.Client, *chain.DeployerKey)
	case chainsel.FamilyAptos:
		chain := cfg.blockchains.AptosChains()[uint64(chainSelector)]
		executor = aptos.NewTimelockExecutor(chain.Client, chain.DeployerSigner)
	case chainsel.FamilySui:
		chain := cfg.blockchains.SuiChains()[uint64(chainSelector)]
		metadata, err := suiMetadataFromProposal(chainSelector, cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		executor, err = sui.NewTimelockExecutor(chain.Client, chain.Signer, metadata.McmsPackageID, metadata.RegistryObj, metadata.AccountObj)
		if err != nil {
			return nil, fmt.Errorf("error creating sui timelock executor: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported chain family %s", family)
	}

	return executor, nil
}

func createTimelockExecutable(ctx context.Context, cfg *cfgv2) (*mcms.TimelockExecutable, error) {
	executors := make(map[types.ChainSelector]sdk.TimelockExecutor, len(cfg.timelockProposal.ChainMetadata))
	for chainSelector := range cfg.timelockProposal.ChainMetadata {
		if cfg.chainSelector != 0 && cfg.chainSelector != uint64(chainSelector) {
			continue
		}
		executor, err := getTimelockExecutorWithChainOverride(cfg, chainSelector)
		if err != nil {
			return &mcms.TimelockExecutable{}, err
		}
		executors[chainSelector] = executor
	}

	return mcms.NewTimelockExecutable(ctx, cfg.timelockProposal, executors)
}

var getInspectorFromChainSelector = func(cfg cfgv2) (sdk.Inspector, error) {
	fam, err := types.GetChainSelectorFamily(types.ChainSelector(cfg.chainSelector))
	if err != nil {
		return nil, fmt.Errorf("error getting chain family: %w", err)
	}

	var inspector sdk.Inspector
	switch fam {
	case chainsel.FamilyEVM:
		chain := cfg.blockchains.EVMChains()[cfg.chainSelector]
		inspector = evm.NewInspector(chain.Client)
	case chainsel.FamilySolana:
		chain := cfg.blockchains.SolanaChains()[cfg.chainSelector]
		inspector = solana.NewInspector(chain.Client)
	case chainsel.FamilyAptos:
		role, err := aptosRoleFromProposal(cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting aptos role from proposal: %w", err)
		}
		chain := cfg.blockchains.AptosChains()[cfg.chainSelector]
		inspector = aptos.NewInspector(chain.Client, *role)
	case chainsel.FamilySui:
		metadata, err := suiMetadataFromProposal(types.ChainSelector(cfg.chainSelector), cfg.timelockProposal)
		if err != nil {
			return nil, fmt.Errorf("error getting sui metadata from proposal: %w", err)
		}
		chain := cfg.blockchains.SuiChains()[cfg.chainSelector]
		inspector, err = sui.NewInspector(chain.Client, chain.Signer, metadata.McmsPackageID, metadata.Role)
		if err != nil {
			return nil, fmt.Errorf("error creating sui inspector: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported chain family %s", fam)
	}

	return inspector, nil
}

func confirmTransaction(ctx context.Context, lggr logger.Logger, tx types.TransactionResult, cfg *cfgv2) error {
	family, err := chainsel.GetSelectorFamily(cfg.chainSelector)
	if err != nil {
		return fmt.Errorf("error getting chain family: %w", err)
	}

	if family == chainsel.FamilyEVM {
		chain := cfg.blockchains.EVMChains()[cfg.chainSelector]
		block, err := chain.Confirm(tx.RawData.(*gethtypes.Transaction))
		if err == nil {
			lggr.Infof("Transaction %s confirmed in block %d", tx.Hash, block)
			return nil
		}
		rcpt, err := chain.Client.TransactionReceipt(ctx, common.HexToHash(tx.Hash))
		if err != nil {
			return fmt.Errorf("error getting transaction receipt for %s: %w", tx.Hash, err)
		}
		if rcpt != nil && rcpt.Status == 0 && cfg.proposalCtx != nil {
			// Decode via simulation to recover revert bytes
			if pretty, ok := tryDecodeTxRevertEVM(
				ctx,
				chain.Client,
				tx.RawData.(*gethtypes.Transaction),
				bindings.ManyChainMultiSigABI,
				rcpt.BlockNumber,
				cfg.proposalCtx); ok {
				return fmt.Errorf("tx %s reverted: %s", tx.Hash, pretty)
			}
		}

		return err
	}

	if family == chainsel.FamilyAptos {
		chain := cfg.blockchains.AptosChains()[cfg.chainSelector]
		err := chain.Confirm(tx.Hash)
		if err != nil {
			return err
		}
		lggr.Infof("Transaction %s confirmed", tx.Hash)
	}

	return nil
}

func getProposalSigners(
	cfgv2 cfgv2,
	ctx context.Context,
	proposal mcms.ProposalInterface,
) (map[types.ChainSelector][]common.Address, error) {
	chainMeta := proposal.ChainMetadatas()

	addresses := make(map[types.ChainSelector][]common.Address, len(chainMeta))
	for chainSelector, metadata := range chainMeta {
		cfgv2.chainSelector = uint64(chainSelector)
		inspector, err := getInspectorFromChainSelector(cfgv2)
		if err != nil {
			return nil, fmt.Errorf("get inspector for selector %d: %w", chainSelector, err)
		}

		config, err := inspector.GetConfig(ctx, metadata.MCMAddress)
		if err != nil {
			return nil, fmt.Errorf("get config for selector %d: %w", chainSelector, err)
		}

		addresses[chainSelector] = config.GetAllSigners()
	}

	return addresses, nil
}

func timelockExecuteOptions(
	ctx context.Context, lggr logger.Logger, _ cldf_domain.Domain, cfg *cfgv2,
) ([]mcms.Option, error) {
	options := []mcms.Option{}

	family, err := chainsel.GetSelectorFamily(cfg.chainSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get selector family: %w", err)
	}
	if family == chainsel.FamilyEVM {
		err := addCallProxyOption(ctx, lggr, cfg, &options)
		if err != nil {
			return options, fmt.Errorf("failed to add CallProxy option: %w", err)
		}
	}

	return options, nil
}

func addCallProxyOption(
	ctx context.Context, lggr logger.Logger, cfg *cfgv2, options *[]mcms.Option,
) error {
	timelockAddress, ok := cfg.timelockProposal.TimelockAddresses[types.ChainSelector(cfg.chainSelector)]
	if !ok {
		return fmt.Errorf("failed to find timelock address for chain selector %d", cfg.chainSelector)
	}

	chain, ok := cfg.blockchains.EVMChains()[cfg.chainSelector]
	if !ok {
		return fmt.Errorf("failed to find evm chain for selector %d", cfg.chainSelector)
	}

	timelockContract, err := bindings.NewRBACTimelock(common.HexToAddress(timelockAddress), chain.Client)
	if err != nil {
		return fmt.Errorf("failed to create timelock contract with address %v: %w", timelockAddress, err)
	}

	callOpts := &bind.CallOpts{Context: ctx}

	role, err := timelockContract.EXECUTORROLE(callOpts)
	if err != nil {
		return fmt.Errorf("failed to get executor role from timelock contract: %w", err)
	}
	memberCount, err := timelockContract.GetRoleMemberCount(callOpts, role)
	if err != nil {
		return fmt.Errorf("failed to get executor member count from timelock contract: %w", err)
	}
	for i := range memberCount.Int64() {
		executorAddress, ierr := timelockContract.GetRoleMember(callOpts, role, big.NewInt(i))
		if ierr != nil {
			return fmt.Errorf("failed to get executor address from timelock contract: %w", ierr)
		}

		// search for executor address in the datastore
		callProxyRefs := cfg.env.DataStore.Addresses().Filter(
			datastore.AddressRefByAddress(executorAddress.Hex()),
			datastore.AddressRefByChainSelector(cfg.chainSelector),
			datastore.AddressRefByType("CallProxy"))

		if len(callProxyRefs) > 0 {
			*options = append(*options, mcms.WithCallProxy(executorAddress.Hex()))
			return nil
		}

		// if not found, search in the addressbook
		addressesForChain, ierr := cfg.env.ExistingAddresses.AddressesForChain(cfg.chainSelector) //nolint:staticcheck
		if ierr != nil {
			lggr.Infof("unable to get addresses for chain %d in addressbook: %s", cfg.chainSelector, ierr.Error())
			continue // ignore error; some domains don't use the addressbook anymore
		}
		for address, typeAndVersion := range addressesForChain {
			if address == executorAddress.Hex() && typeAndVersion.Type == "CallProxy" {
				*options = append(*options, mcms.WithCallProxy(executorAddress.Hex()))
				return nil
			}
		}
	}

	return fmt.Errorf("failed to find call proxy contract for timelock %v", timelockAddress)
}
