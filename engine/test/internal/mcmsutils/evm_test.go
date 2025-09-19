package mcmsutils

import (
	"testing"

	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEVMInspectorFactory(t *testing.T) {
	t.Parallel()

	chain := stubEVMChain()
	factory := newEVMInspectorFactory(chain)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
}

func TestEVMInspectorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubEVMChain()
	factory := newEVMInspectorFactory(chain)

	inspector, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, inspector)
}

func TestNewEVMConverterFactory(t *testing.T) {
	t.Parallel()

	factory := newEVMConverterFactory()

	require.NotNil(t, factory)
}

func TestEVMConverterFactory_Make(t *testing.T) {
	t.Parallel()

	factory := newEVMConverterFactory()

	converter, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, converter)
}

func TestNewEVMExecutorFactory(t *testing.T) {
	t.Parallel()

	chain := stubEVMChain()
	encoder := &mcmsevmsdk.Encoder{} // Empty encoder for testing

	factory := newEVMExecutorFactory(chain, encoder)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, encoder, factory.encoder)
}

func TestEVMExecutorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubEVMChain()
	encoder := &mcmsevmsdk.Encoder{} // Empty encoder for testing

	factory := newEVMExecutorFactory(chain, encoder)

	executor, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestNewEVMTimelockExecutorFactory(t *testing.T) {
	t.Parallel()

	chain := stubEVMChain()

	factory := newEVMTimelockExecutorFactory(chain)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
}

func TestEVMTimelockExecutorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubEVMChain()
	factory := newEVMTimelockExecutorFactory(chain)

	executor, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, executor)
}
