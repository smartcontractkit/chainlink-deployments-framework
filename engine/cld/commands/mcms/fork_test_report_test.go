package mcms

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newForkTestReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want forkTestReport
	}{
		{
			name: "success",
			err:  nil,
			want: forkTestReport{ChainSelector: 1, Status: forkTestStatusSuccess},
		},
		{
			name: "skipped",
			err:  ForkTestSkippedError{message: "unsupported chain family: solana"},
			want: forkTestReport{ChainSelector: 1, Status: forkTestStatusSkipped, Details: "unsupported chain family: solana"},
		},
		{
			name: "wrapped skipped",
			err:  fmt.Errorf("wrapped: %w", ForkTestSkippedError{message: "zksync"}),
			want: forkTestReport{ChainSelector: 1, Status: forkTestStatusSkipped, Details: "zksync"},
		},
		{
			name: "failure",
			err:  ForkTestFailedError{stage: "MCM.setRoot", cause: errors.New("execution reverted")},
			want: forkTestReport{ChainSelector: 1, Status: forkTestStatusFailure, Step: "MCM.setRoot", Details: "execution reverted"},
		},
		{
			name: "wrapped failure",
			err:  fmt.Errorf("wrapped: %w", ForkTestFailedError{stage: "Timelock.execute", cause: errors.New("boom")}),
			want: forkTestReport{ChainSelector: 1, Status: forkTestStatusFailure, Step: "Timelock.execute", Details: "boom"},
		},
		{
			name: "any other error",
			err:  errors.New("failed to load environment"),
			want: forkTestReport{ChainSelector: 1, Status: forkTestStatusError, Details: "failed to load environment"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newForkTestReport(1, tt.err)

			require.Equal(t, tt.want, got)
		})
	}
}

func Test_forkTestReport_save(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		report   forkTestReport
		path     func(t *testing.T) string
		wantJSON string
		wantErr  string
	}{
		{
			name:     "success: save successful fork test",
			report:   forkTestReport{ChainSelector: 1, Status: forkTestStatusSuccess},
			path:     func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "report.json") },
			wantJSON: `{ "chainSelector": 1, "status": "success" } `,
		},
		{
			name:     "success: save failed fork test with step and details",
			report:   forkTestReport{ChainSelector: 2, Status: forkTestStatusFailure, Step: "MCM.execute", Details: "reverted"},
			path:     func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "report.json") },
			wantJSON: `{ "chainSelector": 2, "status": "failure", "step": "MCM.execute", "details": "reverted" } `,
		},
		{
			name:    "failure: path is not writable",
			report:  forkTestReport{ChainSelector: 1, Status: forkTestStatusSuccess},
			path:    func(t *testing.T) string { t.Helper(); return filepath.Join(t.TempDir(), "missing-dir", "report.json") },
			wantErr: "failed to create report file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.path(t)

			err := tt.report.save(path)

			if tt.wantErr == "" {
				require.NoError(t, err)
				got, rerr := os.ReadFile(path)
				require.NoError(t, rerr)
				require.JSONEq(t, tt.wantJSON, string(got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
