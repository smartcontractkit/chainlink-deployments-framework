package operations

import "errors"

// OperationRegistry is a store for operations that allows retrieval based on their definitions.
type OperationRegistry struct {
	ops []*Operation[any, any, any]
}

// NewOperationRegistry creates a new OperationRegistry with the provided untyped operations.
func NewOperationRegistry(ops ...*Operation[any, any, any]) *OperationRegistry {
	return &OperationRegistry{
		ops: ops,
	}
}

// Retrieve retrieves an operation from the store based on its definition.
// It returns an error if the operation is not found.
// The definition must match the operation's ID and version.
func (s OperationRegistry) Retrieve(def Definition) (*Operation[any, any, any], error) {
	for _, op := range s.ops {
		if op.ID() == def.ID && op.Version() == def.Version.String() {
			return op, nil
		}
	}

	return nil, errors.New("operation not found in registry")
}

// RegisterOperation registers new operations in the registry.
// To register operations with different input, output, and dependency types,
// call RegisterOperation multiple times with different type parameters.
func RegisterOperation[D, I, O any](r *OperationRegistry, op ...*Operation[D, I, O]) {
	for _, o := range op {
		r.ops = append(r.ops, o.AsUntyped())
	}
}
