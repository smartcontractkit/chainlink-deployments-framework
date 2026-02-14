package renderer

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock renderer for testing
type mockRenderer struct {
	id string
}

func (m *mockRenderer) ID() string {
	return m.id
}

func (m *mockRenderer) Render(ctx context.Context, w io.Writer, req types.RendererRequest, proposal types.AnalyzedProposal) error {
	_, err := w.Write([]byte("mock output"))
	return err
}

func TestRendererRegistry(t *testing.T) {
	t.Run("Register and Get renderer", func(t *testing.T) {
		registry := NewRendererRegistry()
		renderer := &mockRenderer{id: "test-renderer"}

		err := registry.Register(renderer)
		require.NoError(t, err)

		retrieved, ok := registry.Get("test-renderer")
		assert.True(t, ok)
		assert.Equal(t, renderer, retrieved)
	})

	t.Run("Register nil renderer returns error", func(t *testing.T) {
		registry := NewRendererRegistry()

		err := registry.Register(nil)
		require.ErrorContains(t, err, "cannot be nil")
	})

	t.Run("Register renderer with empty ID returns error", func(t *testing.T) {
		registry := NewRendererRegistry()
		renderer := &mockRenderer{id: ""}

		err := registry.Register(renderer)
		require.ErrorContains(t, err, "cannot be empty")
	})

	t.Run("Register duplicate ID returns error", func(t *testing.T) {
		registry := NewRendererRegistry()
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
		registry := NewRendererRegistry()

		retrieved, ok := registry.Get("non-existent")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("List renderers", func(t *testing.T) {
		registry := NewRendererRegistry()

		renderer1 := &mockRenderer{id: "renderer-1"}
		renderer2 := &mockRenderer{id: "renderer-2"}
		renderer3 := &mockRenderer{id: "renderer-3"}

		registry.Register(renderer1)
		registry.Register(renderer2)
		registry.Register(renderer3)

		ids := registry.List()
		assert.Len(t, ids, 3)
		assert.ElementsMatch(t, []string{"renderer-1", "renderer-2", "renderer-3"}, ids)
	})

	t.Run("List empty registry", func(t *testing.T) {
		registry := NewRendererRegistry()

		ids := registry.List()
		assert.Empty(t, ids)
	})

	t.Run("Render writes to io.Writer", func(t *testing.T) {
		renderer := &mockRenderer{id: "test-renderer"}
		ctx := t.Context()

		// Example: Write to a bytes.Buffer
		var buf bytes.Buffer
		err := renderer.Render(ctx, &buf, types.RendererRequest{}, nil)
		require.NoError(t, err)
		assert.Equal(t, "mock output", buf.String())
	})
}
