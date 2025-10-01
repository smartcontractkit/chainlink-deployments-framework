package predecessors

import (
	"bytes"
	"math/big"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-github/v71/github"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/evm"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

// -- helpers for tests --
// quick McmOpData builder
func mdata(addr string, start, ops uint64) McmOpData {
	return McmOpData{
		MCMAddress:      addr,
		StartingOpCount: start,
		OpsCount:        ops,
	}
}

// ProposalsOpData from a small spec:
// map[selector] = (addr, start, ops)
type chainSpec struct {
	addr       string
	start, ops uint64
}

func podFrom(spec map[uint64]chainSpec) ProposalsOpData {
	out := make(ProposalsOpData, len(spec))
	for sel, s := range spec {
		out[mcmstypes.ChainSelector(sel)] = mdata(s.addr, s.start, s.ops)
	}

	return out
}

func mkPRView(num PRNum, ts time.Time, pod ProposalsOpData) PRView {
	return PRView{
		Number:       num,
		CreatedAt:    ts,
		ProposalData: pod,
	}
}

// makeTimelockProposalBytes builds a valid MCMS Timelock proposal and returns JSON bytes.
func makeTimelockProposalBytes(t *testing.T, chain mcmstypes.ChainSelector, mcmAddr string, startOpCount uint64) []byte {
	t.Helper()
	prop, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(24*time.Hour).Unix())). //nolint:gosec // test code, overflow acceptable
		SetDescription("test").
		AddTimelockAddress(chain, mcmAddr).
		AddChainMetadata(chain, mcmstypes.ChainMetadata{
			StartingOpCount: startOpCount,
			MCMAddress:      mcmAddr,
		}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction([20]byte{}, []byte{}, big.NewInt(0), "noop", nil),
			},
		}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction([20]byte{}, []byte{}, big.NewInt(0), "noop", nil),
			},
		}).
		AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction([20]byte{}, []byte{}, big.NewInt(0), "noop", nil),
			},
		}).
		SetAction(mcmstypes.TimelockActionSchedule).
		SetDelay(mcmstypes.NewDuration(time.Second)).
		Build()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, mcms.WriteTimelockProposal(&buf, prop))

	return buf.Bytes()
}

// ghClientToServer wires a go-github client to the provided httptest.Server.
func ghClientToServer(t *testing.T, srv *httptest.Server) *github.Client {
	t.Helper()
	baseURL, _ := url.Parse(srv.URL + "/")
	cli := github.NewClient(nil)
	cli.BaseURL = baseURL
	cli.UploadURL = baseURL

	return cli
}

// -- unit tests --
func TestComputeHighestOpCountsFromPredecessors_BaselineOnly_NoPreds(t *testing.T) {
	t.Parallel()

	// New proposal has chains 1 and 2
	newView := podFrom(map[uint64]chainSpec{
		1: {"0xA", 100, 5},
		2: {"0xB", 200, 6},
	})

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, nil)

	require.Equal(t, uint64(100), highest[mcmstypes.ChainSelector(1)])
	require.Equal(t, uint64(200), highest[mcmstypes.ChainSelector(2)])
}

func TestComputeHighestOpCountsFromPredecessors_BaselineDifferentFromPredecessorStartOp(t *testing.T) {
	t.Parallel()

	// New proposal has chains 1 and 2
	newView := podFrom(map[uint64]chainSpec{
		// the on chain opcount during the new proposal creation was 1093
		9335212494177455608: {"0xA", 1093, 16},
	})

	pred1 := mkPRView(
		101,
		time.Now().Add(-2*time.Hour),
		// the predecessor starting op count is higher than the new proposal baseline,
		// which can happen if there was a predecessor that got merged in between
		podFrom(map[uint64]chainSpec{9335212494177455608: {"0xA", 1098, 15}}))

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, []PRView{pred1})

	require.Equal(t, 1098+15, int(highest[mcmstypes.ChainSelector(9335212494177455608)])) // #nosec G115
}

func TestComputeHighestOpCountsFromPredecessors_SumsOps(t *testing.T) {
	t.Parallel()

	// New proposal baseline starts at 10
	newView := podFrom(map[uint64]chainSpec{
		1: {"0xA", 10, 1}, // baseline 10
	})

	// Pred1: chain 1, same MCM, Start=50, Ops=10
	// Pred2: chain 1, same MCM, Start=70, Ops=5
	// Sum of ops = 15 → baseline(10) + 15 = 25
	// But since Pred2 has higher StartingOpCount, we take that one as the base for summing ops.
	// EndingOpCount = 70 + 5 = 75
	pred1 := mkPRView(101, time.Now().Add(-2*time.Hour),
		podFrom(map[uint64]chainSpec{1: {"0xA", 50, 10}}))
	pred2 := mkPRView(102, time.Now().Add(-1*time.Hour),
		podFrom(map[uint64]chainSpec{1: {"0xA", 70, 5}}))

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, []PRView{pred1, pred2})

	require.Equal(t, 75, int(highest[mcmstypes.ChainSelector(1)])) // #nosec G115
}

