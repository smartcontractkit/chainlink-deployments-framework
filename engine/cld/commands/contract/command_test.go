package contract

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// mockContractInputsProvider is a no-op provider for testing.
type mockContractInputsProvider struct {
	getInputsErr error
}

func (m *mockContractInputsProvider) GetInputs(_ datastore.ContractType, _ *semver.Version) (evm.SolidityContractMetadata, error) {
	if m != nil && m.getInputsErr != nil {
		return evm.SolidityContractMetadata{}, m.getInputsErr
	}

	return evm.SolidityContractMetadata{}, nil
}

func newTestContractCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd, err := NewCommand(Config{
		Logger:                 logger.Nop(),
		Domain:                 domain.NewDomain(t.TempDir(), "testdomain"),
		ContractInputsProvider: &mockContractInputsProvider{},
	})
	require.NoError(t, err)

	return cmd
}

func TestNewCommand_Structure(t *testing.T) {
	t.Parallel()

	cmd := newTestContractCommand(t)

	require.Equal(t, "contract", cmd.Use)
	require.Equal(t, contractShort, cmd.Short)
	require.NotEmpty(t, cmd.Long)

	subs := cmd.Commands()
	require.Len(t, subs, 1)
	require.Equal(t, "verify-env", subs[0].Use)
}

func TestNewVerifyEnvCmdWithUse_CustomUse(t *testing.T) {
	t.Parallel()

	cmd := NewVerifyEnvCmdWithUse(Config{
		Logger:                 logger.Nop(),
		Domain:                 domain.NewDomain(t.TempDir(), "testdomain"),
		ContractInputsProvider: &mockContractInputsProvider{},
	}, "verify-evm")

	require.NotNil(t, cmd)
	require.Equal(t, "verify-evm", cmd.Use)
}

func TestVerifyEnv_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd := newTestContractCommand(t)
	cmd.SetArgs([]string{"verify-env"})

	err := cmd.Execute()

	require.Error(t, err)
	require.Equal(t, `required flag(s) "environment" not set`, err.Error())
}

func TestVerifyEnv_TxHashWithoutAddressFails(t *testing.T) {
	t.Parallel()

	cmd := newTestContractCommand(t)
	cmd.SetArgs([]string{"verify-env", "-e", "staging", "-t", "0x123"})

	err := cmd.Execute()

	require.Error(t, err)
	require.Equal(t, "--tx-hash requires --address to be specified", err.Error())
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("missing all required fields", func(t *testing.T) {
		t.Parallel()

		cfg := Config{}
		err := cfg.Validate()

		require.Error(t, err)
		require.Equal(t, "contract.Config: missing required fields: Logger, Domain, ContractInputsProvider", err.Error())
	})

	t.Run("missing ContractInputsProvider", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Logger: logger.Nop(),
			Domain: domain.NewDomain(tempDir, "test"),
		}
		err := cfg.Validate()

		require.EqualError(t, err, "contract.Config: missing required fields: ContractInputsProvider")
	})

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			Logger:                 logger.Nop(),
			Domain:                 domain.NewDomain(tempDir, "test"),
			ContractInputsProvider: &mockContractInputsProvider{},
		}
		err := cfg.Validate()

		require.NoError(t, err)
	})
}
