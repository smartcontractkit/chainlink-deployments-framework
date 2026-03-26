package analyzer

import (
	"errors"
	"fmt"
	"slices"
)

// Registry manages analyzer registration and lookup.
type Registry struct {
	analyzers               map[string]BaseAnalyzer
	proposalAnalyzers       []ProposalAnalyzer
	batchOperationAnalyzers []BatchOperationAnalyzer
	callAnalyzers           []CallAnalyzer
	parameterAnalyzers      []ParameterAnalyzer
}

// NewRegistry creates a new analyzer registry.
func NewRegistry() *Registry {
	return &Registry{
		analyzers: make(map[string]BaseAnalyzer),
	}
}

// Register adds an analyzer to the registry.
// Returns an error if:
// - analyzer is nil
// - analyzer ID is empty
// - an analyzer with the same ID is already registered
// - analyzer type is unsupported
func (r *Registry) Register(baseAnalyzer BaseAnalyzer) error {
	if baseAnalyzer == nil {
		return errors.New("analyzer cannot be nil")
	}

	id := baseAnalyzer.ID()
	if id == "" {
		return errors.New("analyzer ID cannot be empty")
	}

	if _, exists := r.analyzers[id]; exists {
		return fmt.Errorf("analyzer with ID %q is already registered", id)
	}

	switch a := baseAnalyzer.(type) {
	case ProposalAnalyzer:
		r.proposalAnalyzers = append(r.proposalAnalyzers, a)
	case BatchOperationAnalyzer:
		r.batchOperationAnalyzers = append(r.batchOperationAnalyzers, a)
	case CallAnalyzer:
		r.callAnalyzers = append(r.callAnalyzers, a)
	case ParameterAnalyzer:
		r.parameterAnalyzers = append(r.parameterAnalyzers, a)
	default:
		return errors.New("unknown analyzer type")
	}

	r.analyzers[id] = baseAnalyzer

	return nil
}

// Get retrieves an analyzer by ID.
func (r *Registry) Get(id string) (BaseAnalyzer, bool) {
	analyzer, ok := r.analyzers[id]
	return analyzer, ok
}

// List returns all registered analyzer IDs.
func (r *Registry) List() []string {
	ids := make([]string, 0, len(r.analyzers))
	for id := range r.analyzers {
		ids = append(ids, id)
	}

	return ids
}

// All returns all registered analyzers in deterministic ID order.
func (r *Registry) All() []BaseAnalyzer {
	ids := r.List()
	slices.Sort(ids)

	analyzers := make([]BaseAnalyzer, 0, len(ids))
	for _, id := range ids {
		analyzers = append(analyzers, r.analyzers[id])
	}

	return analyzers
}

// ProposalAnalyzers returns registered proposal analyzers.
func (r *Registry) ProposalAnalyzers() []ProposalAnalyzer {
	analyzers := make([]ProposalAnalyzer, len(r.proposalAnalyzers))
	copy(analyzers, r.proposalAnalyzers)

	return analyzers
}

// BatchOperationAnalyzers returns registered batch operation analyzers.
func (r *Registry) BatchOperationAnalyzers() []BatchOperationAnalyzer {
	analyzers := make([]BatchOperationAnalyzer, len(r.batchOperationAnalyzers))
	copy(analyzers, r.batchOperationAnalyzers)

	return analyzers
}

// CallAnalyzers returns registered call analyzers.
func (r *Registry) CallAnalyzers() []CallAnalyzer {
	analyzers := make([]CallAnalyzer, len(r.callAnalyzers))
	copy(analyzers, r.callAnalyzers)

	return analyzers
}

// ParameterAnalyzers returns registered parameter analyzers.
func (r *Registry) ParameterAnalyzers() []ParameterAnalyzer {
	analyzers := make([]ParameterAnalyzer, len(r.parameterAnalyzers))
	copy(analyzers, r.parameterAnalyzers)

	return analyzers
}