func TestComputeHighestOpCountsFromPredecessors_MultipleChainsAndPreds(t *testing.T) {
	t.Parallel()

	newView := podFrom(map[uint64]chainSpec{
		1: {"0xA", 10, 2},  // baseline 5
		2: {"0xB", 50, 10}, // baseline 20
		3: {"0xC", 13, 3},  // baseline 0
	})

	// Preds touching different subsets:
	// P1: chain1(A) EndingOpCount= 6+4=10, chain2(B) EndingOpCount=50+1=51
	// P2: chain1(A) EndingOpCount= 9+3=12
	// P3: chain3(C) EndingOpCount= 1+100=101
	p1 := mkPRView(1, time.Now().Add(-3*time.Hour),
		podFrom(map[uint64]chainSpec{
			1: {"0xA", 10, 4},
			2: {"0xB", 50, 1},
		}))
	p2 := mkPRView(2, time.Now().Add(-2*time.Hour),
		podFrom(map[uint64]chainSpec{
			1: {"0xA", 10, 3},
		}))
	p3 := mkPRView(3, time.Now().Add(-1*time.Hour),
		podFrom(map[uint64]chainSpec{
			3: {"0xC", 13, 100},
		}))

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, []PRView{p1, p2, p3})

	require.Equal(t, uint64(17), highest[mcmstypes.ChainSelector(1)]) // from p2

	require.Equal(t, uint64(51), highest[mcmstypes.ChainSelector(2)]) // from p1

	require.Equal(t, uint64(113), highest[mcmstypes.ChainSelector(3)]) // from p3
}

func TestComputeHighestOpcountsFromPredecessors_IgnoresDifferentChainOrMCM(t *testing.T) {
	t.Parallel()

	newView := podFrom(map[uint64]chainSpec{
		1: {"0xA", 100, 1},
		2: {"0xB", 50, 1},
	})

	// Preds with:
	// - same MCM but different chain (should not count)
	// - same chain but different MCM (should not count)
	pDiffChain := mkPRView(10, time.Now().Add(-2*time.Hour),
		podFrom(map[uint64]chainSpec{
			2: {"0xA", 1000, 5}, // MCM "0xA" but chain 2 (newView has 0xA on chain 1)
		}))
	pDiffMCM := mkPRView(11, time.Now().Add(-1*time.Hour),
		podFrom(map[uint64]chainSpec{
			1: {"0xZZ", 2000, 5}, // chain matches (1) but MCM differs
		}))

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, []PRView{pDiffChain, pDiffMCM})

	// unchanged: stick to baseline
	require.Equal(t, 100, int(highest[mcmstypes.ChainSelector(1)])) // #nosec G115

	require.Equal(t, 50, int(highest[mcmstypes.ChainSelector(2)])) // #nosec G115
}

func TestComputeHighestOpcountsFromPredecessors_CaseAndWhitespaceInsensitiveMCM(t *testing.T) {
	t.Parallel()

	newView := podFrom(map[uint64]chainSpec{
		1: {" 0xAbC ", 5, 0},
	})

	// predecessor same addr but lowercased and without spaces
	pred := mkPRView(77, time.Now().Add(-1*time.Hour),
		podFrom(map[uint64]chainSpec{
			1: {"0xabc", 5, 2}, // EndingOpCount = 7
		}))

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, []PRView{pred})

	require.Equal(t, uint64(7), highest[mcmstypes.ChainSelector(1)])
}

func TestComputeHighestOpcountsFromPredecessors_PredLowerThanBaselineIgnored(t *testing.T) {
	t.Parallel()

	// Baseline 100; predecessor EndingOpCount 80 -> should stay 100
	newView := podFrom(map[uint64]chainSpec{
		1: {"0xA", 100, 5},
	})
	pred := mkPRView(5, time.Now().Add(-1*time.Hour),
		podFrom(map[uint64]chainSpec{
			1: {"0xA", 70, 10}, // EndingOpCount 80 < baseline 100
		}))

	highest := ComputeHighestOpCountsFromPredecessors(logger.Test(t), newView, []PRView{pred})

	require.Equal(t, uint64(100), highest[mcmstypes.ChainSelector(1)])
}

func TestMatchesProposalPath_PositiveAndNegative(t *testing.T) {
	t.Parallel()

	domain := "foo"
	env := "bar"

	// positive
	ok := matchesProposalPath(domain, env, "domains/foo/bar/proposals/abc.json")
	require.True(t, ok)

	// wrong domain
	require.False(t, matchesProposalPath(domain, env, "domains/wrong/bar/proposals/abc.json"))
	// wrong env
	require.False(t, matchesProposalPath(domain, env, "domains/foo/wrong/proposals/abc.json"))
	// wrong suffix
	require.False(t, matchesProposalPath(domain, env, "domains/foo/bar/proposals/abc.yaml"))
	// wrong prefix
	require.False(t, matchesProposalPath(domain, env, "something/domains/foo/bar/proposals/abc.json"))
}

