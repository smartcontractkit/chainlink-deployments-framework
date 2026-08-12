package template

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

var updateGolden = flag.Bool("update", false, "update golden files")

// assertGolden compares got against the golden file at testdata/<name>.
// With -update flag it writes the golden file instead of comparing.
func assertGolden(t *testing.T, name string, got string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", name)

	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, []byte(got), 0o600), "writing golden file %s", goldenPath) //nolint:gosec // G703: goldenPath is the in-repo testdata file, only written under -update by the developer
		return
	}

	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "reading golden file %s (run with -update to create it)", goldenPath)
	require.Equal(t, string(want), got, "output does not match golden file %s\n\nrun: go test ./... -run %s -update", goldenPath, t.Name())
}

// typedResolverFixtureResolver accepts resolverInputStruct as input but returns
// envInputFixtureConfig. This verifies that the generated YAML template shows
// the resolver's input type (resolverInputStruct), not the changeset's config
// type (envInputFixtureConfig).
func typedResolverFixtureResolver(in resolverInputStruct) (envInputFixtureConfig, error) {
	return envInputFixtureConfig{
		FeedURL:         "https://example.com/feed",
		PollIntervalSec: 30,
		Enabled:         true,
	}, nil
}

// TestGenerateMultiChangesetYAML_Golden is a golden-file test for the full
// end-to-end output of GenerateMultiChangesetYAML. It covers two variants:
//
//  1. env_input — a changeset wired with WithEnvInput, exercising the
//     cfg.InputType branch where the input type comes from the changeset's
//     generic type parameter C.
//  2. typed_resolver — a changeset wired with WithConfigResolver where the
//     resolver's input type (resolverInputStruct) differs from the changeset's
//     config type (envInputFixtureConfig), verifying the YAML shows the
//     resolver's input type.
//
// Run `go test ./... -run TestGenerateMultiChangesetYAML_Golden -update` to
// regenerate the golden files after intentional changes to the output format.
func TestGenerateMultiChangesetYAML_Golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		golden         string
		changesetNames []string
		regSetup       func() *cs.ChangesetsRegistry
		rmSetup        func() *fresolvers.ConfigResolverManager
	}{
		{
			name:           "env_input",
			golden:         "multi_changeset_env_input.golden.yaml",
			changesetNames: []string{"0002_env"},
			regSetup: func() *cs.ChangesetsRegistry {
				reg := cs.NewChangesetsRegistry()
				reg.Add("0002_env", cs.Configure(&envInputFixtureChangeset{}).WithEnvInput())

				return reg
			},
			rmSetup: func() *fresolvers.ConfigResolverManager {
				return fresolvers.NewConfigResolverManager()
			},
		},
		{
			name:           "typed_resolver",
			golden:         "multi_changeset_typed_resolver.golden.yaml",
			changesetNames: []string{"0003_typed"},
			regSetup: func() *cs.ChangesetsRegistry {
				reg := cs.NewChangesetsRegistry()
				reg.Add("0003_typed", cs.Configure(&typedResolverFixtureChangeset{}).WithConfigResolver(typedResolverFixtureResolver))

				return reg
			},
			rmSetup: func() *fresolvers.ConfigResolverManager {
				rm := fresolvers.NewConfigResolverManager()
				rm.Register(typedResolverFixtureResolver, fresolvers.ResolverInfo{Description: "typedResolverFixture"})

				return rm
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GenerateMultiChangesetYAML("mydomain", "testnet", tt.changesetNames, tt.regSetup(), tt.rmSetup(), 5)
			require.NoError(t, err)
			assertGolden(t, tt.golden, got)
		})
	}
}
