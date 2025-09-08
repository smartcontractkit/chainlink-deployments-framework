package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func Test_Base_AddCommand(t *testing.T) {
	t.Parallel()

	base := Base{
		rootCmd: &cobra.Command{},
	}

	give := &cobra.Command{}

	base.AddCommand(give)

	assert.Equal(t, []*cobra.Command{give}, base.rootCmd.Commands())
}

func Test_Base_Run(t *testing.T) {
	t.Parallel()

	var val string

	base := Base{
		rootCmd: &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				val = "ran"

				return nil
			},
		},
	}

	err := base.Run()
	require.NoError(t, err)
	assert.Equal(t, "ran", val)
}

func Test_NewLogger(t *testing.T) {
	t.Parallel()

	log, err := NewLogger(zapcore.DebugLevel)
	require.NoError(t, err)
	assert.NotNil(t, log)
}
