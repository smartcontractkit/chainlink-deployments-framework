package formatter

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

// FormatterRequest encapsulates the context passed to formatter methods.
type FormatterRequest struct {
	Domain          string
	EnvironmentName string
}

// Formatter transforms an AnalyzedProposal into a specific output format
type Formatter interface {
	ID() string
	Format(ctx context.Context, req FormatterRequest, proposal types.AnalyzedProposal) ([]byte, error)
}

// FormatterRegistry manages formatter registration and lookup
type FormatterRegistry struct {
	formatters map[string]Formatter
}

// NewFormatterRegistry creates a new formatter registry
func NewFormatterRegistry() *FormatterRegistry {
	return &FormatterRegistry{
		formatters: make(map[string]Formatter),
	}
}

// Register adds a formatter to the registry.
// Returns an error if:
// - formatter is nil
// - formatter ID is empty
// - a formatter with the same ID is already registered
func (r *FormatterRegistry) Register(formatter Formatter) error {
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
func (r *FormatterRegistry) Get(id string) (Formatter, bool) {
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
