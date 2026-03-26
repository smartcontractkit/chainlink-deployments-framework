package scheduler

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_ValidGraphBuildsLevels(t *testing.T) {
	t.Parallel()

	g, err := New([]testAnalyzer{
		{id: "proposal"},
		{id: "batch", deps: []string{"proposal"}},
		{id: "call", deps: []string{"batch"}},
		{id: "param", deps: []string{"call"}},
		{id: "cross", deps: []string{"proposal"}},
	})
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"proposal"},
		{"batch", "cross"},
		{"call"},
		{"param"},
	}, g.Levels())
}

func TestNew_DuplicateAnalyzerID(t *testing.T) {
	t.Parallel()

	_, err := New([]testAnalyzer{
		{id: "a"},
		{id: "a"},
	})
	require.ErrorContains(t, err, `duplicate analyzer ID "a"`)
}

func TestNew_MissingDependency(t *testing.T) {
	t.Parallel()

	_, err := New([]testAnalyzer{
		{id: "a", deps: []string{"missing"}},
	})
	require.ErrorContains(t, err, `depends on unknown analyzer "missing"`)
}

func TestNew_Cycle(t *testing.T) {
	t.Parallel()

	_, err := New([]testAnalyzer{
		{id: "a", deps: []string{"b"}},
		{id: "b", deps: []string{"a"}},
	})
	require.ErrorContains(t, err, "contains a cycle")
}

func TestRun_DependencyOrder(t *testing.T) {
	t.Parallel()

	g, err := New([]testAnalyzer{
		{id: "a"},
		{id: "b"},
		{id: "c", deps: []string{"a", "b"}},
		{id: "d", deps: []string{"c"}},
	})
	require.NoError(t, err)

	var (
		mu       sync.Mutex
		executed []string
	)

	err = g.Run(t.Context(), func(_ context.Context, a testAnalyzer) error {
		mu.Lock()
		executed = append(executed, a.id)
		mu.Unlock()

		return nil
	})
	require.NoError(t, err)

	idx := map[string]int{}
	for i, id := range executed {
		idx[id] = i
	}
	require.Less(t, idx["a"], idx["c"])
	require.Less(t, idx["b"], idx["c"])
	require.Less(t, idx["c"], idx["d"])
}

func TestRun_ContinuesOnErrorAndAggregates(t *testing.T) {
	t.Parallel()

	g, err := New([]testAnalyzer{
		{id: "a"},
		{id: "b", deps: []string{"a"}},
		{id: "c"},
	})
	require.NoError(t, err)

	var (
		mu       sync.Mutex
		executed []string
	)

	boom := errors.New("boom")
	err = g.Run(t.Context(), func(_ context.Context, a testAnalyzer) error {
		mu.Lock()
		executed = append(executed, a.id)
		mu.Unlock()

		if a.id == "a" {
			return boom
		}

		return nil
	})
	require.ErrorContains(t, err, `run analyzer "a": boom`)
	require.ErrorContains(t, err, `skip analyzer "b": dependency failure`)
	require.Contains(t, executed, "a")
	require.Contains(t, executed, "c")
	require.NotContains(t, executed, "b")
}

func TestRun_NilFunction(t *testing.T) {
	t.Parallel()

	g, err := New([]testAnalyzer{{id: "a"}})
	require.NoError(t, err)

	err = g.Run(t.Context(), nil)
	require.EqualError(t, err, "run function cannot be nil")
}

type testAnalyzer struct {
	id   string
	deps []string
}

func (a testAnalyzer) ID() string {
	return a.id
}

func (a testAnalyzer) Dependencies() []string {
	return a.deps
}
