package mcmsutils

import (
	"testing"

	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSolanaInspectorFactory(t *testing.T) {
	t.Parallel()

	chain := stubSolanaChain()
	factory := newSolanaInspectorFactory(chain)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, chain.URL, factory.chain.URL)
	assert.Equal(t, chain.WSURL, factory.chain.WSURL)
}

func TestSolanaInspectorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubSolanaChain()
	factory := newSolanaInspectorFactory(chain)

	inspector, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, inspector)
}

func TestNewSolanaConverterFactory(t *testing.T) {
	t.Parallel()

	factory := newSolanaConverterFactory()

	require.NotNil(t, factory)
}

func TestSolanaConverterFactory_Make(t *testing.T) {
	t.Parallel()

	factory := newSolanaConverterFactory()

	converter, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, converter)
}

func TestNewSolanaExecutorFactory(t *testing.T) {
	t.Parallel()

	chain := stubSolanaChain()
	encoder := &mcmssolanasdk.Encoder{} // Empty encoder for testing

	factory := newSolanaExecutorFactory(chain, encoder)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, chain.URL, factory.chain.URL)
	assert.Equal(t, encoder, factory.encoder)
}

func TestSolanaExecutorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubSolanaChain()
	encoder := &mcmssolanasdk.Encoder{} // Empty encoder for testing

	factory := newSolanaExecutorFactory(chain, encoder)

	executor, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestNewSolanaTimelockExecutorFactory(t *testing.T) {
	t.Parallel()

	chain := stubSolanaChain()

	factory := newSolanaTimelockExecutorFactory(chain)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, chain.URL, factory.chain.URL)
}

func TestSolanaTimelockExecutorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubSolanaChain()
	factory := newSolanaTimelockExecutorFactory(chain)

	executor, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, executor)
}
