package resolvers

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// ConfigResolverManager manages config resolvers with thread-safe operations
type ConfigResolverManager struct {
	mu     sync.RWMutex
	byName map[string]registered // name → {fn, info}
}

type registered struct {
	fn   ConfigResolver
	info ResolverInfo
}

// NewConfigResolverManager creates a new ConfigResolverManager
func NewConfigResolverManager() *ConfigResolverManager {
	return &ConfigResolverManager{
		byName: map[string]registered{},
	}
}

// Register binds an explicit name to a resolver and stores its metadata.
// It panics if the name is already taken.
func (m *ConfigResolverManager) Register(
	fn ConfigResolver,
	info ResolverInfo,
) {
	name := extractFunctionName(fn)

	if name == "" {
		panic("resolver name must not be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, dup := m.byName[name]; dup {
		panic(fmt.Sprintf("resolver %q already registered", name))
	}

	// Signature check and type discovery
	rf := reflect.TypeOf(fn)
	if rf.Kind() != reflect.Func || rf.NumIn() != 1 || rf.NumOut() != 2 ||
		!rf.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		panic(fmt.Sprintf(
			"resolver %q must be func(<In>) (<Out>, error) – got %s", name, rf,
		))
	}

	m.byName[name] = registered{fn: fn, info: info}
}

// NameOf returns the registered name for the given resolver, or empty string.
func (m *ConfigResolverManager) NameOf(r ConfigResolver) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Generate the name consistently
	name := extractFunctionName(r)

	// Verify this function is actually registered with this manager
	if registered, exists := m.byName[name]; exists {
		// Double-check it's the same function by comparing pointers
		if reflect.ValueOf(registered.fn).Pointer() == reflect.ValueOf(r).Pointer() {
			return name
		}
	}

	return ""
}

// ListResolvers returns all registered names in deterministic order.
func (m *ConfigResolverManager) ListResolvers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.byName))
	for n := range m.byName {
		names = append(names, n)
	}
	sort.Strings(names)

	return names
}

// extractFunctionName extracts the full qualified function name (with package path) from a function using reflection
// This is used as the key to avoid naming collisions between packages
func extractFunctionName(fn ConfigResolver) string {
	// Get the full function name with package path
	fullName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()

	if fullName == "" {
		return "unknown_resolver"
	}

	return fullName
}

// ConfigResolver can be *any* function whose signature is:
//
//	func(<Input>) (<Output>, error)
//
// The concrete types are discovered at runtime via reflection. Signature check happens at registration time.
type ConfigResolver any

// ResolverInfo contains metadata about a config resolver
type ResolverInfo struct {
	Description string
	ExampleYAML string
}

// CallResolver unmarshals raw JSON into the input type expected by `resolver`,
// invokes it, and converts the first return value to the requested generic
// type C.
func CallResolver[C any](resolver ConfigResolver, payload json.RawMessage) (C, error) {
	var zero C

	rVal := reflect.ValueOf(resolver)
	rType := rVal.Type() // func(<In>) (<Out>, error)

	// Basic validation (double check – already done at registration time).
	if rType.Kind() != reflect.Func || rType.NumIn() != 1 || rType.NumOut() != 2 ||
		!rType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return zero, errors.New("resolver must be func(<In>) (<Out>, error)")
	}

	inType := rType.In(0)

	// Allocate a new value of the required input type and unmarshal into it.
	var arg reflect.Value
	if inType.Kind() == reflect.Ptr {
		// If the function expects a pointer, create the underlying type and get a pointer to it
		elemType := inType.Elem()
		elemPtr := reflect.New(elemType)
		decoder := json.NewDecoder(strings.NewReader(string(payload)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(elemPtr.Interface()); err != nil {
			return zero, fmt.Errorf("unmarshal payload into %v: %w", inType, err)
		}
		arg = elemPtr
	} else {
		// If the function expects a value, create a pointer, unmarshal, then get the value
		inPtr := reflect.New(inType)
		decoder := json.NewDecoder(strings.NewReader(string(payload)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(inPtr.Interface()); err != nil {
			return zero, fmt.Errorf("unmarshal payload into %v: %w", inType, err)
		}
		arg = inPtr.Elem()
	}

	// Invoke the resolver.
	outs := rVal.Call([]reflect.Value{arg})
	if errIface := outs[1].Interface(); errIface != nil {
		return zero, errIface.(error)
	}

	// Convert the first return value to C.
	outIface := outs[0].Interface()
	cfg, ok := outIface.(C)
	if !ok {
		return zero, fmt.Errorf("resolver returned %T, expected %T", outIface, zero)
	}

	return cfg, nil
}
