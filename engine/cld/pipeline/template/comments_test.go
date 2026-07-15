package template

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommentExtractor_FieldComments_RealStruct(t *testing.T) {
	t.Parallel()

	extractor := newCommentExtractor()
	typ := reflect.TypeOf(commentedFixtureStruct{})

	// ChainSelector — single-line doc comment.
	chainComments := extractor.FieldComments(typ, "ChainSelector")
	require.NotEmpty(t, chainComments)
	require.Contains(t, chainComments[0], "ChainSelector is the EVM chain selector")

	// WorkflowName — multi-line doc comment (two // lines).
	wfComments := extractor.FieldComments(typ, "WorkflowName")
	require.Len(t, wfComments, 2)
	require.Contains(t, wfComments[0], "WorkflowName is the name of the CRE workflow")
	require.Contains(t, wfComments[1], "It must match the workflow name registered")

	// Decimals — multi-line doc comment.
	decComments := extractor.FieldComments(typ, "Decimals")
	require.NotEmpty(t, decComments)
	require.Contains(t, decComments[0], "Decimals is the on-chain precision")

	// NoCommentField — no doc comment → nil.
	require.Nil(t, extractor.FieldComments(typ, "NoCommentField"))

	// TrailingComment — has a trailing same-line comment.
	trailingComments := extractor.FieldComments(typ, "TrailingComment")
	require.NotEmpty(t, trailingComments)
}

func TestCommentExtractor_FieldComments_Caching(t *testing.T) {
	t.Parallel()

	extractor := newCommentExtractor()
	typ := reflect.TypeOf(commentedFixtureStruct{})

	// First call triggers packages.Load.
	first := extractor.FieldComments(typ, "ChainSelector")
	require.NotEmpty(t, first)

	// Second call should return the same result from cache.
	second := extractor.FieldComments(typ, "ChainSelector")
	require.Equal(t, first, second)

	// Verify the package is cached.
	pkgPath := typ.PkgPath()
	extractor.mu.RLock()
	_, cached := extractor.cache[pkgPath]
	extractor.mu.RUnlock()
	require.True(t, cached)
}

func TestCommentExtractor_FieldComments_NonExistentPackage(t *testing.T) {
	t.Parallel()

	extractor := newCommentExtractor()

	// A type with a fake package path — should return nil, no panic.
	// We simulate this by using a primitive type which has no PkgPath.
	typ := reflect.TypeOf("")
	require.Nil(t, extractor.FieldComments(typ, "SomeField"))
}

func TestCommentExtractor_FieldComments_AnonymousStruct(t *testing.T) {
	t.Parallel()

	extractor := newCommentExtractor()

	type anonymous struct {
		Field string `yaml:"field"`
	}

	typ := reflect.TypeOf(anonymous{})
	// Anonymous structs have no Name → should return nil.
	require.Nil(t, extractor.FieldComments(typ, "Field"))
}

func TestCommentExtractor_NilSafeInGenerateStructYAML(t *testing.T) {
	t.Parallel()

	type S struct {
		A string `yaml:"a"`
		B int    `yaml:"b"`
	}

	// Passing nil as comments should produce identical output to the
	// pre-comment-injection behavior.
	got, err := GenerateStructYAMLWithDepthLimit(reflect.TypeOf(S{}), "  ", 0, make(map[reflect.Type]bool), 5, nil)
	require.NoError(t, err)
	require.Equal(t, "  a: # string\n  b: # int\n", got)
}
