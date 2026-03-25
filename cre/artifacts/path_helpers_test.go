package artifacts

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_resolveLocalArtifactPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	good := filepath.Join(dir, "f.json")
	require.NoError(t, os.WriteFile(good, []byte("{}"), 0o600))
	sub := filepath.Join(dir, "d")
	require.NoError(t, os.Mkdir(sub, 0o700))

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr string
	}{
		{name: "bare_file", path: good, want: good},
		{name: "padded_file", path: "  " + good + "\t", want: good},
		{name: "missing", path: filepath.Join(dir, "nope"), wantErr: "does not exist"},
		{name: "directory", path: sub, wantErr: "directory"},
		{name: "empty_string", path: "", wantErr: "empty"},
		{name: "whitespace_only", path: "   \t  ", wantErr: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveLocalArtifactPath(tt.path)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Empty(t, got)

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_writeToFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		reader      io.Reader
		wantContent []byte
		wantErr     string
		wantCleanup bool
	}{
		{
			name:        "small_file",
			reader:      strings.NewReader("hello"),
			wantContent: []byte("hello"),
		},
		{
			name:        "exceeds_max_size",
			reader:      io.LimitReader(neverEndingReader{}, maxDownloadSize+1),
			wantErr:     "exceeds maximum size",
			wantCleanup: true,
		},
		{
			name:        "read_error",
			reader:      errReader{err: io.ErrUnexpectedEOF},
			wantErr:     "write file",
			wantCleanup: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "out.bin")
			err := writeToFile(path, tt.reader)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				if tt.wantCleanup {
					entries, dirErr := os.ReadDir(dir)
					require.NoError(t, dirErr)
					require.Empty(t, entries, "partial file should be removed on failure")
				}

				return
			}
			require.NoError(t, err)
			got, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, tt.wantContent, got)
		})
	}
}

type neverEndingReader struct{}

func (neverEndingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0x42
	}

	return len(p), nil
}
