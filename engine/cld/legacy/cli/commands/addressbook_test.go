package commands

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestNewAddressBookCmds_Structure(t *testing.T) {
	t.Parallel()
	c := NewCommands(nil)
	var dom domain.Domain
	root := c.NewAddressBookCmds(dom)

	require.Equal(t, "address-book", root.Use)

	subs := root.Commands()
	require.Len(t, subs, 3, "expected 3 subcommands under 'address-book'")

	uses := make([]string, len(subs))
	for i, sc := range subs {
		uses[i] = sc.Use
	}
	require.ElementsMatch(t,
		[]string{"merge", "migrate", "remove"},
		uses,
	)

	// The "environment" flag is persistent on root
	flag := root.PersistentFlags().Lookup("environment")
	require.NotNil(t, flag, "persistent flag 'environment' should exist")
}

func TestAddressBookCommandMetadata(t *testing.T) {
	t.Parallel()
	c := NewCommands(nil)
	dom := domain.Domain{}

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
			wantShort:           "Merge the address book",
			wantLongPrefix:      "Merges the address book artifact",
			wantExampleContains: "address-book merge --environment staging --name",
			wantFlags: []string{
				"name", "timestamp",
			},
		},
		{
			name:                "migrate",
			cmdKey:              "migrate",
			wantUse:             "migrate",
			wantShort:           "Migrate address book to the new datastore format",
			wantLongPrefix:      "Converts the address book artifact format",
			wantExampleContains: "address-book migrate --environment staging",
			wantFlags:           []string{},
		},
		{
			name:                "remove",
			cmdKey:              "remove",
			wantUse:             "remove",
			wantShort:           "Remove changeset address book entries",
			wantLongPrefix:      "Removes the address book entries",
			wantExampleContains: "address-book remove --environment staging --name",
			wantFlags: []string{
				"name", "timestamp",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Give each subtest its own fresh command tree
			root := c.NewAddressBookCmds(dom)

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
