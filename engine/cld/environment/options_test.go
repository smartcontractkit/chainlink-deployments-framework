package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/cre"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
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

func Test_WithCRERunner(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	assert.Nil(t, opts.creRunner)

	runner := cre.NewCLIRunner("/path/to/cre")
	creR := cre.NewRunner(cre.WithCLI(runner))
	option := WithCRERunner(creR)
	option(opts)

	assert.Equal(t, creR, opts.creRunner)
	assert.Equal(t, runner, opts.creRunner.CLI())
}

func Test_newLoadConfig_defaultCRERunner(t *testing.T) {
	t.Parallel()

	cfg, err := newLoadConfig()
	require.NoError(t, err)

	require.Nil(t, cfg.creRunner, "CRE runner is nil by default; callers opt in via WithCRERunner")
}
