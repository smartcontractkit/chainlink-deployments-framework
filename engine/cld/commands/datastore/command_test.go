package datastore

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// mockCatalogStore implements fdatastore.CatalogStore for testing.
// Methods are not called in tests since we mock the CatalogMerger and CatalogSyncer functions.
type mockCatalogStore struct{}

func (m *mockCatalogStore) WithTransaction(_ context.Context, _ fdatastore.TransactionLogic) error {
	return nil
}

func (m *mockCatalogStore) Addresses() fdatastore.MutableRefStoreV2[fdatastore.AddressRefKey, fdatastore.AddressRef] {
	return nil
}

func (m *mockCatalogStore) ChainMetadata() fdatastore.MutableStoreV2[fdatastore.ChainMetadataKey, fdatastore.ChainMetadata] {
	return nil
}

func (m *mockCatalogStore) ContractMetadata() fdatastore.MutableStoreV2[fdatastore.ContractMetadataKey, fdatastore.ContractMetadata] {
	return nil
}

func (m *mockCatalogStore) EnvMetadata() fdatastore.MutableUnaryStoreV2[fdatastore.EnvMetadata] {
	return nil
}

// TestNewCommand_Structure verifies the command structure is correct.
func TestNewCommand_Structure(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

	require.NoError(t, err)
	require.NotNil(t, cmd)

	// Verify root command
	assert.Equal(t, "datastore", cmd.Use)
	assert.Equal(t, datastoreShort, cmd.Short)
	assert.NotEmpty(t, cmd.Long, "datastore command should have a Long description")

	// Verify NO persistent flags on parent (all flags are local to subcommands)
	envFlag := cmd.PersistentFlags().Lookup("environment")
	assert.Nil(t, envFlag, "environment flag should NOT be persistent")

	// Verify subcommands
	subs := cmd.Commands()
	require.Len(t, subs, 2)

	uses := make([]string, len(subs))
	for i, sc := range subs {
		uses[i] = sc.Use
	}
	assert.ElementsMatch(t, []string{"merge", "sync-to-catalog"}, uses)
}

// TestNewCommand_MergeFlags verifies the merge subcommand has correct local flags.
func TestNewCommand_MergeFlags(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

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

// TestNewCommand_SyncToCatalogFlags verifies the sync-to-catalog subcommand has correct local flags.
func TestNewCommand_SyncToCatalogFlags(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

	require.NoError(t, err)

	// Find the sync-to-catalog subcommand
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Use == "sync-to-catalog" {
			found = true

			// Environment flag - local to sync-to-catalog
			e := sub.Flags().Lookup("environment")
			require.NotNil(t, e, "environment flag should be on sync-to-catalog")
			assert.Equal(t, "e", e.Shorthand)

			break
		}
	}
	require.True(t, found, "sync-to-catalog subcommand not found")
}

// TestMerge_MissingEnvironmentFlagFails verifies required flag validation.
func TestMerge_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

	require.NoError(t, err)

	cmd.SetArgs([]string{"merge", "--name", "test"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "environment" not set`)
}

// TestMerge_MissingNameFlagFails verifies required flag validation.
func TestMerge_MissingNameFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

	require.NoError(t, err)

	cmd.SetArgs([]string{"merge", "-e", "staging"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "name" not set`)
}

// TestMerge_FileMode_Success verifies successful merge in file mode.
func TestMerge_FileMode_Success(t *testing.T) {
	t.Parallel()

	var fileMergerCalled bool
	var mergedName, mergedTimestamp string

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeFile,
				}, nil
			},
			FileMerger: func(_ domain.EnvDir, name, timestamp string) error {
				fileMergerCalled = true
				mergedName = name
				mergedTimestamp = timestamp

				return nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, fileMergerCalled, "file merger should be called")
	assert.Equal(t, "0001_deploy", mergedName)
	assert.Empty(t, mergedTimestamp)
	assert.Contains(t, out.String(), "📁 Using file-based datastore mode")
	assert.Contains(t, out.String(), "✅ Merged datastore to local files")
}

// TestMerge_FileMode_WithTimestamp verifies timestamp is passed through.
func TestMerge_FileMode_WithTimestamp(t *testing.T) {
	t.Parallel()

	var mergedTimestamp string

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeFile,
				}, nil
			},
			FileMerger: func(_ domain.EnvDir, _, timestamp string) error {
				mergedTimestamp = timestamp

				return nil
			},
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

// TestMerge_CatalogMode_Success verifies successful merge in catalog mode.
func TestMerge_CatalogMode_Success(t *testing.T) {
	t.Parallel()

	var catalogMergerCalled bool

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeCatalog,
					Env: &cfgenv.Config{
						Catalog: cfgenv.CatalogConfig{GRPC: "grpc.example.com:443"},
					},
				}, nil
			},
			CatalogLoader: func(_ context.Context, _ string, _ *config.Config, _ domain.Domain) (fdatastore.CatalogStore, error) {
				return &mockCatalogStore{}, nil
			},
			CatalogMerger: func(_ context.Context, _ domain.EnvDir, _, _ string, _ fdatastore.CatalogStore) error {
				catalogMergerCalled = true

				return nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, catalogMergerCalled, "catalog merger should be called")
	assert.Contains(t, out.String(), "📡 Using catalog datastore mode")
	assert.Contains(t, out.String(), "✅ Merged datastore to catalog")
}

