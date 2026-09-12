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

	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// simulationRun tracks one execute-fork state-view simulation: the resolved
// scope, the pre/post view documents, and how far the run got. The deferred
// emitter serializes exactly what exists, so every exit path of executeFork
// — success, early error, bypass — produces an honest artifact: never a
// missing file that reads as "not attempted", never a skip that reads as a
// pass.
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
	lggr.Infow("simulation scope resolved",
		"chainSelector", cfg.chainSelector,
		"targets", len(scope.Targets),
		"context", len(scope.Context),
		"unknownAddresses", len(scopeUnreadable))

	return &simulationRun{
		lggr:            lggr,
		cfg:             cfg,
		provider:        mcmsCfg.SimulationViewProvider,
		scope:           scope,
		scopeUnreadable: scopeUnreadable,
	}, nil
}

// preView takes the "before" view against the fork, after the test-signer
// surgery and before setRoot: the proposal's own effects are excluded, the
// harness mutations are absorbed by the differ's ignore list. A failed view
// does not abort the fork execution — the artifact records why.
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
// It runs on a fresh context derived from the command context's values but
// not its cancellation: an expired execution deadline must not also kill
// the diagnostic view, and the artifact still has to be emitted.
func (r *simulationRun) postViewAndDiff(ctx context.Context, rpcURL string) {
	if !r.preDone {
		if r.reason == "" {
			r.reason = "post-view skipped: no pre-view was taken"
		}

		return
	}
	viewCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.viewTimeout())
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
// mask the fork execution's own result.
func (r *simulationRun) emit() error {
	signingHash, err := r.cfg.proposal.SigningHash()
	if err != nil {
		return fmt.Errorf("computing proposal signing hash: %w", err)
	}

	changes := r.changes
	if changes == nil {
		changes = []statediff.Change{}
	}
	unreadable := r.unreadable
	if unreadable == nil {
		unreadable = []statediff.Unreadable{}
	}

	artifact := simulation.SimulationArtifact{
		SchemaVersion:       simulation.ArtifactSchemaVersion,
		ProposalSigningHash: signingHash.Hex(),
		GeneratedAt:         time.Now(),
		Chains: map[string]simulation.SimulationChainResult{
			strconv.FormatUint(r.cfg.chainSelector, 10): {
				ForkBlockNumber: r.cfg.forkBlockNumber,
				Simulated:       r.simulated,
				Reason:          r.reason,
				Changes:         changes,
				Unreadable:      unreadable,
				Notes:           r.notes,
				Assertions:      []simulation.Assertion{},
			},
		},
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
	r.lggr.Infow("simulation artifact written",
		"path", r.cfg.simulationOut, "simulated", r.simulated,
		"changes", len(changes), "unreadable", len(unreadable))

	return nil
}
