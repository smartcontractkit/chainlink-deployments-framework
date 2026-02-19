package operations

import (
	"context"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-deployments-framework/helper"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type OpDeps struct{}

type OpInput struct {
	A int
	B int
}

func Test_NewOperation(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	description := "test operation"
	handler := func(b Bundle, deps OpDeps, input OpInput) (output int, err error) {
		return input.A + input.B, nil
	}

	op := NewOperation("sum", version, description, handler)

	assert.Equal(t, "sum", op.ID())
	assert.Equal(t, version.String(), op.Version())
	assert.Equal(t, description, op.Description())
	assert.Equal(t, op.def, op.Def())
	res, err := op.handler(Bundle{}, OpDeps{}, OpInput{1, 2})
	require.NoError(t, err)
	assert.Equal(t, 3, res)
}

func Test_Operation_Execute(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	description := "test operation"
	log, observedLog := logger.TestObserved(t, zapcore.InfoLevel)

	// simulate an addition operation
	handler := func(b Bundle, deps OpDeps, input OpInput) (output int, err error) {
		return input.A + input.B, nil
	}

	op := NewOperation("sum", version, description, handler)
	e := NewBundle(context.Background, log, nil)
	input := OpInput{
		A: 1,
		B: 2,
	}

	output, err := op.execute(e, OpDeps{}, input)

	require.NoError(t, err)
	assert.Equal(t, 3, output)

	require.Equal(t, 1, observedLog.Len())
	entry := observedLog.All()[0]
	assert.Equal(t, "Executing operation", entry.Message)
	assert.Equal(t, "sum", entry.ContextMap()["id"])
	assert.Equal(t, version.String(), entry.ContextMap()["version"])
	assert.Equal(t, description, entry.ContextMap()["description"])
}

func Test_Operation_WithEmptyInput(t *testing.T) {
	t.Parallel()

	handler := func(b Bundle, deps OpDeps, _ EmptyInput) (int, error) {
		return 1, nil
	}
	op := NewOperation("return-1", semver.MustParse("1.0.0"), "return 1", handler)

	out, err := op.execute(NewBundle(context.Background, logger.Test(t), nil), OpDeps{}, EmptyInput{})

	require.NoError(t, err)
	assert.Equal(t, 1, out)
}

func Test_Operation_AsUntyped(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	description := "test operation"
	handler1 := func(b Bundle, deps OpDeps, input OpInput) (output int, err error) {
		return input.A + input.B, nil
	}
	typedOp := NewOperation("sum", version, description, handler1)

	untypedOp := typedOp.AsUntyped()
	bundle := NewBundle(t.Context, logger.Test(t), nil)

	assert.Equal(t, "sum", untypedOp.ID())
	assert.Equal(t, version.String(), untypedOp.Version())
	assert.Equal(t, description, untypedOp.Description())

	tests := []struct {
		name        string
		deps        any
		input       any
		wantResult  any
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid input and dependencies",
			deps:       OpDeps{},
			input:      OpInput{A: 3, B: 4},
			wantResult: 7,
			wantErr:    false,
		},
		{
			name:        "invalid input type",
			deps:        OpDeps{},
			input:       struct{ C int }{C: 5},
			wantErr:     true,
			errContains: "input type mismatch",
		},
		{
			name:        "invalid dependencies type",
			deps:        "invalid",
			input:       OpInput{A: 1, B: 2},
			wantErr:     true,
			errContains: "dependencies type mismatch",
		},
		{
			name: "input from YAML unmarshaling (map[string]interface{}) - should fail with AsUntyped",
			deps: OpDeps{},
			input: map[string]interface{}{
				"A": 5,
				"B": 3,
			},
			wantErr:     true,
			errContains: "input type mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := untypedOp.handler(bundle, tt.deps, tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func Test_Operation_AsUntypedRelaxed_WithYAMLUnmarshaling(t *testing.T) {
	t.Parallel()

	handler := func(b Bundle, deps OpDeps, input OpInput) (int, error) {
		return input.A + input.B, nil
	}
	typedOp := NewOperation("sum", semver.MustParse("1.0.0"), "test operation", handler)
	untypedOp := typedOp.AsUntypedRelaxed()
	bundle := NewBundle(t.Context, logger.Test(t), nil)

	// Simulate YAML unmarshaling scenario
	yamlData := `
A: 10
B: 20
`
	var yamlInput interface{}
	err := yaml.Unmarshal([]byte(yamlData), &yamlInput)
	require.NoError(t, err)

	// Coerce big int strings as YAML parsing may interpret large numbers as strings
	matchFunc := helper.DefaultMatchKeysToFix
	yamlInput = helper.CoerceBigIntStringsForKeys(yamlInput, matchFunc)

	// The yamlInput is now a map[string]interface{}, which should work with AsUntypedRelaxed
	result, err := untypedOp.handler(bundle, OpDeps{}, yamlInput)
	require.NoError(t, err)
	assert.Equal(t, 30, result)
}
