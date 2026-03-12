package run

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

type errorReporter struct {
	*operations.MemoryReporter
	getReportsErr error
}

func (e *errorReporter) GetReports() ([]operations.Report[any, any], error) {
	if e.getReportsErr != nil {
		return nil, e.getReportsErr
	}

	return e.MemoryReporter.GetReports()
}

type runStubChangeset struct{}

func (runStubChangeset) Apply(_ fdeployment.Environment, _ any) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}
func (runStubChangeset) VerifyPreconditions(_ fdeployment.Environment, _ any) error {
	return nil
}

var _ fdeployment.ChangeSetV2[any] = (*runStubChangeset)(nil)

func TestConfigureEnvironmentOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		regSetup  func() *cs.ChangesetsRegistry
		changeset string
		dryRun    bool
		wantErr   string
	}{
		{
			name: "success",
			regSetup: func() *cs.ChangesetsRegistry {
				reg := cs.NewChangesetsRegistry()
				reg.Add("0001_test", cs.Configure(&runStubChangeset{}).With(map[string]any{}))

				return reg
			},
			changeset: "0001_test",
			dryRun:    false,
		},
		{
			name: "unknown changeset",
			regSetup: func() *cs.ChangesetsRegistry {
				return cs.NewChangesetsRegistry()
			},
			changeset: "0001_test",
			dryRun:    false,
			wantErr:   "changeset '0001_test' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts, err := ConfigureEnvironmentOptions(tt.regSetup(), tt.changeset, tt.dryRun, logger.Test(t))

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, opts)
		})
	}
}

func TestGetChainOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		regSetup  func() *cs.ChangesetsRegistry
		changeset string
		want      []uint64
		wantErr   string
	}{
		{
			name: "from config returns nil",
			regSetup: func() *cs.ChangesetsRegistry {
				reg := cs.NewChangesetsRegistry()
				reg.Add("0001_test", cs.Configure(&runStubChangeset{}).With(map[string]any{}))

				return reg
			},
			changeset: "0001_test",
			want:      nil,
		},
		{
			name: "unknown changeset",
			regSetup: func() *cs.ChangesetsRegistry {
				return cs.NewChangesetsRegistry()
			},
			changeset: "0001_missing",
			wantErr:   "changeset '0001_missing' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetChainOverrides(tt.regSetup(), tt.changeset)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSaveReports_NoNewReports(t *testing.T) {
	t.Parallel()

	reporter := operations.NewMemoryReporter(operations.WithReports([]operations.Report[any, any]{{}}))
	artdir := domain.NewArtifactsDir(t.TempDir(), "test", "testnet")

	err := SaveReports(reporter, 1, logger.Test(t), artdir, "0001_test")
	require.NoError(t, err)
}

func TestSaveReports_WithNewReports(t *testing.T) {
	t.Parallel()

	reporter := operations.NewMemoryReporter(operations.WithReports([]operations.Report[any, any]{{}, {}, {}}))
	artdir := domain.NewArtifactsDir(t.TempDir(), "test", "testnet")

	err := SaveReports(reporter, 1, logger.Test(t), artdir, "0001_test")
	require.NoError(t, err)
}

func TestSaveReports_ReporterError(t *testing.T) {
	t.Parallel()

	base := operations.NewMemoryReporter()
	reporter := &errorReporter{MemoryReporter: base, getReportsErr: errors.New("reporter failed")}
	artdir := domain.NewArtifactsDir(t.TempDir(), "test", "testnet")

	err := SaveReports(reporter, 0, logger.Test(t), artdir, "0001_test")
	require.Error(t, err)
	require.Equal(t, "reporter failed", err.Error())
}
