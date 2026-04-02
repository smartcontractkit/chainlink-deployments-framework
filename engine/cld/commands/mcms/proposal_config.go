package mcms

import (
	"context"
	"crypto/rand"
	"fmt"
	"slices"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/types"

	"github.com/ethereum/go-ethereum/common"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	acceptExpiredProposal = "accept-expired-proposal-option" // sentinel option to accept expired proposals.
	randomSalt            = "random-salt-option"             // sentinel option to override the proposal's salt with a random value
)

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
	opts ...any,
) (*ProposalConfig, error) {
	// Validate proposal kind
	proposalKind, exists := types.StringToProposalKind[flags.ProposalKind]
	if !exists {
		return nil, fmt.Errorf("unknown proposal kind '%s'", flags.ProposalKind)
	}

	// Load proposal from file
	fileProposal, err := deps.ProposalLoader(proposalKind, flags.ProposalPath)
	if err != nil {
		if !slices.Contains(opts, acceptExpiredProposal) || !isProposalExpiredError(err) {
			return nil, fmt.Errorf("error loading proposal: %w", err)
		}
	}

	var mcmsProposal *mcms.Proposal
	var timelockCastedProposal *mcms.TimelockProposal

	if proposalKind == types.KindTimelockProposal {
		timelockCastedProposal = fileProposal.(*mcms.TimelockProposal)
		if flags.Fork && slices.Contains(opts, randomSalt) && timelockCastedProposal.Action == types.TimelockActionSchedule {
			timelockCastedProposal.SaltOverride = newRandomSalt()
			_, serr := timelockCastedProposal.SetOperationIDs(ctx, true)
			if serr != nil {
				return nil, fmt.Errorf("failed to set operation IDs after resetting the timelock proposal salt: %w", serr)
			}
		}

		converters, cerr := chainwrappers.BuildConverters(timelockCastedProposal.ChainMetadata)
		if cerr != nil {
			return nil, fmt.Errorf("error building converters for timelock proposal: %w", cerr)
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
		cfg.ForkedEnv, err = deps.ForkEnvironmentLoader(ctx, dom, cfg.EnvStr, nil,
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

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
