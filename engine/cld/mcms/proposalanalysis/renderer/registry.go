package renderer

import (
	"errors"
	"fmt"
)

// Registry manages renderer registration and lookup
type Registry struct {
	renderers map[string]Renderer
}

// NewRegistry creates a new renderer registry
func NewRegistry() *Registry {
	return &Registry{
		renderers: make(map[string]Renderer),
	}
}

// Register adds a renderer to the registry.
// Returns an error if:
// - renderer is nil
// - renderer ID is empty
// - a renderer with the same ID is already registered
func (r *Registry) Register(renderer Renderer) error {
	if renderer == nil {
		return errors.New("renderer cannot be nil")
	}

	id := renderer.ID()
	if id == "" {
		return errors.New("renderer ID cannot be empty")
	}

	if _, exists := r.renderers[id]; exists {
		return fmt.Errorf("renderer with ID %q is already registered", id)
	}

	r.renderers[id] = renderer

	return nil
}

// Get retrieves a renderer by ID
func (r *Registry) Get(id string) (Renderer, bool) {
	renderer, ok := r.renderers[id]
	return renderer, ok
}

// List returns all registered renderer IDs
func (r *Registry) List() []string {
	ids := make([]string, 0, len(r.renderers))
	for id := range r.renderers {
		ids = append(ids, id)
	}

	return ids
}
