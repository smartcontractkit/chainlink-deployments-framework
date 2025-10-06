package predecessors

import (
	"time"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// CLDContext holds the context for the GitHub repository where the PRs are managed.
type CLDContext struct {
	Owner       string
	Name        string
	Domain      string
	Environment string
	QueueID     string
}

type PRNum int64

type PRView struct {
	Number           PRNum
	CreatedAt        time.Time
	Body             string
	Head             PRHead
	Proposal         *mcms.TimelockProposal
	ProposalData     ProposalsOpData
	ProposalFilename string
	ProposalContent  string
}

// PRHead holds the repo/owner/SHA for the PR head.
type PRHead struct {
	Owner string
	Repo  string
	SHA   string
	Ref   string
}

type ProposalsOpData map[mcmstypes.ChainSelector]McmOpData

type McmOpData struct {
	MCMAddress      string
	StartingOpCount uint64
	OpsCount        uint64
}

// EndingOpCount returns the exclusive end opcount: StartingOpCount + OpsCount.
// Treat it as the end of a half-open interval [StartingOpCount, EndingOpCount()).
func (d McmOpData) EndingOpCount() uint64 {
	return d.StartingOpCount + d.OpsCount
}
