package mcms

import (
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// forkConfig holds all the configuration needed to execute a proposal on a forked chain.
// This is the internal configuration type that mirrors the legacy cfgv2 struct.
type forkConfig struct {
	kind             types.ProposalKind
	proposalPath     string
	proposal         mcms.Proposal
	timelockProposal *mcms.TimelockProposal
	chainSelector    uint64
	blockchains      chain.BlockChains
	envStr           string
	env              cldf.Environment
	domain           cldfdomain.Domain
	forkedEnv        cldfenvironment.ForkedEnvironment
	fork             bool
	proposalCtx      analyzer.ProposalContext
}
