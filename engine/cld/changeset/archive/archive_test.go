package archive

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

const migrationsTestSourceCode = `package testnet

const (
	migration0001 = "0001_mig"
	migration0003 = "0003_mig"
)

func (p *MigrationsRegistryProvider) Init() error {
	registry := p.Registry()

	registry.Add(migration0001, noopMigration{})
	registry.Add("0002_mig", noopMigration{})
	registry.Add(migration0003, noopMigration{})

	return nil
}`

const migrationsArchiveTestSourceCode = `package testnet

func (*MigrationsRegistryProvider) Archive() {}
`

func Test_Archivist_Archive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		giveMigContent        string
		giveMigArchiveContent string
		giveMigKeys           []string
		want                  archivalReport
		wantErr               string
	}{
		{
			name:                  "success removing a single migration",
			giveMigContent:        migrationsTestSourceCode,
			giveMigArchiveContent: migrationsArchiveTestSourceCode,
			giveMigKeys:           []string{"0001_mig"},
			want: archivalReport{
				"0001_mig": {
					constant:  pointer.To("migration0001"),
					isDeleted: true,
				},
			},
		},
		{
			name:                  "success removing multiple migrations with string",
			giveMigContent:        migrationsTestSourceCode,
			giveMigArchiveContent: migrationsArchiveTestSourceCode,
			giveMigKeys:           []string{"0001_mig", "0002_mig"},
			want: archivalReport{
				"0001_mig": {
					constant:  pointer.To("migration0001"),
					isDeleted: true,
				},
				"0002_mig": {
					constant:  nil,
					isDeleted: true,
				},
			},
		},
		{
			name:                  "success with unknown migration key",
			giveMigContent:        migrationsTestSourceCode,
			giveMigArchiveContent: migrationsArchiveTestSourceCode,
			giveMigKeys:           []string{"1000_mig"},
			want: archivalReport{
				"1000_mig": {
					constant:  nil,
					isDeleted: false,
				},
			},
		},
		{
			name:                  "error with no migrations file",
			giveMigArchiveContent: migrationsArchiveTestSourceCode,
			giveMigKeys:           []string{"1000_mig"},
			wantErr:               "no such file or directory",
		},
		{
			name:           "error with no migrations archive file",
			giveMigContent: migrationsTestSourceCode,
			giveMigKeys:    []string{"1000_mig"},
			wantErr:        "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			envdir := setupEnvDirectory(t)

			if tt.giveMigContent != "" {
				err := os.WriteFile(
					envdir.MigrationsFilePath(),
					[]byte(tt.giveMigContent),
					0600,
				)
				require.NoError(t, err)
			}

			if tt.giveMigArchiveContent != "" {
				err := os.WriteFile(
					envdir.MigrationsArchiveFilePath(),
					[]byte(tt.giveMigArchiveContent),
					0600,
				)
				require.NoError(t, err)
			}

			a := NewArchivist(envdir)
			a.mainBranchSHAGetter = fakeSHAGetter{}
			report, err := a.Archive(tt.giveMigKeys...)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, report)

				b, err := os.ReadFile(envdir.MigrationsFilePath())
				require.NoError(t, err)

				gotMigContext := string(b)

				for _, key := range tt.giveMigKeys {
					assert.NotContains(t, gotMigContext, key)
				}

				b, err = os.ReadFile(envdir.MigrationsArchiveFilePath())
				require.NoError(t, err)

				for _, key := range slices.Collect(maps.Keys(report)) {
					if report[key].isDeleted {
						assert.Contains(t, string(b), fmt.Sprintf("registry.Archive(\"%s\", \"%s\")",
							key, "fake-sha",
						))
					}
				}
			}
		})
	}
}

// fakeSHAGetter is a fake implementation of the SHAGetter interface
type fakeSHAGetter struct{}

// GetSHA returns a fake SHA
func (f fakeSHAGetter) Get() (string, error) {
	return "fake-sha", nil
}

// setupEnvDirectory creates the domain directory structure for archival testing.
func setupEnvDirectory(t *testing.T) domain.EnvDir {
	t.Helper()

	// Setup the root directory.
	rootDir := t.TempDir()

	var (
		domDir = filepath.Join(rootDir, "ccip")
		envDir = filepath.Join(domDir, "staging")
	)

	// Create the test domains.
	err := os.Mkdir(domDir, 0755)
	require.NoError(t, err)

	// Create the environments.
	err = os.Mkdir(envDir, 0755)
	require.NoError(t, err)

	return domain.NewDomain(rootDir, "ccip").EnvDir("staging")
}
