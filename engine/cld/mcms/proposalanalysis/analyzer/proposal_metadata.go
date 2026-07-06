package analyzer

import mcmstypes "github.com/smartcontractkit/mcms/types"

// ProposalExecutionMetadata carries timelock proposal fields needed by analyzers.
type ProposalExecutionMetadata struct {
	Action            mcmstypes.TimelockAction
	Delay             mcmstypes.Duration
	TimelockAddresses map[uint64]string
	ChainMetadata     map[uint64]mcmstypes.ChainMetadata
}
