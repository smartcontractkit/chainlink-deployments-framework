package fileutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WriteFileGitKeep(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()

	err := WriteFileGitKeep(rootDir)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(rootDir, ".gitkeep"))
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(rootDir, "subdir"), 0000)
	require.NoError(t, err)

	err = WriteFileGitKeep(filepath.Join(rootDir, "subdir"))
	require.Error(t, err)
	require.ErrorContains(t, err, "permission denied")
}

func Test_MkdirAllGitKeep(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(t *testing.T, rootDir string)
		givePath   string
		wantErr    string
	}{
		{
			name:     "success",
			givePath: "foo/bar",
		},
		{
			name: "error when permissions are denied to create dir",
			beforeFunc: func(t *testing.T, rootDir string) {
				t.Helper()

				err := os.MkdirAll(filepath.Join(rootDir, "subdir"), 0000)
				require.NoError(t, err)
			},
			givePath: "subdir/foo/bar",
			wantErr:  "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, rootDir)
			}

			err := MkdirAllGitKeep(filepath.Join(rootDir, tt.givePath))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)

				_, err = os.Stat(filepath.Join(rootDir, tt.givePath, ".gitkeep"))
				require.NoError(t, err)
			}
		})
	}
}
