package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// simulationRun tracks one execute-fork state-view simulation: the resolved
// scope, the pre/post view documents, and how far the run got. The deferred
// emitter serializes exactly what exists after setup establishes trusted
// proposal identity. Earlier setup failures leave a missing expected chain,
// which strict aggregation rejects.
type simulationRun struct {
	lggr            logger.Logger
	cfg             *forkConfig
	provider        simulation.ViewProvider
	scope           simulation.Scope
	scopeUnreadable []statediff.Unreadable
	pre             json.RawMessage
	post            json.RawMessage
	changes         []statediff.Change
	unreadable      []statediff.Unreadable
	simulated       bool
	reason          string
	notes           []string
	preDone         bool
	emitted         bool
	forkBlockNumber uint64
	forkBlockHash   string
	// proposalSigningHash is captured from the pristine converted proposal
	// at setup — BEFORE the test-signer surgery mutates it. The mutation
	// (OverridePreviousRoot, cleared signatures, possibly bumped validUntil,
	// random salt) changes the signing hash, because OverridePreviousRoot is
	// part of the merkle root's chain-metadata leaf. Consumers compare the
	// artifact's hash against the proposal file's own hash, so the artifact
	// must record the pristine one, not the run variant's.
	proposalSigningHash string
}

// prepareSimulationRun loads the authored proposal once and establishes its
// identity before environment startup or random-salt mutation. The caller
// supplies the returned proposal through its local loader dependency, so the
// executed proposal and recorded hash cannot come from different file reads.
func prepareSimulationRun(ctx context.Context, lggr logger.Logger, deps Deps, f executeForkFlags) (*simulationRun, mcms.ProposalInterface, error) {
	if info, err := os.Lstat(f.simulationOut); err == nil {
		if !info.Mode().IsRegular() {
			return nil, nil, errors.New("simulation output must be a regular file")
		}
		if proposalInfo, statErr := os.Stat(f.proposalPath); statErr == nil && os.SameFile(info, proposalInfo) {
			return nil, nil, errors.New("simulation output must not alias the proposal input")
		}
		if rerr := os.Remove(f.simulationOut); rerr != nil {
			return nil, nil, fmt.Errorf("removing previous simulation output: %w", rerr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}
	loaded, err := deps.ProposalLoader(types.ProposalKind(f.proposalKind), f.proposalPath)
	if err != nil && !isProposalExpiredError(err) {
		return nil, nil, fmt.Errorf("loading simulation proposal: %w", err)
	}
	proposal, ok := loaded.(*mcms.TimelockProposal)
	if !ok || proposal == nil {
		return nil, nil, errors.New("simulation requires a TimelockProposal")
	}
	converters, err := chainwrappers.BuildConverters(proposal.ChainMetadata)
	if err != nil {
		return nil, nil, fmt.Errorf("building simulation converters: %w", err)
	}
	converted, _, err := proposal.Convert(ctx, converters)
	if err != nil {
		return nil, nil, fmt.Errorf("converting authored simulation proposal: %w", err)
	}
	hash, err := converted.SigningHash()
	if err != nil {
		return nil, nil, fmt.Errorf("hashing authored simulation proposal: %w", err)
	}
	cfg := &forkConfig{chainSelector: f.chainSelector, simulationOut: f.simulationOut, forkTimeout: f.forkTimeout, forkBlockNumber: f.forkBlockNumber}

	return &simulationRun{lggr: lggr, cfg: cfg, proposalSigningHash: hash.Hex()}, loaded, nil
}

// newSimulationRun resolves the scope for cfg's chain and returns the run.
// The view provider comes from the mcms Config; the scope resolves from the
// fork environment's own datastore (already loaded with the env's address
// refs), so no separate ref source is needed.
func newSimulationRun(ctx context.Context, lggr logger.Logger, mcmsCfg Config, cfg *forkConfig) (*simulationRun, error) {
	if mcmsCfg.SimulationViewProvider == nil {
		return nil, errors.New("simulation requested (--simulate-state) but no SimulationViewProvider is configured")
	}

	timelockProposal := cfg.timelockProposal
	if timelockProposal == nil {
		return nil, errors.New("simulation requires a TimelockProposal")
	}
	refStore := cfg.env.DataStore.Addresses()
	scope, scopeUnreadable, err := simulation.ResolveScope(ctx, refStore, timelockProposal, types.ChainSelector(cfg.chainSelector))
	if err != nil {
		return nil, fmt.Errorf("resolving simulation scope: %w", err)
	}

	// Capture the pristine signing hash before the caller's test-signer
	// surgery mutates cfg.proposal (see simulationRun.proposalSigningHash).
	signingHash, err := cfg.proposal.SigningHash()
	if err != nil {
		return nil, fmt.Errorf("computing proposal signing hash: %w", err)
	}

	lggr.Infow("simulation scope resolved",
		"chainSelector", cfg.chainSelector,
		"targets", len(scope.Targets),
		"context", len(scope.Context),
		"unknownAddresses", len(scopeUnreadable))

	run := &simulationRun{
		lggr:                lggr,
		cfg:                 cfg,
		provider:            mcmsCfg.SimulationViewProvider,
		scope:               scope,
		scopeUnreadable:     scopeUnreadable,
		proposalSigningHash: signingHash.Hex(),
		unreadable:          slices.Clone(scopeUnreadable),
	}
	if cfg.preparedSimulation != nil {
		run.proposalSigningHash = cfg.preparedSimulation.proposalSigningHash
		*cfg.preparedSimulation = *run
		run = cfg.preparedSimulation
	}

	return run, nil
}

// recordForkBaseline captures the actual block before harness transactions.
// A requested pin is checked against the node rather than copied as evidence.
func (r *simulationRun) recordForkBaseline(ctx context.Context, rpcURL string) error {
	client, err := ethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("connecting to fork baseline RPC: %w", err)
	}
	defer client.Close()
	var block *struct {
		Number hexutil.Uint64 `json:"number"`
		Hash   common.Hash    `json:"hash"`
	}
	if err := client.CallContext(ctx, &block, "eth_getBlockByNumber", "latest", false); err != nil {
		return fmt.Errorf("reading fork baseline: %w", err)
	}
	if block == nil || block.Hash == (common.Hash{}) {
		return errors.New("fork baseline block or hash missing")
	}
	r.forkBlockNumber = uint64(block.Number)
	r.forkBlockHash = block.Hash.Hex()
	if r.cfg.forkBlockNumber != 0 && r.forkBlockNumber != r.cfg.forkBlockNumber {
		return fmt.Errorf("fork baseline block %d does not match requested pin %d", r.forkBlockNumber, r.cfg.forkBlockNumber)
	}

	return nil
}

// preView captures pristine target state before test-signer surgery. A failed
// view does not abort execution; the artifact records the coverage failure.
func (r *simulationRun) preView(ctx context.Context, rpcURL string) {
	pre, err := r.provider.ScopedView(ctx, r.scope, rpcURL)
	if err != nil {
		r.reason = fmt.Sprintf("pre-view failed: %v", err)
		r.lggr.Warnw("simulation pre-view failed; continuing without it", "err", err)

		return
	}
	r.pre = pre
	r.preDone = true
	r.lggr.Info("simulation pre-view complete")
}

// postViewAndDiff takes the "after" view and diffs it against the pre-view.
// It shares the execution deadline and respects caller cancellation.
func (r *simulationRun) postViewAndDiff(ctx context.Context, rpcURL string) {
	if !r.preDone {
		if r.reason == "" {
			r.reason = "post-view skipped: no pre-view was taken"
		}

		return
	}
	viewCtx, cancel := context.WithTimeout(ctx, r.viewTimeout())
	defer cancel()

	post, err := r.provider.ScopedView(viewCtx, r.scope, rpcURL)
	if err != nil {
		r.reason = fmt.Sprintf("post-view failed: %v", err)
		r.lggr.Warnw("simulation post-view failed", "err", err)

		return
	}
	r.post = post

	changes, unreadable, err := statediff.DiffJSON(r.pre, r.post)
	if err != nil {
		r.reason = fmt.Sprintf("diffing pre/post views: %v", err)
		r.lggr.Warnw("simulation diff failed", "err", err)

		return
	}
	r.changes = changes
	r.unreadable = slices.Concat(r.scopeUnreadable, unreadable)
	r.simulated = true
	r.reason = ""
	r.lggr.Infow("simulation complete",
		"changes", len(changes), "unreadable", len(r.unreadable))
}

// notSimulated marks the run as not simulated for a named structural reason
// (non-EVM chain, bypassed stage).
func (r *simulationRun) notSimulated(reason string) {
	if r.reason == "" {
		r.reason = reason
	}
}

// note records provenance that qualified an otherwise-complete simulation.
func (r *simulationRun) note(note string) {
	r.notes = append(r.notes, note)
}

// earlyExitReason explains an incomplete run from the command's error when
// no more specific reason was recorded.
func (r *simulationRun) earlyExitReason(err error) {
	if !r.simulated && r.reason == "" && err != nil {
		r.reason = fmt.Sprintf("execute-fork exited before completing: %v", err)
	}
}

func (r *simulationRun) viewTimeout() time.Duration {
	if r.cfg.forkTimeout > 0 {
		return r.cfg.forkTimeout
	}

	return 300 * time.Second
}

// emit writes the .statediff.json artifact for whatever the run observed.
// Best effort: an emit failure is logged, never propagated — it must not
// mask the fork execution's own result. The signing hash is the pristine one
// captured at setup, so the artifact identifies the proposal as authored —
// what consumers compare against — even though the fork itself ran a
// test-signer-mutated variant.
func (r *simulationRun) emit() error {
	if !r.simulated && r.reason == "" {
		r.reason = "simulation did not complete"
	}
	changes := r.changes
	if changes == nil {
		changes = []statediff.Change{}
	}
	unreadable := r.unreadable
	if unreadable == nil {
		unreadable = []statediff.Unreadable{}
	}
	targets := make([]simulation.Target, 0, len(r.scope.Targets)+len(r.scopeUnreadable))
	for _, ref := range r.scope.Targets {
		version := ""
		if ref.Version != nil {
			version = ref.Version.String()
		}
		targets = append(targets, simulation.Target{Address: ref.Address, ContractType: string(ref.Type), Version: version})
	}
	for _, entry := range r.scopeUnreadable {
		targets = append(targets, simulation.Target{Address: entry.Address, ContractType: entry.ContractType, Version: entry.Version})
	}

	artifact := simulation.SimulationArtifact{
		SchemaVersion:       simulation.ArtifactSchemaVersion,
		ProposalSigningHash: r.proposalSigningHash,
		GeneratedAt:         time.Now(),
		Chains: map[string]simulation.SimulationChainResult{
			strconv.FormatUint(r.cfg.chainSelector, 10): {
				ForkBlockNumber: r.forkBlockNumber,
				ForkBlockHash:   r.forkBlockHash,
				Targets:         targets,
				Simulated:       r.simulated,
				Reason:          r.reason,
				Changes:         changes,
				Unreadable:      unreadable,
				Notes:           r.notes,
				Assertions:      []simulation.Assertion{},
			},
		},
	}
	if err := artifact.Validate(); err != nil {
		return fmt.Errorf("validating simulation artifact before write: %w", err)
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling simulation artifact: %w", err)
	}
	if dir := filepath.Dir(r.cfg.simulationOut); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating simulation artifact dir: %w", err)
		}
	}
	if err := os.WriteFile(r.cfg.simulationOut, data, 0o600); err != nil {
		return fmt.Errorf("writing simulation artifact: %w", err)
	}
	r.emitted = true
	r.lggr.Infow("simulation artifact written",
		"path", r.cfg.simulationOut, "simulated", r.simulated,
		"changes", len(changes), "unreadable", len(unreadable))

	return nil
}
