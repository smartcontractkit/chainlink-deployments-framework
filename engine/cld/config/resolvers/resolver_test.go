package resolvers

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for ConfigResolverManager
func TestConfigResolverManager_NewConfigResolverManager(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.byName)
	assert.Empty(t, manager.ListResolvers())
}

func TestConfigResolverManager_Register(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return "test-config", nil
	}
	info := ResolverInfo{
		Description: "Test resolver",
		ExampleYAML: "example: value",
	}

	// Test successful registration
	manager.Register(resolver, info)

	// Verify it was registered
	resolvers := manager.ListResolvers()
	assert.Len(t, resolvers, 1)

	// Test duplicate registration panics
	assert.Panics(t, func() {
		manager.Register(resolver, info)
	})
}

func TestConfigResolverManager_NameOf(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()
	resolver := func(input map[string]any) (any, error) {
		return "test-config", nil
	}
	info := ResolverInfo{Description: "Test resolver"}

	manager.Register(resolver, info)
	expectedName := manager.ListResolvers()[0]

	// Test successful name retrieval
	name := manager.NameOf(resolver)
	assert.Equal(t, expectedName, name)

	// Test non-registered resolver
	otherResolver := func(input map[string]any) (any, error) {
		return "other-config", nil
	}
	name = manager.NameOf(otherResolver)
	assert.Empty(t, name)
}

func TestConfigResolverManager_ListResolvers(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()

	// Test empty list
	resolvers := manager.ListResolvers()
	assert.Empty(t, resolvers)

	// Add multiple resolvers
	resolver1 := func(input map[string]any) (any, error) { return "config1", nil }
	resolver2 := func(input map[string]any) (any, error) { return "config2", nil }
	resolver3 := func(input map[string]any) (any, error) { return "config3", nil }

	manager.Register(resolver1, ResolverInfo{Description: "Resolver 1"})
	manager.Register(resolver2, ResolverInfo{Description: "Resolver 2"})
	manager.Register(resolver3, ResolverInfo{Description: "Resolver 3"})

	resolvers = manager.ListResolvers()
	assert.Len(t, resolvers, 3)

	// Test that they are sorted
	for i := 1; i < len(resolvers); i++ {
		assert.Less(t, resolvers[i-1], resolvers[i], "Resolvers should be sorted")
	}
}

func TestConfigResolverManager_ThreadSafety(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()

	// Test concurrent registration and retrieval
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := range [numGoroutines]struct{}{} {
		go func(index int) {
			defer func() { done <- true }()

			resolver := func(input map[string]any) (any, error) {
				return "config", nil
			}
			info := ResolverInfo{Description: "Test resolver"}

			// Some goroutines register, others read
			if index%2 == 0 {
				func() {
					defer func() {
						// Ignore panics from duplicate registration
						if r := recover(); r != nil {
							t.Logf("Recovered from panic: %v", r)
						}
					}()
					manager.Register(resolver, info)
				}()
			} else {
				manager.ListResolvers()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for range [numGoroutines]struct{}{} {
		<-done
	}

	// Verify manager is still functional
	resolvers := manager.ListResolvers()
	assert.NotEmpty(t, resolvers)
}

// Named test functions for extraction testing
func testResolverFunction1(input map[string]any) (any, error) { return struct{}{}, nil }
func testResolverFunction2(input map[string]any) (any, error) { return struct{}{}, nil }

func TestExtractFunctionName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fn           ConfigResolver
		expectedName string
	}{
		{
			name:         "named function 1",
			fn:           testResolverFunction1,
			expectedName: "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config.testResolverFunction1",
		},
		{
			name:         "named function 2",
			fn:           testResolverFunction2,
			expectedName: "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config.testResolverFunction2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			extractedName := extractFunctionName(tt.fn)
			assert.Equal(t, tt.expectedName, extractedName,
				"Should extract the correct function name")
		})
	}
}

func TestCallResolver_Success(t *testing.T) {
	t.Parallel()

	type TestConfigType struct {
		Value string
		Count int
	}

	resolver := func(input map[string]any) (TestConfigType, error) {
		return TestConfigType{
			Value: input["value"].(string),
			Count: int(input["count"].(float64)),
		}, nil
	}

	payload := json.RawMessage(`{"value": "test-value", "count": 42}`)

	result, err := CallResolver[TestConfigType](resolver, payload)
	require.NoError(t, err)
	assert.Equal(t, "test-value", result.Value)
	assert.Equal(t, 42, result.Count)
}

