package formatter

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

// FormatterRegistry manages formatter registration and lookup
type FormatterRegistry struct {
	formatters map[string]types.Formatter
}

// NewFormatterRegistry creates a new formatter registry
func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[string]types.Formatter),
	}
}

// Register adds a formatter to the registry.
// Returns an error if:
// - formatter is nil
// - formatter ID is empty
// - a formatter with the same ID is already registered
func (r *FormatterRegistry) Register(formatter types.Formatter) error {
	if formatter == nil {
		return fmt.Errorf("formatter cannot be nil")
	}

	id := formatter.ID()
	if id == "" {
		return fmt.Errorf("formatter ID cannot be empty")
	}

	if _, exists := r.formatters[id]; exists {
		return fmt.Errorf("formatter with ID %q is already registered", id)
	}

	r.formatters[id] = formatter
	return nil
}

// Get retrieves a formatter by ID
func (r *FormatterRegistry) Get(id string) (types.Formatter, bool) {
	f, ok := r.formatters[id]
	return f, ok
}

// List returns all registered formatter IDs
func (r *FormatterRegistry) List() []string {
	ids := make([]string, 0, len(r.formatters))
	for id := range r.formatters {
		ids = append(ids, id)
	}
	return ids
}
