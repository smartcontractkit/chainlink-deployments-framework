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

	runner := &cre.CLIRunner{BinaryPath: "/path/to/cre"}
	option := WithCRERunner(runner)
	option(opts)

	assert.Equal(t, runner, opts.creRunner)
}

func Test_WithCREBinaryPath(t *testing.T) {
	t.Parallel()

	opts := &LoadConfig{}
	assert.Empty(t, opts.creBinaryPath)

	binaryPath := "/custom/path/to/cre"
	option := WithCREBinaryPath(binaryPath)
	option(opts)

	assert.Equal(t, binaryPath, opts.creBinaryPath)
}

func Test_resolveCRERunner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		override   cre.Runner
		binaryPath string
	}{
		{
			name:       "override takes precedence",
			override:   &cre.CLIRunner{BinaryPath: "/override/cre"},
			binaryPath: "/default/cre",
		},
		{
			name:       "no override uses binary path",
			override:   nil,
			binaryPath: "/custom/cre",
		},
		{
			name:       "empty_binary_path_uses_cli_default",
			override:   nil,
			binaryPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolveCRERunner(tt.override, tt.binaryPath)

			if tt.override != nil {
				assert.Equal(t, tt.override, got)
				return
			}

			require.NotNil(t, got)
			cliRunner, ok := got.(*cre.CLIRunner)
			require.True(t, ok, "expected *cre.CLIRunner, got %T", got)
			assert.Equal(t, tt.binaryPath, cliRunner.BinaryPath)
		})
	}
}