func TestCallResolver_InvalidJSON(t *testing.T) {
	t.Parallel()

	resolver := func(input map[string]any) (string, error) {
		return "config", nil
	}

	payload := json.RawMessage(`invalid json`)

	_, err := CallResolver[string](resolver, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal payload into")
}

func TestCallResolver_ResolverError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("resolver failed")
	resolver := func(input map[string]any) (string, error) {
		return "", expectedError
	}

	payload := json.RawMessage(`{"value": "test"}`)

	_, err := CallResolver[string](resolver, payload)
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestCallResolver_WrongReturnType(t *testing.T) {
	t.Parallel()

	resolver := func(input map[string]any) (int, error) {
		return 42, nil // Return int instead of string
	}

	payload := json.RawMessage(`{"value": "test"}`)

	_, err := CallResolver[string](resolver, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolver returned")
}

func TestCallResolver_InvalidResolverSignature(t *testing.T) {
	t.Parallel()

	// Function with wrong signature (no error return)
	invalidResolver := func(input map[string]any) string {
		return "config"
	}

	payload := json.RawMessage(`{"value": "test"}`)

	_, err := CallResolver[string](invalidResolver, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolver must be func(<In>) (<Out>, error)")
}

func TestRegister_InvalidSignature(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()

	// Test various invalid signatures
	testCases := []struct {
		name     string
		resolver any
	}{
		{
			name:     "no parameters",
			resolver: func() (string, error) { return "", nil },
		},
		{
			name:     "too many parameters",
			resolver: func(a, b string) (string, error) { return "", nil },
		},
		{
			name:     "no error return",
			resolver: func(input string) string { return "" },
		},
		{
			name:     "wrong error return type",
			resolver: func(input string) (string, string) { return "", "" },
		},
		{
			name:     "not a function",
			resolver: "not a function",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Panics(t, func() {
				manager.Register(tc.resolver, ResolverInfo{Description: "Test"})
			})
		})
	}
}

func TestRegister_EmptyName(t *testing.T) {
	t.Parallel()

	manager := NewConfigResolverManager()

	// This should panic because reflect.TypeOf(nil) will cause issues in signature validation
	assert.Panics(t, func() {
		manager.Register(nil, ResolverInfo{Description: "Test"})
	})
}

func TestCallResolver_PointerTypes(t *testing.T) {
	t.Parallel()

	type TestInput struct {
		Value string `json:"value"`
	}
	type TestOutput struct {
		Result string `json:"result"`
	}

	// Test resolver that expects pointer input
	resolver := func(input *TestInput) (TestOutput, error) {
		return TestOutput{Result: input.Value + "_processed"}, nil
	}

	payload := json.RawMessage(`{"value": "test"}`)

	result, err := CallResolver[TestOutput](resolver, payload)
	require.NoError(t, err)
	assert.Equal(t, "test_processed", result.Result)
}

func TestCallResolver_ValueTypes(t *testing.T) {
	t.Parallel()
	type TestInput struct {
		Value string `json:"value"`
	}
	type TestOutput struct {
		Result string `json:"result"`
	}

	// Test resolver that expects value input
	resolver := func(input TestInput) (TestOutput, error) {
		return TestOutput{Result: input.Value + "_processed"}, nil
	}

	payload := json.RawMessage(`{"value": "test"}`)

	result, err := CallResolver[TestOutput](resolver, payload)
	require.NoError(t, err)
	assert.Equal(t, "test_processed", result.Result)
}

func TestResolverInfo(t *testing.T) {
	t.Parallel()

	info := ResolverInfo{
		Description: "Test resolver description",
		ExampleYAML: "key: value",
	}

	assert.Equal(t, "Test resolver description", info.Description)
	assert.Equal(t, "key: value", info.ExampleYAML)
}

func TestConfigResolverType(t *testing.T) {
	t.Parallel()

	// Test that ConfigResolver can hold different function types
	var resolver1 ConfigResolver = func(input string) (string, error) { return "", nil }
	var resolver2 ConfigResolver = func(input map[string]any) (int, error) { return 0, nil }

	assert.NotNil(t, resolver1)
	assert.NotNil(t, resolver2)

	// Verify they have the expected function type
	assert.Equal(t, reflect.Func, reflect.TypeOf(resolver1).Kind())
	assert.Equal(t, reflect.Func, reflect.TypeOf(resolver2).Kind())
}
