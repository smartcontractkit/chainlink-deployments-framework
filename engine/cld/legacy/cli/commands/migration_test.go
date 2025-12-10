package commands

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// fakeLoadRegistry is a no-op LoadRegistryFunc.
func fakeLoadRegistry(envKey string) (*changeset.ChangesetsRegistry, error) {
	return nil, errors.New("fakeLoadRegistry should not be called in metadata tests")
}

// fakeDecodeCtx is a no-op DecodeProposalCtxProvider.
func fakeDecodeCtx(env deployment.Environment) (analyzer.ProposalContext, error) {
	return nil, errors.New("fakeDecodeCtx should not be called in metadata tests")
}

func TestNewMigrationCmds_Structure(t *testing.T) {
	t.Parallel()
	c := NewCommands(nil)
	var domain domain.Domain
	root := c.NewMigrationCmds(domain, fakeLoadRegistry, fakeDecodeCtx)

	require.Equal(t, "migration", root.Use)

	subs := root.Commands()
	require.Len(t, subs, 5, "expected 5 subcommands under 'migration'")

	uses := make([]string, len(subs))
	for i, sc := range subs {
		uses[i] = sc.Use
	}
	require.ElementsMatch(t,
		[]string{"run", "list", "latest", "address-book", "datastore"},
		uses,
	)

	// The "environment" flag is persistent on root
	flag := root.PersistentFlags().Lookup("environment")
	require.NotNil(t, flag, "persistent flag 'environment' should exist")

	// address-book group
	abIdx := indexOf(subs, "address-book")
	require.NotEqual(t, -1, abIdx)
	abSubs := subs[abIdx].Commands()
	abUses := make([]string, len(abSubs))
	for i, sc := range abSubs {
		abUses[i] = sc.Use
	}
	require.ElementsMatch(t,
		[]string{"merge", "migrate", "remove"},
		abUses,
	)

	// datastore group
	dsIdx := indexOf(subs, "datastore")
	require.NotEqual(t, -1, dsIdx)
	dsSubs := subs[dsIdx].Commands()
	dsUses := make([]string, len(dsSubs))
	for i, sc := range dsSubs {
		dsUses[i] = sc.Use
	}
	require.ElementsMatch(t,
		[]string{"merge", "sync-to-catalog"},
		dsUses,
	)
}

func TestCommandMetadata(t *testing.T) {
	t.Parallel()
	c := NewCommands(nil)
	domain := domain.Domain{}

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
			name:                "run",
			cmdKey:              "run",
			wantUse:             "run",
			wantShort:           "Run a migration",
			wantLongPrefix:      "Run a specific migration",
			wantExampleContains: "ccip migration run --environment staging --name",
			wantFlags: []string{
				"environment", "changeset", "name", "force", "dry-run",
			},
		},
		{
			name:                "list",
			cmdKey:              "list",
			wantUse:             "list",
			wantShort:           "Lists migration keys",
			wantLongPrefix:      "Lists the migration keys",
			wantExampleContains: "ccip migration list --environment staging",
			wantFlags: []string{
				"environment",
			},
		},
		{
			name:                "latest",
			cmdKey:              "latest",
			wantUse:             "latest",
			wantShort:           "Get latest migration key",
			wantLongPrefix:      "Gets the latest migration key",
			wantExampleContains: "ccip migration latest --environment staging",
			wantFlags: []string{
				"environment",
			},
		},
		{
			name:                "address-book merge",
			cmdKey:              "address-book merge",
			wantUse:             "merge",
			wantShort:           "Merge the address book",
			wantLongPrefix:      "Merges the address book artifact",
			wantExampleContains: "address-book merge --environment staging --name",
			wantFlags: []string{
				"name", "timestamp",
			},
		},
		{
			name:                "address-book migrate",
			cmdKey:              "address-book migrate",
			wantUse:             "migrate",
			wantShort:           "Migrate address book to the new datastore format",
			wantLongPrefix:      "Converts the address book artifact format",
			wantExampleContains: "address-book migrate --environment staging",
			wantFlags:           []string{},
		},
		{
			name:                "address-book remove",
			cmdKey:              "address-book remove",
			wantUse:             "remove",
			wantShort:           "Remove migration address book",
			wantLongPrefix:      "Removes the address book entries",
			wantExampleContains: "address-book remove --environment staging --name",
			wantFlags: []string{
				"changeset", "name", "timestamp",
			},
		},
		{
			name:                "datastore merge",
			cmdKey:              "datastore merge",
			wantUse:             "merge",
			wantShort:           "Merge data stores",
			wantLongPrefix:      "Merge the data store for a migration",
			wantExampleContains: "datastore merge --environment staging --name",
			wantFlags: []string{
				"changeset", "name", "timestamp",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Give each subtest its own fresh command tree
			root := c.NewMigrationCmds(domain, fakeLoadRegistry, fakeDecodeCtx)

			t.Parallel()

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
				if flagName == "environment" {
					// persistent flag lives on root
					flag = root.PersistentFlags().Lookup("environment")
				} else {
					flag = cmd.Flags().Lookup(flagName)
					if flag == nil {
						flag = cmd.PersistentFlags().Lookup(flagName)
					}
				}
				require.NotNil(t, flag, "flag %q not found on %s", flagName, tc.name)
			}
		})
	}
}
