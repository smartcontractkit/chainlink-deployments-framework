package predecessors

import (
	"testing"
	"time"

	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

func TestBuildPRDependencyGraph_BasicEdges(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name   string
		views  []PRView
		assert func(t *testing.T, g *ProposalsPRGraph)
	}{
		{
			name: "edge when same chain+MCM; direction older->newer",
			views: []PRView{
				mkPR(10, now.Add(-3*time.Hour), mcmData(1, "0xABC")),
				mkPR(11, now.Add(-2*time.Hour), mcmData(1, "0xABC")), // same chain+MCM -> edge 10->11
				mkPR(12, now.Add(-1*time.Hour), mcmData(1, "0xDEF")), // different MCM -> no edge from 10/11
			},
			assert: func(t *testing.T, g *ProposalsPRGraph) {
				t.Helper()
				require.Len(t, g.Nodes, 3)
				// 10 -> 11
				require.ElementsMatch(t, []PRNum{11}, g.Nodes[10].Succ)
				require.ElementsMatch(t, []PRNum{10}, g.Nodes[11].Pred)
				// 12 is isolated relative to {10,11} because MCM differs
				require.Empty(t, g.Nodes[12].Pred)
				require.Empty(t, g.Nodes[12].Succ)

				// topo should respect 10 before 11; 12
				require.Equal(t, []PRNum{10, 11, 12}, g.Topo)
			},
		},
		{
			name: "no edge when same MCM but different chains",
			views: []PRView{
				mkPR(20, now.Add(-3*time.Hour), mcmData(1, "0xMCM")),
				mkPR(21, now.Add(-2*time.Hour), mcmData(2, "0xMCM")), // same MCM, different chain -> no edge
			},
			assert: func(t *testing.T, g *ProposalsPRGraph) {
				t.Helper()
				require.Empty(t, g.Nodes[20].Succ)
				require.Empty(t, g.Nodes[21].Pred)
				require.Equal(t, []PRNum{20, 21}, g.Topo)
			},
		},
		{
			name: "tie on CreatedAt resolved by smaller PR number",
			views: func() []PRView {
				ts := now.Add(-5 * time.Hour)
				return []PRView{
					mkPR(101, ts, mcmData(1, "0xA")),
					mkPR(100, ts, mcmData(3, "0xB")),
				}
			}(),
			assert: func(t *testing.T, g *ProposalsPRGraph) {
				t.Helper()
				// No edges (different chains/MCMs). Order falls back to time+number -> 100 then 101.
				require.Equal(t, []PRNum{100, 101}, g.Topo)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g, err := BuildPRDependencyGraph(tc.views)
			require.NoError(t, err)
			tc.assert(t, g)
		})
	}
}

