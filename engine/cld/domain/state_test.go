package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMarshaler struct {
	Name string
}

func (m *testMarshaler) MarshalJSON() ([]byte, error) {
	type Alias testMarshaler

	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

type failedMarshaler struct{}

func (m *failedMarshaler) MarshalJSON() ([]byte, error) {
	return nil, assert.AnError
}

func Test_SaveViewState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		beforeFunc   func(t *testing.T, rootDir string)
		giveFilePath string
		giveState    json.Marshaler
		want         string
		wantErr      string
	}{
		{
			name:         "success",
			giveFilePath: "state.json",
			giveState: &testMarshaler{
				Name: "test",
			},
			want: `{"Name":"test"}`,
		},
		{
			name:         "failure with non existent directory",
			giveFilePath: "non/existent/directory/state.json",
			giveState:    &testMarshaler{},
			wantErr:      "failed to stat",
		},
		{
			name:         "failure with marshal error",
			giveFilePath: "state.json",
			giveState:    &failedMarshaler{},
			wantErr:      "unable to marshal state",
		},
		{
			name: "failure with write error due to permissions",
			beforeFunc: func(t *testing.T, rootDir string) {
				t.Helper()

				err := os.MkdirAll(filepath.Join(rootDir, "dir"), 0400)
				require.NoError(t, err)
			},
			giveFilePath: "dir/state.json",
			giveState: &testMarshaler{
				Name: "test",
			},
			wantErr: "failed to write state file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, rootDir)
			}

			err := SaveViewState(filepath.Join(rootDir, tt.giveFilePath), tt.giveState)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				b, err := os.ReadFile(filepath.Join(rootDir, tt.giveFilePath))
				require.NoError(t, err)

				assert.JSONEq(t, tt.want, string(b))
			}
		})
	}
}
