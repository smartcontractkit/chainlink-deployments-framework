package renderer

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/mocks"
)

// Mock renderer for testing
type mockRenderer struct {
	id string
}

func (m *mockRenderer) ID() string {
	return m.id
}

func (m *mockRenderer) RenderTo(w io.Writer, req RenderRequest, proposal analyzer.AnalyzedProposal) error {
	_, err := w.Write([]byte("mock output"))
	return err
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register and Get renderer", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		renderer := &mockRenderer{id: "test-renderer"}

		err := registry.Register(renderer)
		require.NoError(t, err)

		retrieved, ok := registry.Get("test-renderer")
		assert.True(t, ok)
		assert.Equal(t, renderer, retrieved)
	})

	t.Run("Register nil renderer returns error", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()

		err := registry.Register(nil)
		require.ErrorContains(t, err, "renderer cannot be nil")
	})

	t.Run("Register renderer with empty ID returns error", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		renderer := &mockRenderer{id: ""}

		err := registry.Register(renderer)
		require.ErrorContains(t, err, "renderer ID cannot be empty")
	})

	t.Run("Register duplicate ID returns error", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()
		renderer1 := &mockRenderer{id: "duplicate"}
		renderer2 := &mockRenderer{id: "duplicate"}

		err := registry.Register(renderer1)
		require.NoError(t, err)

		err = registry.Register(renderer2)
		require.EqualError(t, err, `renderer with ID "duplicate" is already registered`)

		// Verify first renderer is still registered
		retrieved, ok := registry.Get("duplicate")
		assert.True(t, ok)
		assert.Equal(t, renderer1, retrieved)
	})

	t.Run("Get non-existent renderer", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()

		retrieved, ok := registry.Get("non-existent")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("List renderers", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()

		renderer1 := &mockRenderer{id: "renderer-1"}
		renderer2 := &mockRenderer{id: "renderer-2"}
		renderer3 := &mockRenderer{id: "renderer-3"}

		err := registry.Register(renderer1)
		require.NoError(t, err)
		err = registry.Register(renderer2)
		require.NoError(t, err)
		err = registry.Register(renderer3)
		require.NoError(t, err)

		ids := registry.List()
		assert.Len(t, ids, 3)
		assert.ElementsMatch(t, []string{"renderer-1", "renderer-2", "renderer-3"}, ids)
	})

	t.Run("List empty registry", func(t *testing.T) {
		t.Parallel()

		registry := NewRegistry()

		ids := registry.List()
		assert.Empty(t, ids)
	})

	t.Run("Render writes to io.Writer", func(t *testing.T) {
		t.Parallel()

		renderer := &mockRenderer{id: "test-renderer"}
		proposal := mocks.NewMockAnalyzedProposal(t)

		var buf bytes.Buffer
		err := renderer.RenderTo(&buf, RenderRequest{}, proposal)
		require.NoError(t, err)
		assert.Equal(t, "mock output", buf.String())
	})
}
