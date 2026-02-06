package commands

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestNewDatastoreCmds_Structure(t *testing.T) {
	t.Parallel()
	c := NewCommands(logger.Nop())
	dom := domain.NewDomain("/tmp", "testdomain")
	root := c.NewDatastoreCmds(dom)

	require.Equal(t, "datastore", root.Use)

	subs := root.Commands()
	require.Len(t, subs, 2, "expected 2 subcommands under 'datastore'")

	uses := make([]string, len(subs))
	for i, sc := range subs {
		uses[i] = sc.Use
	}
	require.ElementsMatch(t,
		[]string{"merge", "sync-to-catalog"},
		uses,
	)

	// Environment flag is now local to each subcommand (not persistent on root)
	// Verify it exists on the merge subcommand
	mergeCmd, _, _ := root.Find([]string{"merge"})
	require.NotNil(t, mergeCmd)
	flag := mergeCmd.Flags().Lookup("environment")
	require.NotNil(t, flag, "flag 'environment' should exist on merge subcommand")
}

func TestDatastoreCommandMetadata(t *testing.T) {
	t.Parallel()
	c := NewCommands(logger.Nop())
	dom := domain.NewDomain("/tmp", "testdomain")

	tests := []struct {
		name                string
		cmdKey              string
		wantUse             string
		wantShort           string
		wantLongPrefix      string
		wantExampleContains string
		wantFlags           []string
	}{
		{
			name:                "merge",
			cmdKey:              "merge",
			wantUse:             "merge",
			wantShort:           "Merge datastore artifacts",
			wantLongPrefix:      "Merges the datastore artifact",
			wantExampleContains: "datastore merge --environment staging --name",
			wantFlags: []string{
				"name", "timestamp",
			},
		},
		{
			name:                "sync-to-catalog",
			cmdKey:              "sync-to-catalog",
			wantUse:             "sync-to-catalog",
			wantShort:           "Sync local datastore to catalog",
			wantLongPrefix:      "Syncs the entire local datastore",
			wantExampleContains: "datastore sync-to-catalog --environment staging",
			wantFlags:           []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Give each subtest its own fresh command tree
			root := c.NewDatastoreCmds(dom)

			parts := strings.Split(tc.cmdKey, " ")
			cmd, _, err := root.Find(parts)
			require.NoError(t, err)
			require.NotNil(t, cmd, "command not found: %s", tc.cmdKey)

			require.Equal(t, tc.wantUse, cmd.Use)
			require.Contains(t, cmd.Short, tc.wantShort)
			require.Contains(t, cmd.Long, tc.wantLongPrefix)
			require.Contains(t, cmd.Example, tc.wantExampleContains)

			for _, flagName := range tc.wantFlags {
				var flag *pflag.Flag
				// All flags are now local to subcommands
				flag = cmd.Flags().Lookup(flagName)
				if flag == nil {
					flag = cmd.PersistentFlags().Lookup(flagName)
				}
				require.NotNil(t, flag, "flag %q not found on %s", flagName, tc.name)
			}
		})
	}
}
