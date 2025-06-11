package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExampleOperationRegistry demonstrates how to create and use an OperationRegistry
// with operations being executed dynamically with different input/output types.
func ExampleOperationRegistry() {
	// example dependencies for operations
	type Deps1 struct{}
	type Deps2 struct{}

	// Create operations with different input/output types
	stringOp := NewOperation(
		"string-op",
		semver.MustParse("1.0.0"),
		"Echo string operation",
		func(e Bundle, deps Deps1, input string) (string, error) {
			return input, nil
		},
	)

	intOp := NewOperation(
		"int-op",
		semver.MustParse("1.0.0"),
		"Echo integer operation",
		func(e Bundle, deps Deps2, input int) (int, error) {
			return input, nil
		},
	)
	// Create registry with untyped operations
	registry := NewOperationRegistry(stringOp.AsUntyped(), intOp.AsUntyped())

	// Create execution environment
	b := NewBundle(context.Background, logger.Nop(), NewMemoryReporter(), WithOperationRegistry(registry))

	// Define inputs and dependencies for operations
	// inputs[0] is for stringOp, inputs[1] is for intOp
	// deps[0] is for stringOp, deps[1] is for intOp
	inputs := []any{"input1", 42}
	deps := []any{Deps1{}, Deps2{}}
	defs := []Definition{
		stringOp.Def(),
		intOp.Def(),
	}

	// dynamically retrieve and execute operations on different inputs
	for i, def := range defs {
		retrievedOp, err := registry.Retrieve(def)
		if err != nil {
			fmt.Println("error retrieving operation:", err)
			continue
		}

		report, err := ExecuteOperation(b, retrievedOp, deps[i], inputs[i])
		if err != nil {
			fmt.Println("error executing operation:", err)
			continue
		}

		fmt.Println("operation output:", report.Output)
	}

	// Output:
	// operation output: input1
	// operation output: 42
}

func TestOperationRegistry_Retrieve(t *testing.T) {
	t.Parallel()

	op1 := NewOperation(
		"test-op-1",
		semver.MustParse("1.0.0"),
		"Operation 1",
		func(e Bundle, deps OpDeps, input string) (string, error) { return input, nil },
	)
	op2 := NewOperation(
		"test-op-2",
		semver.MustParse("2.0.0"),
		"Operation 2",
		func(e Bundle, deps OpDeps, input int) (int, error) { return input * 2, nil },
	)

	tests := []struct {
		name        string
		operations  []*Operation[any, any, any]
		lookup      Definition
		wantErr     bool
		wantErrMsg  string
		wantID      string
		wantVersion string
	}{
		{
			name:       "empty registry",
			operations: nil,
			lookup:     Definition{ID: "test-op-1", Version: semver.MustParse("1.0.0")},
			wantErr:    true,
			wantErrMsg: "operation not found in registry",
		},
		{
			name:        "retrieval by exact match - first operation",
			operations:  []*Operation[any, any, any]{op1.AsUntyped(), op2.AsUntyped()},
			lookup:      Definition{ID: "test-op-1", Version: semver.MustParse("1.0.0")},
			wantErr:     false,
			wantID:      "test-op-1",
			wantVersion: "1.0.0",
		},
		{
			name:        "retrieval by exact match - second operation",
			operations:  []*Operation[any, any, any]{op1.AsUntyped(), op2.AsUntyped()},
			lookup:      Definition{ID: "test-op-2", Version: semver.MustParse("2.0.0")},
			wantErr:     false,
			wantID:      "test-op-2",
			wantVersion: "2.0.0",
		},
		{
			name:       "operation not found - non-existent ID",
			operations: []*Operation[any, any, any]{op1.AsUntyped(), op2.AsUntyped()},
			lookup:     Definition{ID: "non-existent", Version: semver.MustParse("1.0.0")},
			wantErr:    true,
			wantErrMsg: "operation not found in registry",
		},
		{
			name:       "operation not found - wrong version",
			operations: []*Operation[any, any, any]{op1.AsUntyped(), op2.AsUntyped()},
			lookup:     Definition{ID: "test-op-1", Version: semver.MustParse("3.0.0")},
			wantErr:    true,
			wantErrMsg: "operation not found in registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := NewOperationRegistry(tt.operations...)
			retrievedOp, err := registry.Retrieve(tt.lookup)

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, retrievedOp.ID())
				assert.Equal(t, tt.wantVersion, retrievedOp.Version())
			}
		})
	}
}
