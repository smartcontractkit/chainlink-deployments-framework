package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment(t *testing.T) {
	t.Parallel()

	t.Run("required flag", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		var env string
		Environment(cmd, &env, true)

		f := cmd.Flags().Lookup("environment")
		require.NotNil(t, f)
		assert.Equal(t, "e", f.Shorthand)
		assert.Empty(t, f.DefValue)

		err := cmd.ValidateRequiredFlags()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("optional flag", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		var env string
		Environment(cmd, &env, false)

		err := cmd.ValidateRequiredFlags()
		require.NoError(t, err)
	})

	t.Run("value binding", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
		var env string
		Environment(cmd, &env, false)

		cmd.SetArgs([]string{"-e", "staging"})
		err := cmd.Execute()

		require.NoError(t, err)
		assert.Equal(t, "staging", env)
	})
}

func TestPrint(t *testing.T) {
	t.Parallel()

	t.Run("default false", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		var printFlag bool
		Print(cmd, &printFlag)

		f := cmd.Flags().Lookup("print")
		require.NotNil(t, f)
		assert.Empty(t, f.Shorthand)
		assert.Equal(t, "false", f.DefValue)
	})

	t.Run("value binding", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
		var printFlag bool
		Print(cmd, &printFlag)

		cmd.SetArgs([]string{"--print"})
		err := cmd.Execute()

		require.NoError(t, err)
		assert.True(t, printFlag)
	})
}

func TestOutput(t *testing.T) {
	t.Parallel()

	t.Run("new flag name", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
		var out string
		Output(cmd, &out, "")

		f := cmd.Flags().Lookup("out")
		require.NotNil(t, f)
		assert.Equal(t, "o", f.Shorthand)

		cmd.SetArgs([]string{"-o", "/new/path.json"})
		err := cmd.Execute()

		require.NoError(t, err)
		assert.Equal(t, "/new/path.json", out)
	})

	t.Run("deprecated alias still works", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, args []string) {}}
		var out string
		Output(cmd, &out, "")

		f := cmd.Flags().Lookup("outputPath")
		require.NotNil(t, f)

		cmd.SetArgs([]string{"--outputPath", "/old/path.json"})
		err := cmd.Execute()

		require.NoError(t, err)
		assert.Equal(t, "/old/path.json", out)
	})

	t.Run("default value", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		var out string
		Output(cmd, &out, "default.json")

		f := cmd.Flags().Lookup("out")
		require.NotNil(t, f)
		assert.Equal(t, "default.json", f.DefValue)
	})
}

func TestFlagsAreLocal(t *testing.T) {
	t.Parallel()

	parent := &cobra.Command{Use: "parent"}
	child := &cobra.Command{Use: "child", Run: func(cmd *cobra.Command, args []string) {}}

	var env string
	Environment(parent, &env, false)
	parent.AddCommand(child)

	// Child should NOT have the environment flag
	f := child.Flags().Lookup("environment")
	assert.Nil(t, f, "local flags should not be inherited by subcommands")

	// But parent should have it
	pf := parent.Flags().Lookup("environment")
	assert.NotNil(t, pf)
}
