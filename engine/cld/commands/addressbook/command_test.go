package addressbook

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// newTestCommand creates a new command with a test domain rooted in a temp directory.
// This helper reduces boilerplate and ensures portability across platforms.
func newTestCommand(t *testing.T, deps Deps) (*cobra.Command, error) {
	t.Helper()

	return NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain(t.TempDir(), "testdomain"),
		Deps:   deps,
	})
}

// TestNewCommand_Structure verifies the command structure is correct.
func TestNewCommand_Structure(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})

	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Verify root command
	assert.Equal(t, "address-book", cmd.Use)
	assert.Equal(t, addressbookShort, cmd.Short)
	assert.NotEmpty(t, cmd.Long, "address-book command should have a Long description")

	// Verify NO persistent flags on parent (all flags are local to subcommands)
	envFlag := cmd.PersistentFlags().Lookup("environment")
	assert.Nil(t, envFlag, "environment flag should NOT be persistent")

	// Verify subcommands
	subs := cmd.Commands()
	require.Len(t, subs, 3)

	uses := make([]string, len(subs))
	for i, sc := range subs {
		uses[i] = sc.Use
	}
	assert.ElementsMatch(t, []string{"merge", "migrate", "remove"}, uses)
}

// TestNewCommand_MergeFlags verifies the merge subcommand has correct local flags.
func TestNewCommand_MergeFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	// Find the merge subcommand
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "merge" {
			found = true

			// Environment flag - local to merge
			e := sub.Flags().Lookup("environment")
			require.NotNil(t, e, "environment flag should be on merge")
			assert.Equal(t, "e", e.Shorthand)

			// Name flag
			n := sub.Flags().Lookup("name")
			require.NotNil(t, n)
			assert.Equal(t, "n", n.Shorthand)

			// Timestamp flag
			ts := sub.Flags().Lookup("timestamp")
			require.NotNil(t, ts)
			assert.Equal(t, "t", ts.Shorthand)

			break
		}
	}
	require.True(t, found, "merge subcommand not found")
}

// TestNewCommand_MigrateFlags verifies the migrate subcommand has correct local flags.
func TestNewCommand_MigrateFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	// Find the migrate subcommand
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "migrate" {
			found = true

			// Environment flag - local to migrate
			e := sub.Flags().Lookup("environment")
			require.NotNil(t, e, "environment flag should be on migrate")
			assert.Equal(t, "e", e.Shorthand)

			break
		}
	}
	require.True(t, found, "migrate subcommand not found")
}

// TestNewCommand_RemoveFlags verifies the remove subcommand has correct local flags.
func TestNewCommand_RemoveFlags(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	// Find the remove subcommand
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "remove" {
			found = true

			// Environment flag - local to remove
			e := sub.Flags().Lookup("environment")
			require.NotNil(t, e, "environment flag should be on remove")
			assert.Equal(t, "e", e.Shorthand)

			// Name flag
			n := sub.Flags().Lookup("name")
			require.NotNil(t, n)
			assert.Equal(t, "n", n.Shorthand)

			// Timestamp flag
			ts := sub.Flags().Lookup("timestamp")
			require.NotNil(t, ts)
			assert.Equal(t, "t", ts.Shorthand)

			break
		}
	}
	require.True(t, found, "remove subcommand not found")
}

// TestMerge_MissingEnvironmentFlagFails verifies required flag validation.
func TestMerge_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	cmd.SetArgs([]string{"merge", "--name", "test"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "environment" not set`)
}

// TestMerge_MissingNameFlagFails verifies required flag validation.
func TestMerge_MissingNameFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	cmd.SetArgs([]string{"merge", "-e", "staging"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "name" not set`)
}

