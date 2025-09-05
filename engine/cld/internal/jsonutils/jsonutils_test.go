package jsonutils

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_WriteFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		givePath string
		giveObj  any
		want     string
		wantErr  string
	}{
		{
			name:     "success",
			givePath: "valid.json",
			giveObj:  map[string]string{"key": "value"},
			want:     `{"key":"value"}`,
		},
		{
			name:     "failure: cannot marshal JSON",
			givePath: "invalid.json",
			giveObj:  make(chan int),
			wantErr:  "json: unsupported type: chan int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()

			err := WriteFile(filepath.Join(rootDir, tt.givePath), tt.giveObj)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)

				b, err := os.ReadFile(filepath.Join(rootDir, tt.givePath))
				require.NoError(t, err)

				assert.JSONEq(t, tt.want, string(b))
			}
		})
	}
}

func Test_LoadJSON(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"valid.json":   {Data: []byte(`{"key": "value"}`)},
		"invalid.json": {Data: []byte(`invalid`)},
	}

	tests := []struct {
		name    string
		give    string
		want    map[string]string
		wantErr string
	}{
		{
			name: "success",
			give: "valid.json",
			want: map[string]string{"key": "value"},
		},
		{
			name:    "failure: cannot read path",
			give:    "notfound.json",
			wantErr: "failed to read notfound.json",
		},
		{
			name:    "failure: cannot unmarshal JSON",
			give:    "invalid.json",
			wantErr: "failed to unmarshal JSON at path invalid.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := LoadFromFS[map[string]string](fsys, tt.give)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
