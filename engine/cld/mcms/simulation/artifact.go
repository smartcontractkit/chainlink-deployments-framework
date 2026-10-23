package simulation

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	ForkBlockHash   string                 `json:"forkBlockHash,omitempty"`
	Targets         []Target               `json:"targets,omitempty"`
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

// Target identifies a contract included in the effect scope. Dependency-only
// context is not part of this list or the proposal's coverage denominator.
type Target struct {
	Address      string `json:"address"`
	ContractType string `json:"contractType"`
	Version      string `json:"version,omitempty"`
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

// LoadArtifact reads and validates a .statediff.json artifact from disk.
func LoadArtifact(path string) (*SimulationArtifact, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading simulation artifact %q: %w", path, err)
	}

	var artifact SimulationArtifact
	if err := statediff.DecodeJSON(raw, &artifact); err != nil {
		return nil, fmt.Errorf("parsing simulation artifact %q: %w", path, err)
	}
	if err := artifact.Validate(); err != nil {
		return nil, fmt.Errorf("invalid simulation artifact %q: %w", path, err)
	}

	return &artifact, nil
}

// Validate checks the artifact's internal consistency. Proposal identity and
// the expected chain set are checked separately; producer trust is established
// by the workflow, never by a self-reported field in this file.
func (a *SimulationArtifact) Validate() error {
	if a == nil {
		return errors.New("simulation artifact is nil")
	}
	if a.SchemaVersion != ArtifactSchemaVersion {
		return fmt.Errorf("schema version %d, expected %d", a.SchemaVersion, ArtifactSchemaVersion)
	}
	if !isHash(a.ProposalSigningHash) {
		return errors.New("proposal signing hash must be a 0x-prefixed 32-byte hash")
	}
	for key, result := range a.Chains {
		selector, err := strconv.ParseUint(key, 10, 64)
		if err != nil || selector == 0 || strconv.FormatUint(selector, 10) != key {
			return fmt.Errorf("invalid chain selector %q", key)
		}
		if result.ForkBlockHash != "" && !isHash(result.ForkBlockHash) {
			return fmt.Errorf("invalid fork block hash for chain %s", key)
		}
		if !result.Simulated && (len(result.Changes) > 0 || len(result.Assertions) > 0) {
			return fmt.Errorf("chain %s was not simulated but carries changes or assertions", key)
		}
		if !result.Simulated && strings.TrimSpace(result.Reason) == "" {
			return fmt.Errorf("chain %s was not simulated but has no reason", key)
		}
		if result.Simulated && result.Reason != "" {
			return fmt.Errorf("simulated chain %s carries a failure reason", key)
		}
		bad := make(map[string]bool, len(result.Unreadable))
		for _, entry := range result.Unreadable {
			if entry.ChainSelector != selector || entry.Reason == "" {
				return fmt.Errorf("invalid unreadable entry for chain %s", key)
			}
			bad[scopeAddressKey(entry.Address)] = true
		}
		for _, change := range result.Changes {
			if change.ChainSelector != selector || change.Path == "" || change.Kind > statediff.KindChanged {
				return fmt.Errorf("invalid change for chain %s", key)
			}
			if change.Kind == statediff.KindAdded && change.Before != nil {
				return fmt.Errorf("added change for chain %s carries a before value", key)
			}
			if change.Kind == statediff.KindRemoved && change.After != nil {
				return fmt.Errorf("removed change for chain %s carries an after value", key)
			}
			parts := strings.SplitN(change.Path, ".", 4)
			if len(parts) < 3 || parts[0] != "chains" || parts[1] != key {
				return fmt.Errorf("change path does not identify chain %s", key)
			}
			pathAddress, _, _ := strings.Cut(parts[2], "_")
			if change.Address == "" || scopeAddressKey(pathAddress) != scopeAddressKey(change.Address) {
				return fmt.Errorf("change path/address mismatch for chain %s", key)
			}
			if bad[""] || bad[scopeAddressKey(change.Address)] {
				return fmt.Errorf("chain %s contains changes for an unreadable target %s", key, change.Address)
			}
		}
		targets := make(map[string]bool, len(result.Targets))
		for _, target := range result.Targets {
			if strings.TrimSpace(target.Address) == "" {
				return fmt.Errorf("empty target address for chain %s", key)
			}
			address := scopeAddressKey(target.Address)
			if targets[address] {
				return fmt.Errorf("duplicate target %s for chain %s", target.Address, key)
			}
			targets[address] = true
		}
		if result.Targets != nil {
			for _, change := range result.Changes {
				if !targets[scopeAddressKey(change.Address)] {
					return fmt.Errorf("change outside target inventory for chain %s", key)
				}
			}
			for _, entry := range result.Unreadable {
				if entry.Address != "" && !targets[scopeAddressKey(entry.Address)] {
					return fmt.Errorf("unreadable entry outside target inventory for chain %s", key)
				}
			}
		}
	}

	return nil
}

