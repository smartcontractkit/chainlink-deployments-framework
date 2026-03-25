package artifacts

import (
	"os"
	"path/filepath"
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
