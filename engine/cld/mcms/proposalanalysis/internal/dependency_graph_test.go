package internal

import (
	"context"
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock analyzer for testing
type mockAnalyzer struct {
	id           string
	dependencies []string
}

func (m *mockAnalyzer) ID() string {
	return m.id
}

func (m *mockAnalyzer) Dependencies() []string {
	return m.dependencies
}

func TestNewDependencyGraph(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("empty graph", func(t *testing.T) {
		graph, err := NewDependencyGraph([]types.BaseAnalyzer{})
		require.NoError(t, err)
		assert.NotNil(t, graph)
		assert.Empty(t, graph.nodes)
	})

	t.Run("single analyzer", func(t *testing.T) {
		a1 := &mockAnalyzer{id: "a1"}
		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a1})
		require.NoError(t, err)
		assert.Len(t, graph.nodes, 1)
		assert.Contains(t, graph.nodes, "a1")
	})

	t.Run("duplicate ID error", func(t *testing.T) {
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a1"}
		_, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate analyzer ID")
	})

	t.Run("empty ID error", func(t *testing.T) {
		a1 := &mockAnalyzer{id: ""}
		_, err := NewDependencyGraph([]types.BaseAnalyzer{a1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-empty ID")
	})

	t.Run("unknown dependency error", func(t *testing.T) {
		a1 := &mockAnalyzer{id: "a1", dependencies: []string{"unknown"}}
		_, err := NewDependencyGraph([]types.BaseAnalyzer{a1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown analyzer")
	})
}

func TestTopologicalSort(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("linear dependency chain", func(t *testing.T) {
		// a1 -> a2 -> a3
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a2", dependencies: []string{"a1"}}
		a3 := &mockAnalyzer{id: "a3", dependencies: []string{"a2"}}

		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a3, a1, a2})
		require.NoError(t, err)

		sorted, err := graph.TopologicalSort()
		require.NoError(t, err)
		require.Len(t, sorted, 3)

		// a1 should come before a2, a2 before a3
		ids := make([]string, len(sorted))
		for i, a := range sorted {
			ids[i] = a.ID()
		}
		assert.Equal(t, []string{"a1", "a2", "a3"}, ids)
	})

	t.Run("diamond dependency", func(t *testing.T) {
		//     a1
		//    /  \
		//   a2  a3
		//    \  /
		//     a4
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a2", dependencies: []string{"a1"}}
		a3 := &mockAnalyzer{id: "a3", dependencies: []string{"a1"}}
		a4 := &mockAnalyzer{id: "a4", dependencies: []string{"a2", "a3"}}

		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a4, a2, a3, a1})
		require.NoError(t, err)

		sorted, err := graph.TopologicalSort()
		require.NoError(t, err)
		require.Len(t, sorted, 4)

		// Build position map
		pos := make(map[string]int)
		for i, a := range sorted {
			pos[a.ID()] = i
		}

		// Assert ordering constraints
		assert.Less(t, pos["a1"], pos["a2"])
		assert.Less(t, pos["a1"], pos["a3"])
		assert.Less(t, pos["a2"], pos["a4"])
		assert.Less(t, pos["a3"], pos["a4"])
	})

	t.Run("independent analyzers", func(t *testing.T) {
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a2"}
		a3 := &mockAnalyzer{id: "a3"}

		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2, a3})
		require.NoError(t, err)

		sorted, err := graph.TopologicalSort()
		require.NoError(t, err)
		assert.Len(t, sorted, 3)
	})
}

func TestDetectCycles(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("simple cycle", func(t *testing.T) {
		// a1 -> a2 -> a1 (cycle)
		a1 := &mockAnalyzer{id: "a1", dependencies: []string{"a2"}}
		a2 := &mockAnalyzer{id: "a2", dependencies: []string{"a1"}}

		_, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
	})

	t.Run("self dependency", func(t *testing.T) {
		a1 := &mockAnalyzer{id: "a1", dependencies: []string{"a1"}}

		_, err := NewDependencyGraph([]types.BaseAnalyzer{a1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
	})

	t.Run("complex cycle", func(t *testing.T) {
		// a1 -> a2 -> a3 -> a1 (cycle)
		a1 := &mockAnalyzer{id: "a1", dependencies: []string{"a3"}}
		a2 := &mockAnalyzer{id: "a2", dependencies: []string{"a1"}}
		a3 := &mockAnalyzer{id: "a3", dependencies: []string{"a2"}}

		_, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2, a3})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "circular dependency")
	})
}

func TestGetLevels(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	t.Run("linear chain has sequential levels", func(t *testing.T) {
		// a1 -> a2 -> a3
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a2", dependencies: []string{"a1"}}
		a3 := &mockAnalyzer{id: "a3", dependencies: []string{"a2"}}

		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2, a3})
		require.NoError(t, err)

		levels := graph.Levels()
		require.Len(t, levels, 3)
		assert.Len(t, levels[0], 1)
		assert.Equal(t, "a1", levels[0][0].ID())
		assert.Len(t, levels[1], 1)
		assert.Equal(t, "a2", levels[1][0].ID())
		assert.Len(t, levels[2], 1)
		assert.Equal(t, "a3", levels[2][0].ID())
	})

	t.Run("diamond allows parallel execution", func(t *testing.T) {
		//     a1
		//    /  \
		//   a2  a3
		//    \  /
		//     a4
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a2", dependencies: []string{"a1"}}
		a3 := &mockAnalyzer{id: "a3", dependencies: []string{"a1"}}
		a4 := &mockAnalyzer{id: "a4", dependencies: []string{"a2", "a3"}}

		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2, a3, a4})
		require.NoError(t, err)

		levels := graph.Levels()
		require.Len(t, levels, 3)

		// Level 0: a1
		assert.Len(t, levels[0], 1)
		assert.Equal(t, "a1", levels[0][0].ID())

		// Level 1: a2 and a3 (can run in parallel)
		assert.Len(t, levels[1], 2)
		ids := []string{levels[1][0].ID(), levels[1][1].ID()}
		assert.ElementsMatch(t, []string{"a2", "a3"}, ids)

		// Level 2: a4
		assert.Len(t, levels[2], 1)
		assert.Equal(t, "a4", levels[2][0].ID())
	})

	t.Run("independent analyzers in same level", func(t *testing.T) {
		a1 := &mockAnalyzer{id: "a1"}
		a2 := &mockAnalyzer{id: "a2"}
		a3 := &mockAnalyzer{id: "a3"}

		graph, err := NewDependencyGraph([]types.BaseAnalyzer{a1, a2, a3})
		require.NoError(t, err)

		levels := graph.Levels()
		require.Len(t, levels, 1)
		assert.Len(t, levels[0], 3)
	})
}
