package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestSimulationOutputCannotOverwriteProposal(t *testing.T) {
	t.Parallel()
	for _, hardlink := range []bool{false, true} {
		t.Run(strconv.FormatBool(hardlink), func(t *testing.T) {
			t.Parallel()
			proposal := filepath.Join(t.TempDir(), "proposal.json")
			require.NoError(t, os.WriteFile(proposal, []byte("original proposal"), 0o600))
			output := proposal
			if hardlink {
				output = filepath.Join(filepath.Dir(proposal), "alias.json")
				require.NoError(t, os.Link(proposal, output))
			}
			_, _, err := prepareSimulationRun(t.Context(), logger.Nop(), Deps{}, executeForkFlags{proposalPath: proposal, simulationOut: output})
			require.ErrorContains(t, err, "must not alias")
			content, err := os.ReadFile(proposal)
			require.NoError(t, err)
			require.Equal(t, "original proposal", string(content))
		})
	}
}

func TestSimulationRecordsActualForkBaseline(t *testing.T) {
	t.Parallel()
	hash := "0x" + strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var rpc struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(request.Body).Decode(&rpc); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if rpc.Method != "eth_getBlockByNumber" || string(rpc.Params[0]) != `"latest"` {
			http.Error(w, "unexpected RPC", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": rpc.ID, "result": map[string]string{"number": "0x1896f4a", "hash": hash}})
	}))
	t.Cleanup(server.Close)
	run := &simulationRun{cfg: &forkConfig{chainSelector: 1, simulationOut: filepath.Join(t.TempDir(), "simulation.json")}, lggr: logger.Nop(), proposalSigningHash: "0x" + strings.Repeat("a", 64)}
	require.NoError(t, run.recordForkBaseline(t.Context(), server.URL))
	require.Equal(t, uint64(25784138), run.forkBlockNumber)
	require.Equal(t, hash, run.forkBlockHash)
	run.notSimulated("execution failed after baseline")
	require.NoError(t, run.emit())
	artifact, err := simulation.LoadArtifact(run.cfg.simulationOut)
	require.NoError(t, err)
	require.Equal(t, run.forkBlockNumber, artifact.Chains["1"].ForkBlockNumber)
	require.Equal(t, hash, artifact.Chains["1"].ForkBlockHash)
	run.cfg.forkBlockNumber = 42
	require.ErrorContains(t, run.recordForkBaseline(t.Context(), server.URL), "does not match requested pin")
}

type simulationTestViewFunc func(context.Context, simulation.Scope, string) (json.RawMessage, error)

func (f simulationTestViewFunc) ScopedView(ctx context.Context, scope simulation.Scope, url string) (json.RawMessage, error) {
	return f(ctx, scope, url)
}

func TestSimulationViewsRespectCancellationAndReportFailures(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"pre", "post", "canceled post"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			run := &simulationRun{cfg: &forkConfig{chainSelector: 1, forkTimeout: time.Second, simulationOut: filepath.Join(t.TempDir(), "failed.json")}, lggr: logger.Nop(), proposalSigningHash: "0x" + strings.Repeat("a", 64)}
			run.provider = simulationTestViewFunc(func(ctx context.Context, _ simulation.Scope, _ string) (json.RawMessage, error) {
				if stage == "canceled post" {
					return nil, ctx.Err()
				}

				return nil, errors.New("fixture getter failure")
			})
			if stage == "pre" {
				run.preView(t.Context(), "")
			} else {
				run.preDone = true
				run.pre = json.RawMessage(`{"chains":{}}`)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if stage == "canceled post" {
					cancel()
				}
				run.postViewAndDiff(ctx, "")
			}
			require.False(t, run.simulated)
			if stage == "canceled post" {
				require.Contains(t, run.reason, "context canceled")
			} else {
				require.Contains(t, run.reason, "fixture getter failure")
			}
			require.NoError(t, run.emit())
			artifact, err := simulation.LoadArtifact(run.cfg.simulationOut)
			require.NoError(t, err)
			require.False(t, artifact.Chains["1"].Simulated)
			require.Empty(t, artifact.Chains["1"].Changes)
		})
	}
}
