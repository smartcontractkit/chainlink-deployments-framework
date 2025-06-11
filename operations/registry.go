package operations

import "errors"

// OperationRegistry is a store for operations that allows retrieval based on their definitions.
type OperationRegistry struct {
	ops []*Operation[any, any, any]
}

// NewOperationRegistry creates a new OperationRegistry with the provided untyped operations.
func NewOperationRegistry(ops ...*Operation[any, any, any]) OperationRegistry {
	return OperationRegistry{
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