// TestMerge_Success verifies successful merge.
func TestMerge_Success(t *testing.T) {
	t.Parallel()

	var mergerCalled bool
	var mergedName, mergedTimestamp string

	cmd, err := newTestCommand(t, Deps{
		AddressBookMerger: func(_ domain.EnvDir, name, timestamp string) error {
			mergerCalled = true
			mergedName = name
			mergedTimestamp = timestamp

			return nil
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, mergerCalled, "address book merger should be called")
	assert.Equal(t, "0001_deploy", mergedName)
	assert.Empty(t, mergedTimestamp)
	assert.Contains(t, out.String(), "✅ Merged address book")
}

// TestMerge_WithTimestamp verifies timestamp is passed through.
func TestMerge_WithTimestamp(t *testing.T) {
	t.Parallel()

	var mergedTimestamp string

	cmd, err := newTestCommand(t, Deps{
		AddressBookMerger: func(_ domain.EnvDir, _, timestamp string) error {
			mergedTimestamp = timestamp

			return nil
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy", "-t", "1234567890"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.Equal(t, "1234567890", mergedTimestamp)
}

// TestMerge_Error verifies error handling.
func TestMerge_Error(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("merge failed")

	cmd, err := newTestCommand(t, Deps{
		AddressBookMerger: func(_ domain.EnvDir, _, _ string) error {
			return expectedError
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "error during address book merge")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestMigrate_MissingEnvironmentFlagFails verifies required flag validation.
func TestMigrate_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	cmd.SetArgs([]string{"migrate"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "environment" not set`)
}

// TestMigrate_Success verifies successful migration.
func TestMigrate_Success(t *testing.T) {
	t.Parallel()

	var migratorCalled bool

	cmd, err := newTestCommand(t, Deps{
		AddressBookMigrator: func(_ domain.EnvDir) error {
			migratorCalled = true

			return nil
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"migrate", "-e", "staging"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, migratorCalled, "address book migrator should be called")
	assert.Contains(t, out.String(), "✅ Address book")
	assert.Contains(t, out.String(), "successfully migrated to the new datastore format")
}

// TestMigrate_Error verifies error handling.
func TestMigrate_Error(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("migration failed")

	cmd, err := newTestCommand(t, Deps{
		AddressBookMigrator: func(_ domain.EnvDir) error {
			return expectedError
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"migrate", "-e", "staging"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "error during address book migration")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestRemove_MissingEnvironmentFlagFails verifies required flag validation.
func TestRemove_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	cmd.SetArgs([]string{"remove", "--name", "test"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "environment" not set`)
}

// TestRemove_MissingNameFlagFails verifies required flag validation.
func TestRemove_MissingNameFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := newTestCommand(t, Deps{})
	require.NoError(t, err)

	cmd.SetArgs([]string{"remove", "-e", "staging"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "name" not set`)
}

// TestRemove_Success verifies successful removal.
func TestRemove_Success(t *testing.T) {
	t.Parallel()

	var removerCalled bool
	var removedName, removedTimestamp string

	cmd, err := newTestCommand(t, Deps{
		AddressBookRemover: func(_ domain.EnvDir, name, timestamp string) error {
			removerCalled = true
			removedName = name
			removedTimestamp = timestamp

			return nil
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"remove", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, removerCalled, "address book remover should be called")
	assert.Equal(t, "0001_deploy", removedName)
	assert.Empty(t, removedTimestamp)
	assert.Contains(t, out.String(), "✅ Removed address book entries")
}

// TestRemove_WithTimestamp verifies timestamp is passed through.
func TestRemove_WithTimestamp(t *testing.T) {
	t.Parallel()

	var removedTimestamp string

	cmd, err := newTestCommand(t, Deps{
		AddressBookRemover: func(_ domain.EnvDir, _, timestamp string) error {
			removedTimestamp = timestamp

			return nil
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"remove", "-e", "staging", "-n", "0001_deploy", "-t", "1234567890"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.Equal(t, "1234567890", removedTimestamp)
}

// TestRemove_Error verifies error handling.
func TestRemove_Error(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("remove failed")

	cmd, err := newTestCommand(t, Deps{
		AddressBookRemover: func(_ domain.EnvDir, _, _ string) error {
			return expectedError
		},
	})
	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"remove", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "error during address book remove")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestConfig_Validate verifies validation catches missing required fields.
func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("missing all required fields", func(t *testing.T) {
		t.Parallel()

		cfg := Config{}
		err := cfg.Validate()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Logger")
		assert.Contains(t, err.Error(), "Domain")
	})

	t.Run("missing Logger only", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Domain: domain.NewDomain(tempDir, "test"),
		}
		err := cfg.Validate()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Logger")
		assert.NotContains(t, err.Error(), "Domain")
	})

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Logger: logger.Nop(),
			Domain: domain.NewDomain(tempDir, "test"),
		}
		err := cfg.Validate()

		require.NoError(t, err)
	})
}

// TestNewCommand_InvalidConfigReturnsError verifies NewCommand returns error for invalid config.
func TestNewCommand_InvalidConfigReturnsError(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: nil, // Missing required field
		Domain: domain.NewDomain(t.TempDir(), "testdomain"),
	})

	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "Logger")
}
