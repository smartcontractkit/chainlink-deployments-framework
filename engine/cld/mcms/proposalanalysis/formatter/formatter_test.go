package formatter

import (
	"context"
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock formatter for testing
type mockFormatter struct {
	id string
}

func (m *mockFormatter) ID() string {
	return m.id
}

func (m *mockFormatter) Format(ctx context.Context, req FormatterRequest, proposal types.AnalyzedProposal) ([]bytes, error) {
	return []byte("mock output"), nil
}

func TestFormatterRegistry(t *testing.T) {
	t.Run("Register and Get formatter", func(t *testing.T) {
		registry := NewFormatterRegistry()
		formatter := &mockFormatter{id: "test-formatter"}

		err := registry.Register(formatter)
		require.NoError(t, err)

		retrieved, ok := registry.Get("test-formatter")
		assert.True(t, ok)
		assert.Equal(t, formatter, retrieved)
	})

	t.Run("Register nil formatter returns error", func(t *testing.T) {
		registry := NewFormatterRegistry()

		err := registry.Register(nil)
		require.ErrorContains(t, err, "cannot be nil")
	})

	t.Run("Register formatter with empty ID returns error", func(t *testing.T) {
		registry := NewFormatterRegistry()
		formatter := &mockFormatter{id: ""}

		err := registry.Register(formatter)
		require.ErrorContains(t, err, "cannot be empty")
	})

	t.Run("Register duplicate ID returns error", func(t *testing.T) {
		registry := NewFormatterRegistry()
		formatter1 := &mockFormatter{id: "duplicate"}
		formatter2 := &mockFormatter{id: "duplicate"}

		err := registry.Register(formatter1)
		require.NoError(t, err)

		err = registry.Register(formatter2)
		require.EqualError(t, err, `formatter with ID "duplicate" is already registered`)

		// Verify first formatter is still registered
		retrieved, ok := registry.Get("duplicate")
		assert.True(t, ok)
		assert.Equal(t, formatter1, retrieved)
	})

	t.Run("Get non-existent formatter", func(t *testing.T) {
		registry := NewFormatterRegistry()

		retrieved, ok := registry.Get("non-existent")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("List formatters", func(t *testing.T) {
		registry := NewFormatterRegistry()

		formatter1 := &mockFormatter{id: "formatter-1"}
		formatter2 := &mockFormatter{id: "formatter-2"}
		formatter3 := &mockFormatter{id: "formatter-3"}

		registry.Register(formatter1)
		registry.Register(formatter2)
		registry.Register(formatter3)

		ids := registry.List()
		assert.Len(t, ids, 3)
		assert.ElementsMatch(t, []string{"formatter-1", "formatter-2", "formatter-3"}, ids)
	})

	t.Run("List empty registry", func(t *testing.T) {
		registry := NewFormatterRegistry()

		ids := registry.List()
		assert.Empty(t, ids)
	})
}