// TestMerge_AllMode_Success verifies successful merge in all mode.
func TestMerge_AllMode_Success(t *testing.T) {
	t.Parallel()

	var fileMergerCalled, catalogMergerCalled bool

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeAll,
					Env: &cfgenv.Config{
						Catalog: cfgenv.CatalogConfig{GRPC: "grpc.example.com:443"},
					},
				}, nil
			},
			CatalogLoader: func(_ context.Context, _ string, _ *config.Config, _ domain.Domain) (fdatastore.CatalogStore, error) {
				return &mockCatalogStore{}, nil
			},
			CatalogMerger: func(_ context.Context, _ domain.EnvDir, _, _ string, _ fdatastore.CatalogStore) error {
				catalogMergerCalled = true

				return nil
			},
			FileMerger: func(_ domain.EnvDir, _, _ string) error {
				fileMergerCalled = true

				return nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, catalogMergerCalled, "catalog merger should be called")
	assert.True(t, fileMergerCalled, "file merger should be called")
	assert.Contains(t, out.String(), "📡 Using all datastore mode")
	assert.Contains(t, out.String(), "✅ Merged datastore to both catalog and local files")
}

// TestMerge_ConfigLoadError verifies error handling.
func TestMerge_ConfigLoadError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("config not found")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return nil, expectedError
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "failed to load config")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestMerge_CatalogLoadError verifies error handling.
func TestMerge_CatalogLoadError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("catalog connection failed")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeCatalog,
					Env: &cfgenv.Config{
						Catalog: cfgenv.CatalogConfig{GRPC: "grpc.example.com:443"},
					},
				}, nil
			},
			CatalogLoader: func(_ context.Context, _ string, _ *config.Config, _ domain.Domain) (fdatastore.CatalogStore, error) {
				return nil, expectedError
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "failed to load catalog")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestMerge_FileMergerError verifies error handling.
func TestMerge_FileMergerError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("merge failed")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeFile,
				}, nil
			},
			FileMerger: func(_ domain.EnvDir, _, _ string) error {
				return expectedError
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"merge", "-e", "staging", "-n", "0001_deploy"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "error during datastore merge to file")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestSyncToCatalog_MissingEnvironmentFlagFails verifies required flag validation.
func TestSyncToCatalog_MissingEnvironmentFlagFails(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

	require.NoError(t, err)

	cmd.SetArgs([]string{"sync-to-catalog"})
	execErr := cmd.Execute()

	require.ErrorContains(t, execErr, `required flag(s) "environment" not set`)
}

// TestSyncToCatalog_Success verifies successful sync.
func TestSyncToCatalog_Success(t *testing.T) {
	t.Parallel()

	var catalogSyncerCalled bool

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeCatalog,
					Env: &cfgenv.Config{
						Catalog: cfgenv.CatalogConfig{GRPC: "grpc.example.com:443"},
					},
				}, nil
			},
			CatalogLoader: func(_ context.Context, _ string, _ *config.Config, _ domain.Domain) (fdatastore.CatalogStore, error) {
				return &mockCatalogStore{}, nil
			},
			CatalogSyncer: func(_ context.Context, _ domain.EnvDir, _ fdatastore.CatalogStore) error {
				catalogSyncerCalled = true

				return nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"sync-to-catalog", "-e", "staging"})

	execErr := cmd.Execute()

	require.NoError(t, execErr)
	assert.True(t, catalogSyncerCalled, "catalog syncer should be called")
	assert.Contains(t, out.String(), "📡 Syncing local datastore to catalog")
	assert.Contains(t, out.String(), "✅ Successfully synced entire datastore to catalog")
}

// TestSyncToCatalog_CatalogNotConfigured verifies error when catalog is not configured.
func TestSyncToCatalog_CatalogNotConfigured(t *testing.T) {
	t.Parallel()

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeFile,
				}, nil
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"sync-to-catalog", "-e", "staging"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "catalog is not configured")
	assert.Contains(t, execErr.Error(), "staging")
}

// TestSyncToCatalog_SyncError verifies error handling.
func TestSyncToCatalog_SyncError(t *testing.T) {
	t.Parallel()

	expectedError := errors.New("sync failed")

	cmd, err := NewCommand(Config{
		Logger: logger.Nop(),
		Domain: domain.NewDomain("/tmp", "testdomain"),
		Deps: Deps{
			ConfigLoader: func(_ domain.Domain, _ string, _ logger.Logger) (*config.Config, error) {
				return &config.Config{
					DatastoreType: cfgdomain.DatastoreTypeCatalog,
					Env: &cfgenv.Config{
						Catalog: cfgenv.CatalogConfig{GRPC: "grpc.example.com:443"},
					},
				}, nil
			},
			CatalogLoader: func(_ context.Context, _ string, _ *config.Config, _ domain.Domain) (fdatastore.CatalogStore, error) {
				return &mockCatalogStore{}, nil
			},
			CatalogSyncer: func(_ context.Context, _ domain.EnvDir, _ fdatastore.CatalogStore) error {
				return expectedError
			},
		},
	})

	require.NoError(t, err)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"sync-to-catalog", "-e", "staging"})

	execErr := cmd.Execute()

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "error syncing datastore to catalog")
	assert.Contains(t, execErr.Error(), expectedError.Error())
}

// TestConfig_Validate verifies validation catches missing required fields.
func TestConfig_Validate(t *testing.T) {
	t.Parallel()

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
			Domain: domain.NewDomain("/tmp", "test"),
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
			Domain: domain.NewDomain("/tmp", "test"),
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
		Domain: domain.NewDomain("/tmp", "testdomain"),
	})

	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "Logger")
}
