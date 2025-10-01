package predecessors

import "sort"

// proposalsPRGraph is a dependency graph of proposals where an edge u -> v means:
// "u must precede v" (they share at least one MCM on some chain and u is older).
type proposalsPRGraph struct {
	Nodes map[PRNum]*prNode // node lookup by PR number
	Topo  []PRNum           // topologically sorted PR numbers (older -> newer respecting deps)
}

// prNode stores the PR view plus explicit predecessor/successor lists.
type prNode struct {
	PR   PRView
	Pred []PRNum
	Succ []PRNum
}

// relatedByMCMOnAnyChain returns true iff there is at least one chain selector
// that exists in both proposals where the MCM matches (case/space-insensitive).
func relatedByMCMOnAnyChain(x, y ProposalsOpData) bool {
	if len(x) > len(y) {
		x, y = y, x
	}
	for selector, mcmDataX := range x {
		if mcmDataY, ok := y[selector]; ok && sameMCM(mcmDataX.MCMAddress, mcmDataY.MCMAddress) {
			return true
		}
	}

	return false
}

// buildPRDependencyGraph creates a DAG with edges ONLY from older -> newer,
// and only when relatedByMCMOnAnyChain(...) is true.
// Ties on CreatedAt are broken by PR number (smaller = older).
func buildPRDependencyGraph(views []PRView) *proposalsPRGraph {
	n := len(views)
	g := &proposalsPRGraph{
		Nodes: make(map[PRNum]*prNode, n),
	}

	if n == 0 {
		return g
	}

	// stable time order: older first
	sorted := make([]PRView, n)
	copy(sorted, views)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].Number < sorted[j].Number
		}

		return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
	})

	for _, v := range sorted {
		g.Nodes[v.Number] = &prNode{PR: v}
	}

	// only add edges from strictly older -> strictly newer
	for i := range n {
		old := sorted[i]
		for j := i + 1; j < n; j++ {
			newer := sorted[j]
			if relatedByMCMOnAnyChain(old.ProposalData, newer.ProposalData) {
				u := old.Number
				v := newer.Number
				g.Nodes[u].Succ = append(g.Nodes[u].Succ, v)
				g.Nodes[v].Pred = append(g.Nodes[v].Pred, u)
			}
		}
	}

	g.Topo = topoOrderStable(sorted, g.Nodes)

	return g
}

// topoOrderStable apply topological sort to the graph.
// using Kahn's algorithm https://www.geeksforgeeks.org/dsa/topological-sorting-indegree-based-solution/
// with a min-PQ tie-breaker based on the PR time order
func topoOrderStable(sorted []PRView, nodes map[PRNum]*prNode) []PRNum {
	// indegree
	indeg := make(map[PRNum]int, len(nodes))
	for id, nd := range nodes {
		indeg[id] = len(nd.Pred)
	}

	// rank by stable time/ID order
	rank := make(map[PRNum]int, len(sorted))
	for i, v := range sorted {
		rank[v.Number] = i
	}

	// min-PQ by rank (smallest rank first)
	pq := make([]PRNum, 0, len(nodes))
	push := func(x PRNum) {
		// insert x keeping pq sorted by rank
		i := sort.Search(len(pq), func(i int) bool { return rank[pq[i]] > rank[x] })
		pq = append(pq, 0)
		copy(pq[i+1:], pq[i:])
		pq[i] = x
	}
	pop := func() PRNum {
		x := pq[0]
		pq = pq[1:]

		return x
	}

	// seed PQ with all initial zero-indegree nodes in stable order
	for _, v := range sorted {
		if indeg[v.Number] == 0 {
			push(v.Number)
		}
	}

	order := make([]PRNum, 0, len(nodes))
	for len(pq) > 0 {
		u := pop()
		order = append(order, u)
		for _, v := range nodes[u].Succ {
			indeg[v]--
			if indeg[v] == 0 {
				push(v)
			}
		}
	}

	return order
}
