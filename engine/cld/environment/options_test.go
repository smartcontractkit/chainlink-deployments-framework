package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

func Test_WithAnvilKeyAsDeployer(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	require.False(t, opts.anvilKeyAsDeployer)

	option := WithAnvilKeyAsDeployer()
	option(opts)

	assert.True(t, opts.anvilKeyAsDeployer)
}

func Test_WithReporter(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	assert.Nil(t, opts.reporter)

	reporter := foperations.NewMemoryReporter()
	option := WithReporter(reporter)
	option(opts)

	assert.Equal(t, reporter, opts.reporter)
}

func Test_WithOutJD(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	require.False(t, opts.withoutJD)

	option := WithoutJD()
	option(opts)

	assert.True(t, opts.withoutJD)
}

func Test_OnlyLoadChainsFor(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	assert.Nil(t, opts.chainSelectorsToLoad)

	chainSelectors := []uint64{1, 2, 3}

	option := OnlyLoadChainsFor(chainSelectors)
	option(opts)

	assert.Equal(t, chainSelectors, opts.chainSelectorsToLoad)

	// Test with nil chainSelectors
	opts = &LoadConfig{}
	option = OnlyLoadChainsFor(nil)
	option(opts)
	assert.Equal(t, []uint64{}, opts.chainSelectorsToLoad)

	// Test with empty chainSelectors
	opts = &LoadConfig{}
	option = OnlyLoadChainsFor([]uint64{})
	option(opts)
	assert.Equal(t, []uint64{}, opts.chainSelectorsToLoad)
}

func Test_WithOperationRegistry(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	assert.Nil(t, opts.operationRegistry)

	registry := foperations.NewOperationRegistry()
	option := WithOperationRegistry(registry)
	option(opts)

	assert.Equal(t, registry, opts.operationRegistry)
}

func Test_WithLogger(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	assert.Nil(t, opts.lggr)

	lggr := logger.Test(t)
	option := WithLogger(lggr)
	option(opts)

	assert.Equal(t, lggr, opts.lggr)
}

func Test_WithDryRunJobDistributor(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	require.False(t, opts.useDryRunJobDistributor)

	option := WithDryRunJobDistributor()
	option(opts)

	assert.True(t, opts.useDryRunJobDistributor)
}
