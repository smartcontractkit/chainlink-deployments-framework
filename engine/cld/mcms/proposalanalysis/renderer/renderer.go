package renderer

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

// RendererRegistry manages renderer registration and lookup
type RendererRegistry struct {
	renderers map[string]types.Renderer
}

// NewRendererRegistry creates a new renderer registry
func NewRendererRegistry() *RendererRegistry {
	return &RendererRegistry{
		renderers: make(map[string]types.Renderer),
	}
}

// Register adds a renderer to the registry.
// Returns an error if:
// - renderer is nil
// - renderer ID is empty
// - a renderer with the same ID is already registered
func (r *RendererRegistry) Register(renderer types.Renderer) error {
	if renderer == nil {
		return fmt.Errorf("renderer cannot be nil")
	}

	id := renderer.ID()
	if id == "" {
		return fmt.Errorf("renderer ID cannot be empty")
	}

	if _, exists := r.renderers[id]; exists {
		return fmt.Errorf("renderer with ID %q is already registered", id)
	}

	r.renderers[id] = renderer
	return nil
}

// Get retrieves a renderer by ID
func (r *RendererRegistry) Get(id string) (types.Renderer, bool) {
	renderer, ok := r.renderers[id]
	return renderer, ok
}

// List returns all registered renderer IDs
func (r *RendererRegistry) List() []string {
	ids := make([]string, 0, len(r.renderers))
	for id := range r.renderers {
		ids = append(ids, id)
	}
	return ids
}
