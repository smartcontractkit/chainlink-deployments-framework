package mcms

import (
	"context"
	"crypto/rand"
	"fmt"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/xssnick/tonutils-go/tlb"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// acceptExpiredProposal is a sentinel option to accept expired proposals.
var acceptExpiredProposal = struct{}{}

// ProposalConfig holds the loaded proposal configuration.
type ProposalConfig struct {
	Kind             types.ProposalKind
	Proposal         mcms.Proposal
	TimelockProposal *mcms.TimelockProposal // nil if not a timelock proposal
	ChainSelector    uint64
	EnvStr           string
	Env              cldf.Environment
	ForkedEnv        cldfenvironment.ForkedEnvironment // populated when Fork is true
	Fork             bool
	ProposalCtx      analyzer.ProposalContext
}

// ProposalFlags contains common proposal-related flags.
type ProposalFlags struct {
	ProposalPath  string
	ProposalKind  string
	Environment   string
	ChainSelector uint64
	Fork          bool
}

// LoadProposalConfig loads and validates a proposal configuration.
func LoadProposalConfig(
	ctx context.Context,
	lggr logger.Logger,
	dom domain.Domain,
	deps *Deps,
	proposalCtxProvider analyzer.ProposalContextProvider,
	flags ProposalFlags,
	opts ...interface{},
) (*ProposalConfig, error) {
	// Validate proposal kind
	proposalKind, exists := types.StringToProposalKind[flags.ProposalKind]
	if !exists {
		return nil, fmt.Errorf("unknown proposal kind '%s'", flags.ProposalKind)
	}

	// Load proposal from file
	fileProposal, err := deps.ProposalLoader(proposalKind, flags.ProposalPath)
	if err != nil {
		if !containsAcceptExpired(opts) || !isProposalExpiredError(err) {
			return nil, fmt.Errorf("error loading proposal: %w", err)
		}
	}

	var mcmsProposal *mcms.Proposal
	var timelockCastedProposal *mcms.TimelockProposal

	if proposalKind == types.KindTimelockProposal {
		timelockCastedProposal = fileProposal.(*mcms.TimelockProposal)
		if flags.Fork && timelockCastedProposal.Action == types.TimelockActionSchedule {
			timelockCastedProposal.SaltOverride = newRandomSalt()
		}

		// Construct converters for each chain
		converters := make(map[types.ChainSelector]sdk.TimelockConverter)
		for chain := range timelockCastedProposal.ChainMetadata {
			fam, famErr := types.GetChainSelectorFamily(chain)
			if famErr != nil {
				return nil, fmt.Errorf("error getting chain family: %w", famErr)
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
				var suiErr error
				converter, suiErr = sui.NewTimelockConverter()
				if suiErr != nil {
					return nil, fmt.Errorf("error creating Sui timelock converter: %w", suiErr)
				}
			case chainsel.FamilyTon:
				converter = ton.NewTimelockConverter(tlb.MustFromTON(defaultTONExecutorAmount))
			default:
				return nil, fmt.Errorf("unsupported chain family %s", fam)
			}

			converters[chain] = converter
		}

		convertedProposal, _, convErr := timelockCastedProposal.Convert(ctx, converters)
		if convErr != nil {
			return nil, fmt.Errorf("error converting timelock proposal: %w", convErr)
		}

		mcmsProposal = &convertedProposal
	} else {
		mcmsProposal = fileProposal.(*mcms.Proposal)
	}

	cfg := &ProposalConfig{
		Kind:             proposalKind,
		Proposal:         *mcmsProposal,
		TimelockProposal: timelockCastedProposal,
		ChainSelector:    flags.ChainSelector,
		EnvStr:           flags.Environment,
		Fork:             flags.Fork,
	}

	// Determine chain selectors to load
	chainSelectors := make([]uint64, len(cfg.Proposal.ChainSelectors()))
	if cfg.ChainSelector != 0 {
		chainSelectors = []uint64{cfg.ChainSelector}
	} else {
		for i, selector := range cfg.Proposal.ChainSelectors() {
			chainSelectors[i] = uint64(selector)
		}
	}

	// Load Environment
	if cfg.Fork {
		// For forked environments, load via LoadFork
		cfg.ForkedEnv, err = cldfenvironment.LoadFork(ctx, dom, cfg.EnvStr, nil,
			cldfenvironment.OnlyLoadChainsFor(chainSelectors),
			cldfenvironment.WithoutJD(),
			cldfenvironment.WithLogger(lggr))
		if err != nil {
			return nil, fmt.Errorf("error loading forked environment: %w", err)
		}
		cfg.Env = cfg.ForkedEnv.Environment
	} else {
		cfg.Env, err = deps.EnvironmentLoader(ctx, dom, cfg.EnvStr, lggr,
			cldfenvironment.OnlyLoadChainsFor(chainSelectors),
			cldfenvironment.WithoutJD())
		if err != nil {
			return nil, fmt.Errorf("error loading environment: %w", err)
		}
	}

	// Create ProposalContext
	if proposalCtxProvider != nil {
		cfg.ProposalCtx, err = proposalCtxProvider(cfg.Env)
		if err != nil {
			return nil, fmt.Errorf("error creating proposal context: %w", err)
		}
	}

	return cfg, nil
}

// newRandomSalt generates a random 32-byte salt.
func newRandomSalt() *common.Hash {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	h := common.BytesToHash(b[:])

	return &h
}

// isProposalExpiredError checks if the error indicates an expired proposal.
func isProposalExpiredError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common expired proposal error patterns
	errStr := err.Error()

	return containsStr(errStr, "expired") || containsStr(errStr, "valid_until")
}

// containsAcceptExpired checks if the opts contain acceptExpiredProposal.
func containsAcceptExpired(opts []interface{}) bool {
	for _, opt := range opts {
		if opt == acceptExpiredProposal {
			return true
		}
	}

	return false
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
