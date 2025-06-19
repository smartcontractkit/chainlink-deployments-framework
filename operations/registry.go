package operations

import (
	"errors"
	"fmt"
)

var ErrOperationNotFound = errors.New("operation not found in registry")

// OperationRegistry is a store for operations that allows retrieval based on their definitions.
type OperationRegistry struct {
	ops map[string]*Operation[any, any, any]
}

// NewOperationRegistry creates a new OperationRegistry with the provided untyped operations.
func NewOperationRegistry(ops ...*Operation[any, any, any]) *OperationRegistry {
	reg := &OperationRegistry{
		ops: make(map[string]*Operation[any, any, any]),
	}
	for _, op := range ops {
		key := generateRegistryKey(op.Def())
		reg.ops[key] = op
	}

	return reg
}

// Retrieve retrieves an operation from the store based on its definition.
// It returns an error if the operation is not found.
// The definition must match the operation's ID and version.
// Description of the definition is not used for retrieval, only ID and Version.
// This allows for simplicity in retrieving operations with the same ID and version only without having to provide the description.
// This is useful when definition has to be provided via manual input.
func (s OperationRegistry) Retrieve(def Definition) (*Operation[any, any, any], error) {
	key := generateRegistryKey(def)
	if op, ok := s.ops[key]; ok {
		return op, nil
	}

	return nil, ErrOperationNotFound
}

// RegisterOperation registers new operations in the registry.
// To register operations with different input, output, and dependency types,
// call RegisterOperation multiple times with different type parameters.
// If the same operation is registered multiple times, it will overwrite the previous one.
func RegisterOperation[D, I, O any](r *OperationRegistry, op ...*Operation[D, I, O]) {
	for _, o := range op {
		key := generateRegistryKey(o.Def())
		r.ops[key] = o.AsUntyped()
	}
}

// generateRegistryKey creates a unique key for the operation registry based on the operation's ID and version.
// This key is used to store and retrieve operations in the registry.
func generateRegistryKey(def Definition) string {
	return fmt.Sprintf("%s:%s", def.ID, def.Version)
}
