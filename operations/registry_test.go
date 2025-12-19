package operations

import (
	"context"
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
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
	// Create registry with untyped operations by providing optional initial operation
	registry := NewOperationRegistry(stringOp.AsUntyped())

	// An alternative way to register additional operations without calling AsUntyped()
	RegisterOperation(registry, intOp)

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

func TestRegisterOperation(t *testing.T) {
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

	t.Run("register single operation", func(t *testing.T) {
		t.Parallel()

		registry := NewOperationRegistry()
		RegisterOperation(registry, op1)

		retrievedOp, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)
		assert.Equal(t, "test-op-1", retrievedOp.ID())
		assert.Equal(t, "1.0.0", retrievedOp.Version())
	})

	t.Run("register multiple operations with different types", func(t *testing.T) {
		t.Parallel()

		registry := NewOperationRegistry()
		// Register operations separately since they have different type parameters
		RegisterOperation(registry, op1)
		RegisterOperation(registry, op2)

		retrievedOp1, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)
		assert.Equal(t, "test-op-1", retrievedOp1.ID())

		retrievedOp2, err := registry.Retrieve(op2.Def())
		require.NoError(t, err)
		assert.Equal(t, "test-op-2", retrievedOp2.ID())
	})

	t.Run("overwrite existing operation", func(t *testing.T) {
		t.Parallel()

		op1Updated := NewOperation(
			"test-op-1",
			semver.MustParse("1.0.0"),
			"Operation 1 Updated",
			func(e Bundle, deps OpDeps, input string) (string, error) { return input + "-updated", nil },
		)

		registry := NewOperationRegistry()
		RegisterOperation(registry, op1)
		RegisterOperation(registry, op1Updated)

		retrievedOp, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)
		assert.Equal(t, "Operation 1 Updated", retrievedOp.Description())
	})
}

func TestRegisterOperationRelaxed(t *testing.T) {
	t.Parallel()

	type TestInput struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	op1 := NewOperation(
		"sum-op",
		semver.MustParse("1.0.0"),
		"Sum operation with struct input",
		func(e Bundle, deps OpDeps, input TestInput) (int, error) {
			return input.A + input.B, nil
		},
	)

	op2 := NewOperation(
		"multiply-op",
		semver.MustParse("1.0.0"),
		"Multiply operation",
		func(e Bundle, deps OpDeps, input int) (int, error) {
			return input * 2, nil
		},
	)

	t.Run("register and execute with map input from YAML", func(t *testing.T) {
		t.Parallel()

		registry := NewOperationRegistry()
		RegisterOperationRelaxed(registry, op1)

		retrievedOp, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)

		// Simulate input from YAML unmarshaling
		yamlInput := map[string]any{
			"a": 10,
			"b": 20,
		}

		bundle := NewBundle(context.Background, logger.Nop(), nil)
		result, err := retrievedOp.handler(bundle, OpDeps{}, yamlInput)
		require.NoError(t, err)
		assert.Equal(t, 30, result)
	})

	t.Run("register multiple operations with relaxed typing", func(t *testing.T) {
		t.Parallel()

		registry := NewOperationRegistry()
		// Register operations separately since they have different type parameters
		RegisterOperationRelaxed(registry, op1)
		RegisterOperationRelaxed(registry, op2)

		// Verify both operations are registered
		retrievedOp1, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)
		assert.Equal(t, "sum-op", retrievedOp1.ID())

		retrievedOp2, err := registry.Retrieve(op2.Def())
		require.NoError(t, err)
		assert.Equal(t, "multiply-op", retrievedOp2.ID())
	})

	t.Run("overwrite operation with relaxed version", func(t *testing.T) {
		t.Parallel()

		op1Strict := NewOperation(
			"sum-op",
			semver.MustParse("1.0.0"),
			"Sum operation strict",
			func(e Bundle, deps OpDeps, input TestInput) (int, error) {
				return input.A + input.B + 1, nil
			},
		)

		registry := NewOperationRegistry()
		RegisterOperation(registry, op1Strict)
		RegisterOperationRelaxed(registry, op1) // Should overwrite

		retrievedOp, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)

		// Use map input which would fail with strict version
		yamlInput := map[string]any{
			"a": 10,
			"b": 20,
		}

		bundle := NewBundle(context.Background, logger.Nop(), nil)
		result, err := retrievedOp.handler(bundle, OpDeps{}, yamlInput)
		require.NoError(t, err)
		// If it was the strict version, result would be 31
		assert.Equal(t, 30, result)
	})

	t.Run("handle type conversion errors gracefully", func(t *testing.T) {
		t.Parallel()

		registry := NewOperationRegistry()
		RegisterOperationRelaxed(registry, op1)

		retrievedOp, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)

		// Provide input that cannot be converted
		invalidInput := "not a struct"

		bundle := NewBundle(context.Background, logger.Nop(), nil)
		_, err = retrievedOp.handler(bundle, OpDeps{}, invalidInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input type mismatch")
	})

	t.Run("execute operation with direct struct input", func(t *testing.T) {
		t.Parallel()

		registry := NewOperationRegistry()
		RegisterOperationRelaxed(registry, op1)

		retrievedOp, err := registry.Retrieve(op1.Def())
		require.NoError(t, err)

		// Even with relaxed typing, direct struct input should work
		directInput := TestInput{A: 5, B: 15}

		bundle := NewBundle(context.Background, logger.Nop(), nil)
		result, err := retrievedOp.handler(bundle, OpDeps{}, directInput)
		require.NoError(t, err)
		assert.Equal(t, 20, result)
	})
}
