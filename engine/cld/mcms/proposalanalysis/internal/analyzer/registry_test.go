package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	analyzerpkg "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
	analyzermocks "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/mocks"
)

func TestAnalyzerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register and Get analyzer", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()
		analyzer := newMockProposalAnalyzer(t, "test-analyzer")

		err := registry.Register(analyzer)
		require.NoError(t, err)

		retrieved, ok := registry.Get("test-analyzer")
		assert.True(t, ok)
		assert.Equal(t, analyzer, retrieved)
	})

	t.Run("Register nil analyzer returns error", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()

		err := registry.Register(nil)
		require.ErrorContains(t, err, "cannot be nil")
	})

	t.Run("Register analyzer with empty ID returns error", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()
		analyzer := newMockProposalAnalyzer(t, "")

		err := registry.Register(analyzer)
		require.ErrorContains(t, err, "cannot be empty")
	})

	t.Run("Register duplicate ID returns error", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()
		analyzer1 := newMockProposalAnalyzer(t, "duplicate")
		analyzer2 := newMockProposalAnalyzer(t, "duplicate")

		err := registry.Register(analyzer1)
		require.NoError(t, err)

		err = registry.Register(analyzer2)
		require.EqualError(t, err, `analyzer with ID "duplicate" is already registered`)

		retrieved, ok := registry.Get("duplicate")
		assert.True(t, ok)
		assert.Equal(t, analyzer1, retrieved)
	})

	t.Run("Register unsupported analyzer type returns error", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()
		analyzer := &unknownAnalyzer{id: "invalid"}

		err := registry.Register(analyzer)
		require.EqualError(t, err, "unknown analyzer type")
	})

	t.Run("Get non-existent analyzer", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()

		retrieved, ok := registry.Get("non-existent")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("List analyzers", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()

		a1 := newMockProposalAnalyzer(t, "proposal-analyzer")
		a2 := newMockBatchOperationAnalyzer(t, "batch-analyzer")
		a3 := newMockCallAnalyzer(t, "call-analyzer")
		a4 := newMockParameterAnalyzer(t, "parameter-analyzer")

		require.NoError(t, registry.Register(a1))
		require.NoError(t, registry.Register(a2))
		require.NoError(t, registry.Register(a3))
		require.NoError(t, registry.Register(a4))

		ids := registry.List()
		assert.Len(t, ids, 4)
		assert.ElementsMatch(t, []string{"proposal-analyzer", "batch-analyzer", "call-analyzer", "parameter-analyzer"}, ids)
	})

	t.Run("type-specific analyzer lists", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()

		proposalAnalyzer := newMockProposalAnalyzer(t, "proposal-analyzer")
		batchAnalyzer := newMockBatchOperationAnalyzer(t, "batch-analyzer")
		callAnalyzer := newMockCallAnalyzer(t, "call-analyzer")
		parameterAnalyzer := newMockParameterAnalyzer(t, "parameter-analyzer")

		require.NoError(t, registry.Register(proposalAnalyzer))
		require.NoError(t, registry.Register(batchAnalyzer))
		require.NoError(t, registry.Register(callAnalyzer))
		require.NoError(t, registry.Register(parameterAnalyzer))

		assert.Len(t, registry.ProposalAnalyzers(), 1)
		assert.Equal(t, proposalAnalyzer, registry.ProposalAnalyzers()[0])

		assert.Len(t, registry.BatchOperationAnalyzers(), 1)
		assert.Equal(t, batchAnalyzer, registry.BatchOperationAnalyzers()[0])

		assert.Len(t, registry.CallAnalyzers(), 1)
		assert.Equal(t, callAnalyzer, registry.CallAnalyzers()[0])

		assert.Len(t, registry.ParameterAnalyzers(), 1)
		assert.Equal(t, parameterAnalyzer, registry.ParameterAnalyzers()[0])
	})

	t.Run("All returns all analyzers in deterministic order", func(t *testing.T) {
		t.Parallel()

		registry := analyzerpkg.NewRegistry()

		a1 := newMockProposalAnalyzer(t, "z-proposal-analyzer")
		a2 := newMockBatchOperationAnalyzer(t, "a-batch-analyzer")
		a3 := newMockCallAnalyzer(t, "m-call-analyzer")

		require.NoError(t, registry.Register(a1))
		require.NoError(t, registry.Register(a2))
		require.NoError(t, registry.Register(a3))

		all := registry.All()
		require.Len(t, all, 3)
		assert.Equal(t, "a-batch-analyzer", all[0].ID())
		assert.Equal(t, "m-call-analyzer", all[1].ID())
		assert.Equal(t, "z-proposal-analyzer", all[2].ID())
	})
}

func newMockProposalAnalyzer(t *testing.T, id string) *analyzermocks.MockProposalAnalyzer {
	t.Helper()

	mockAnalyzer := analyzermocks.NewMockProposalAnalyzer(t)
	mockAnalyzer.EXPECT().ID().Return(id)

	return mockAnalyzer
}

func newMockBatchOperationAnalyzer(t *testing.T, id string) *analyzermocks.MockBatchOperationAnalyzer {
	t.Helper()

	mockAnalyzer := analyzermocks.NewMockBatchOperationAnalyzer(t)
	mockAnalyzer.EXPECT().ID().Return(id)

	return mockAnalyzer
}

func newMockCallAnalyzer(t *testing.T, id string) *analyzermocks.MockCallAnalyzer {
	t.Helper()

	mockAnalyzer := analyzermocks.NewMockCallAnalyzer(t)
	mockAnalyzer.EXPECT().ID().Return(id)

	return mockAnalyzer
}

func newMockParameterAnalyzer(t *testing.T, id string) *analyzermocks.MockParameterAnalyzer {
	t.Helper()

	mockAnalyzer := analyzermocks.NewMockParameterAnalyzer(t)
	mockAnalyzer.EXPECT().ID().Return(id)

	return mockAnalyzer
}

type unknownAnalyzer struct {
	id string
}

func (m *unknownAnalyzer) ID() string {
	return m.id
}

func (m *unknownAnalyzer) Dependencies() []string {
	return nil
}