func TestGraph_MultiChain_CrossDependencies(t *testing.T) {
	t.Parallel()

	// Chain selectors we’ll use: 1, 2, 3
	// MCMs: X, Y, Y2, Z, ZZ, Q2
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// P1: {1:X, 2:Y}
	p1 := mkPR(101, t0.Add(1*time.Hour), podm(map[uint64]string{
		1: "0xX",
		2: "0xY",
	}))
	// P2: {1:X, 3:Z} -> shares chain1:X with P1
	p2 := mkPR(102, t0.Add(2*time.Hour), podm(map[uint64]string{
		1: "0xX",
		3: "0xZ",
	}))
	// P3: {2:Y, 3:ZZ} -> shares chain2:Y with P1
	p3 := mkPR(103, t0.Add(3*time.Hour), podm(map[uint64]string{
		2: "0xY",
		3: "0xZZ",
	}))
	// P4: {1:X, 2:Y2, 3:Z} -> shares two chains with P2 (1:X and 3:Z), and one with P1 (1:X)
	p4 := mkPR(104, t0.Add(4*time.Hour), podm(map[uint64]string{
		1: "0xX",
		2: "0xY2",
		3: "0xZ",
	}))
	// P5: {2:Y, 3:Q2} -> shares chain2:Y with P1 and P3
	p5 := mkPR(105, t0.Add(5*time.Hour), podm(map[uint64]string{
		2: "0xY",
		3: "0xQ2",
	}))

	g, err := BuildPRDependencyGraph([]PRView{p1, p2, p3, p4, p5})

	require.NoError(t, err)
	require.Len(t, g.Nodes, 5)

	// Pred/Succ expectations:
	// 101 -> {102,103,104,105}
	require.ElementsMatch(t, []PRNum{102, 103, 104, 105}, g.Nodes[101].Succ)
	require.ElementsMatch(t, []PRNum{}, g.Nodes[101].Pred)
	requireUniquePRs(t, g.Nodes[101].Succ)

	// 102 -> {104}
	require.ElementsMatch(t, []PRNum{104}, g.Nodes[102].Succ)
	require.ElementsMatch(t, []PRNum{101}, g.Nodes[102].Pred)
	requireUniquePRs(t, g.Nodes[102].Succ)

	// 103 -> {105}
	require.ElementsMatch(t, []PRNum{105}, g.Nodes[103].Succ)
	require.ElementsMatch(t, []PRNum{101}, g.Nodes[103].Pred)

	// 104 -> {}
	require.ElementsMatch(t, []PRNum{}, g.Nodes[104].Succ)
	require.ElementsMatch(t, []PRNum{101, 102}, g.Nodes[104].Pred)
	requireUniquePRs(t, g.Nodes[104].Pred)

	// 105 -> {}
	require.ElementsMatch(t, []PRNum{}, g.Nodes[105].Succ)
	require.ElementsMatch(t, []PRNum{101, 103}, g.Nodes[105].Pred)
	requireUniquePRs(t, g.Nodes[105].Pred)

	// Topo should be strictly time-ordered because all edges are older->newer and times are ascending.
	require.Equal(t, []PRNum{101, 102, 103, 104, 105}, g.Topo)
}

func TestRelatedByMCMOnAnyChain_CaseAndSpaceInsensitive(t *testing.T) {
	t.Parallel()

	x := ProposalsOpData{
		mcmstypes.ChainSelector(1): McmOpData{MCMAddress: " 0xAbC ", StartingOpCount: 0, OpsCount: 1},
	}
	y := ProposalsOpData{
		mcmstypes.ChainSelector(1): McmOpData{MCMAddress: "0xabc", StartingOpCount: 10, OpsCount: 5},
	}
	z := ProposalsOpData{
		mcmstypes.ChainSelector(1): McmOpData{MCMAddress: "0xdef", StartingOpCount: 0, OpsCount: 1},
	}

	require.True(t, relatedByMCMOnAnyChain(x, y), "same chain & same MCM ignoring case/space should relate")
	require.False(t, relatedByMCMOnAnyChain(x, z), "different MCM on same chain should not relate")

	// different chains with same MCM -> should be false
	a := ProposalsOpData{
		mcmstypes.ChainSelector(1): McmOpData{MCMAddress: "0xabc"},
	}
	b := ProposalsOpData{
		mcmstypes.ChainSelector(2): McmOpData{MCMAddress: "0xabc"},
	}
	require.False(t, relatedByMCMOnAnyChain(a, b))
}

func requireUniquePRs(t *testing.T, xs []PRNum) {
	t.Helper()
	set := map[PRNum]struct{}{}
	for _, x := range xs {
		set[x] = struct{}{}
	}
	require.Len(t, set, len(xs), "expected slice to have no duplicate PR numbers")
}

func podm(m map[uint64]string) ProposalsOpData {
	out := make(ProposalsOpData, len(m))
	for sel, mcm := range m {
		out[mcmstypes.ChainSelector(sel)] = McmOpData{
			MCMAddress:      mcm,
			StartingOpCount: 0,
			OpsCount:        1,
		}
	}

	return out
}

// helper to build a ProposalsOpData with one (selector -> MCM) entry
func mcmData(sel mcmstypes.ChainSelector, mcm string) ProposalsOpData {
	return ProposalsOpData{
		sel: McmOpData{
			MCMAddress:      mcm,
			StartingOpCount: 0,
			OpsCount:        1,
		},
	}
}

// helper to quickly create a PRView
func mkPR(num PRNum, ts time.Time, pod ProposalsOpData) PRView {
	return PRView{
		Number:       num,
		CreatedAt:    ts,
		ProposalData: pod, // note: matches your struct field name
	}
}