func newTimelockProposal(t *testing.T, start uint64, ops int) *mcms.TimelockProposal {
	t.Helper()

	chain := mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)

	b := mcms.
		NewTimelockProposalBuilder().
		SetVersion("v1").
		SetValidUntil(uint32(time.Now().Add(24*time.Hour).Unix())).
		SetDescription("unit test proposal").
		AddTimelockAddress(chain, "0xTimelock").
		AddChainMetadata(chain, mcmstypes.ChainMetadata{
			StartingOpCount: start,
			MCMAddress:      "0xMCM",
		}).
		SetAction(mcmstypes.TimelockActionSchedule).
		SetDelay(mcmstypes.NewDuration(2 * time.Second))

	// add N no-op transactions so OperationCounts == ops
	for i := 0; i < ops; i++ {
		_ = b.AddOperation(mcmstypes.BatchOperation{
			ChainSelector: chain,
			Transactions: []mcmstypes.Transaction{
				evm.NewTransaction(common.Address{}, []byte{}, big.NewInt(0), "test", nil),
			},
		})
	}

	prop, err := b.Build()
	require.NoError(t, err)
	return prop
}

func writeProposal(t *testing.T, p *mcms.TimelockProposal) (string, mcmstypes.ChainSelector) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "p.json")
	fh, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o600)
	require.NoError(t, err)
	require.NoError(t, mcms.WriteTimelockProposal(fh, p))
	_ = fh.Close()

	// Return the chain we used
	var sel mcmstypes.ChainSelector
	for cs := range p.ChainMetadatas() {
		sel = cs
		break
	}
	return path, sel
}

func TestApplyHighestOpCountsToProposal(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)

	t.Run("updates when proposed is higher", func(t *testing.T) {
		t.Parallel()

		prop := newTimelockProposal(t, 1, 2) // start=1, ops=2
		path, sel := writeProposal(t, prop)

		err := ApplyHighestOpCountsToProposal(lggr, path, map[mcmstypes.ChainSelector]uint64{
			sel: 5, // higher than current 1
		})
		require.NoError(t, err)

		// Reload and assert StartingOpCount bumped to 5
		loaded, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, path)
		require.NoError(t, err)
		tp := loaded.(*mcms.TimelockProposal)
		got := tp.ChainMetadatas()[sel].StartingOpCount
		assert.Equal(t, uint64(5), got)
	})

	t.Run("no change when proposed <= current", func(t *testing.T) {
		t.Parallel()

		prop := newTimelockProposal(t, 3, 1) // start=3
		path, sel := writeProposal(t, prop)

		err := ApplyHighestOpCountsToProposal(lggr, path, map[mcmstypes.ChainSelector]uint64{
			sel: 3, // equal to current
		})
		require.NoError(t, err)

		loaded, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, path)
		require.NoError(t, err)
		tp := loaded.(*mcms.TimelockProposal)
		assert.Equal(t, uint64(3), tp.ChainMetadatas()[sel].StartingOpCount)
	})

	t.Run("ignores unknown chains", func(t *testing.T) {
		t.Parallel()

		prop := newTimelockProposal(t, 2, 1)
		path, sel := writeProposal(t, prop)

		otherChain := mcmstypes.ChainSelector(chainsel.ETHEREUM_MAINNET.Selector)
		require.NotEqual(t, sel, otherChain)

		err := ApplyHighestOpCountsToProposal(lggr, path, map[mcmstypes.ChainSelector]uint64{
			otherChain: 999,
		})
		require.NoError(t, err)

		loaded, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, path)
		require.NoError(t, err)
		tp := loaded.(*mcms.TimelockProposal)
		assert.Equal(t, uint64(2), tp.ChainMetadatas()[sel].StartingOpCount)
	})

	t.Run("error when file does not exist", func(t *testing.T) {
		t.Parallel()

		err := ApplyHighestOpCountsToProposal(lggr, "does/not/exist.json", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "load proposal")
	})
}

func TestParseProposalOpsData(t *testing.T) {
	t.Run("success returns correct op data", func(t *testing.T) {
		t.Parallel()

		prop := newTimelockProposal(t, 4, 3) // start=4, 3 ops
		path, sel := writeProposal(t, prop)

		got, err := ParseProposalOpsData(t.Context(), path)
		require.NoError(t, err)

		// Must contain exactly the chain we wrote
		require.Contains(t, got, sel)
		entry := got[sel]
		assert.Equal(t, "0xMCM", entry.MCMAddress)
		assert.Equal(t, uint64(4), entry.StartingOpCount)
		assert.Equal(t, uint64(3), entry.OpsCount) // 3 transactions added above
	})

	t.Run("error when file missing", func(t *testing.T) {
		t.Parallel()

		_, err := ParseProposalOpsData(t.Context(), "missing.json")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "load proposal from")
	})
}
