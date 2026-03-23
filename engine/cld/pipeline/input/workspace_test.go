package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func TestFindWorkspaceRoot(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (expectedDir string, cleanup func())
		wantErr     string
		checkResult func(t *testing.T, got string, expectedDir string)
	}{
		{
			name: "success from nested subdir",
			setup: func(t *testing.T) (string, func()) {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains"), 0o755))
				subDir := filepath.Join(dir, "a", "b", "c")
				require.NoError(t, os.MkdirAll(subDir, 0o755))
				originalWd, _ := os.Getwd()
				require.NoError(t, os.Chdir(subDir))

				return dir, func() { _ = os.Chdir(originalWd) }
			},
			wantErr: "",
			checkResult: func(t *testing.T, got, expectedDir string) {
				t.Helper()
				expectedCanon, _ := filepath.EvalSymlinks(expectedDir)
				gotCanon, _ := filepath.EvalSymlinks(got)
				require.Equal(t, expectedCanon, gotCanon)
			},
		},
		{
			name: "not found when no domains dir",
			setup: func(t *testing.T) (string, func()) {
				t.Helper()
				dir := t.TempDir()
				require.NoError(t, os.Chdir(dir))

				return "", func() { _ = os.Chdir("/") }
			},
			wantErr: "could not find workspace root (directory with domains/)",
			checkResult: func(t *testing.T, _, _ string) {
				t.Helper()
			},
		},
		{
			name: "success from domains dir",
			setup: func(t *testing.T) (string, func()) {
				t.Helper()
				dir := t.TempDir()
				domainsDir := filepath.Join(dir, "domains")
				require.NoError(t, os.MkdirAll(domainsDir, 0o755))
				originalWd, _ := os.Getwd()
				require.NoError(t, os.Chdir(domainsDir))

				return dir, func() { _ = os.Chdir(originalWd) }
			},
			wantErr: "",
			checkResult: func(t *testing.T, got, expectedDir string) {
				t.Helper()
				expectedCanon, _ := filepath.EvalSymlinks(expectedDir)
				gotCanon, _ := filepath.EvalSymlinks(got)
				require.Equal(t, expectedCanon, gotCanon)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedDir, cleanup := tt.setup(t)
			t.Cleanup(cleanup)

			root, err := FindWorkspaceRoot()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			tt.checkResult(t, root, expectedDir)
		})
	}
}
