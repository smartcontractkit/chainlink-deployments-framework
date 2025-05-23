package deployment

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

// todo: clean up in future once Chainlink is migrated
type SolChain = solana.Chain
type SolProgramInfo = solana.ProgramInfo

var ProgramIDPrefix = solana.ProgramIDPrefix
var BufferIDPrefix = solana.BufferIDPrefix
var SolDefaultCommitment = solana.SolDefaultCommitment
