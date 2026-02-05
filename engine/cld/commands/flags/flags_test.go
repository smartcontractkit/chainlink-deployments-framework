package flags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironment(t *testing.T) {
	t.Parallel()

	t.Run("flag properties", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		Environment(cmd)

		f := cmd.Flags().Lookup("environment")
		require.NotNil(t, f)
		assert.Equal(t, "e", f.Shorthand)
		assert.Empty(t, f.DefValue)
	})

	t.Run("is required", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		Environment(cmd)

		err := cmd.ValidateRequiredFlags()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
	})

	t.Run("value retrieval", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, _ []string) {}}
		Environment(cmd)

		cmd.SetArgs([]string{"-e", "staging"})
		err := cmd.Execute()

		require.NoError(t, err)
		env, _ := cmd.Flags().GetString("environment")
		assert.Equal(t, "staging", env)
	})
}

func TestPrint(t *testing.T) {
	t.Parallel()

	t.Run("default true", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		Print(cmd)

		f := cmd.Flags().Lookup("print")
		require.NotNil(t, f)
		assert.Empty(t, f.Shorthand)
		assert.Equal(t, "true", f.DefValue)
	})

	t.Run("value retrieval", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, _ []string) {}}
		Print(cmd)

		cmd.SetArgs([]string{"--print"})
		err := cmd.Execute()

		require.NoError(t, err)
		shouldPrint, _ := cmd.Flags().GetBool("print")
		assert.True(t, shouldPrint)
	})
}

func TestOutput(t *testing.T) {
	t.Parallel()

	t.Run("new flag name", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, _ []string) {}}
		Output(cmd, "")

		f := cmd.Flags().Lookup("out")
		require.NotNil(t, f)
		assert.Equal(t, "o", f.Shorthand)

		cmd.SetArgs([]string{"-o", "/new/path.json"})
		err := cmd.Execute()

		require.NoError(t, err)
		out, _ := cmd.Flags().GetString("out")
		assert.Equal(t, "/new/path.json", out)
	})

	t.Run("deprecated alias still works", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test", Run: func(cmd *cobra.Command, _ []string) {}}
		Output(cmd, "")

		// --outputPath is normalized to --out, so Lookup finds the same flag
		cmd.SetArgs([]string{"--outputPath", "/old/path.json"})
		err := cmd.Execute()

		require.NoError(t, err)
		out, _ := cmd.Flags().GetString("out")
		assert.Equal(t, "/old/path.json", out)
	})

	t.Run("default value", func(t *testing.T) {
		t.Parallel()

		cmd := &cobra.Command{Use: "test"}
		Output(cmd, "default.json")

		f := cmd.Flags().Lookup("out")
		require.NotNil(t, f)
		assert.Equal(t, "default.json", f.DefValue)
	})
}

func TestFlagsAreLocal(t *testing.T) {
	t.Parallel()

	parent := &cobra.Command{Use: "parent", Run: func(cmd *cobra.Command, _ []string) {}}
	child := &cobra.Command{Use: "child", Run: func(cmd *cobra.Command, _ []string) {}}

	Environment(parent)
	parent.AddCommand(child)

	// Child should NOT have the environment flag
	f := child.Flags().Lookup("environment")
	assert.Nil(t, f, "local flags should not be inherited by subcommands")

	// But parent should have it
	pf := parent.Flags().Lookup("environment")
	assert.NotNil(t, pf)
}
