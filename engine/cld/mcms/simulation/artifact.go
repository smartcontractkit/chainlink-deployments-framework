package simulation

import (
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
)

// ArtifactSchemaVersion is the current .statediff.json schema version.
const ArtifactSchemaVersion = 1

// SimulationArtifact is the on-disk shape of <proposal>.statediff.json: one
// artifact per proposal, one chain entry per simulated chain. Fork-test
// matrix jobs emit one artifact per chain; the aggregation job merges them.
//
// proposalSigningHash is the staleness guard: consumers compare it against
// the live proposal's signing hash and discard the artifact on mismatch.
type SimulationArtifact struct {
	SchemaVersion       int                              `json:"schemaVersion"`
	ProposalSigningHash string                           `json:"proposalSigningHash"`
	GeneratedAt         time.Time                        `json:"generatedAt"`
	Chains              map[string]SimulationChainResult `json:"chains"`
}

// SimulationChainResult is one chain's simulation outcome.
//
// simulated is false whenever the full pre/post view pair with a diff could
// not be produced — a skipped non-EVM chain, an early exit, a failed view —
// so coverage reporting never reads a skip as a pass. Reason names why.
type SimulationChainResult struct {
	ForkBlockNumber uint64                 `json:"forkBlockNumber"`
	Simulated       bool                   `json:"simulated"`
	Reason          string                 `json:"reason,omitempty"`
	Changes         []statediff.Change     `json:"changes"`
	Unreadable      []statediff.Unreadable `json:"unreadable"`
	// Notes records provenance that qualified an otherwise-complete
	// simulation (e.g. the timelock stage being skipped for a non-schedule
	// action).
	Notes []string `json:"notes,omitempty"`
	// Assertions is filled by the intent-assertion engine (Phase 3): it
	// partitions the observed changes against the author's declared
	// expectations. Empty until then, by design — an absent assertion
	// section must never be mistaken for all-Confirmed.
	Assertions []Assertion `json:"assertions"`
}

// Assertion is one intent-check outcome (Phase 3). It is declared here so
// the artifact schema is stable across phases, but no assertion engine
// populates it yet.
type Assertion struct {
	Path     string `json:"path"`
	Expected any    `json:"expected,omitempty"`
	Observed any    `json:"observed,omitempty"`
	Bucket   string `json:"bucket"` // confirmed | unexpected | missing
}