func isHash(value string) bool {
	if len(value) != 66 || !strings.HasPrefix(value, "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])

	return err == nil
}

// ValidateExpectedChains requires an exact, nonempty selector set. Missing
// matrix jobs are failures, not a successful subset of the proposal.
func (a *SimulationArtifact) ValidateExpectedChains(selectors []string) error {
	if len(selectors) == 0 {
		return errors.New("expected chain selectors must not be empty")
	}
	expected := make(map[string]bool, len(selectors))
	for _, selector := range selectors {
		if expected[selector] {
			return fmt.Errorf("duplicate expected chain selector %s", selector)
		}
		expected[selector] = true
		if _, ok := a.Chains[selector]; !ok {
			return fmt.Errorf("missing simulation result for expected chain %s", selector)
		}
	}
	for selector := range a.Chains {
		if !expected[selector] {
			return fmt.Errorf("unexpected simulation result for chain %s", selector)
		}
	}

	return nil
}

// LoadArtifactForSigningHash reads, validates, and staleness-checks a
// .statediff.json artifact: its recorded proposal signing hash must match the
// hash of the proposal it is about to be rendered for. A mismatch means the
// proposal changed after the simulation ran and the artifact is stale — the
// simulation must be re-run, never rendered against a newer proposal.
func LoadArtifactForSigningHash(path, proposalSigningHash string) (*SimulationArtifact, error) {
	artifact, err := LoadArtifact(path)
	if err != nil {
		return nil, err
	}
	if artifact.ProposalSigningHash != proposalSigningHash {
		return nil, fmt.Errorf("simulation artifact %q is stale: recorded proposal signing hash %s does not match this proposal's %s",
			path, artifact.ProposalSigningHash, proposalSigningHash)
	}

	return artifact, nil
}

// MergeArtifacts combines the per-chain artifacts a fork-test matrix run
// emits into one artifact. Every input must describe the same proposal —
// same schema version, same proposal signing hash — because each matrix job
// simulated the same pinned proposal file; anything else refuses to merge.
// The same chain appearing twice must carry an identical result (an overlap
// from a re-run): a conflict is an error, never silently overwritten.
// Not-simulated chains (simulated:false) merge verbatim so coverage never
// reads a skip as a pass.
func MergeArtifacts(artifacts ...*SimulationArtifact) (*SimulationArtifact, error) {
	if len(artifacts) == 0 {
		return nil, errors.New("no simulation artifacts to merge")
	}
	for i, artifact := range artifacts {
		if artifact == nil {
			return nil, fmt.Errorf("simulation artifact %d is nil", i)
		}
	}

	merged := &SimulationArtifact{
		SchemaVersion:       artifacts[0].SchemaVersion,
		ProposalSigningHash: artifacts[0].ProposalSigningHash,
		GeneratedAt:         artifacts[0].GeneratedAt,
		Chains:              make(map[string]SimulationChainResult),
	}

	for _, artifact := range artifacts {
		switch {
		case artifact.SchemaVersion != merged.SchemaVersion:
			return nil, fmt.Errorf("simulation artifact schema version %d does not match %d",
				artifact.SchemaVersion, merged.SchemaVersion)
		case artifact.ProposalSigningHash != merged.ProposalSigningHash:
			return nil, fmt.Errorf("simulation artifact signing hash %s does not match %s — artifacts for different proposals cannot be merged",
				artifact.ProposalSigningHash, merged.ProposalSigningHash)
		}
		if err := artifact.Validate(); err != nil {
			return nil, err
		}
		if artifact.GeneratedAt.After(merged.GeneratedAt) {
			merged.GeneratedAt = artifact.GeneratedAt
		}
		for sel, result := range artifact.Chains {
			if existing, ok := merged.Chains[sel]; ok {
				if !statediff.EqualJSONValues(existing, result) {
					return nil, fmt.Errorf("conflicting simulation results for chain %s", sel)
				}

				continue
			}
			merged.Chains[sel] = result
		}
	}

	return merged, nil
}
