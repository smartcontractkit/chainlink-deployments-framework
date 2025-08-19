package domain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/lib/jsonutils"
)

type testJobSpec struct {
	A string `toml:"a"`
	B string `toml:"b"`
}

func (t testJobSpec) MustMarshal() string {
	b, err := toml.Marshal(t)
	if err != nil {
		panic(err)
	}

	return string(b)
}

func Test_LoadJobSpecs(t *testing.T) {
	t.Parallel()

	jss := map[string][]string{
		"node1": {testJobSpec{A: "a", B: "b"}.MustMarshal()},
	}

	tests := []struct {
		name         string
		beforeFunc   func(t *testing.T, rootDir string)
		giveFilePath string
		want         map[string][]string
		wantErr      string
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, rootDir string) {
				t.Helper()

				err := jsonutils.WriteFile(filepath.Join(rootDir, "jss.json"), jss)
				require.NoError(t, err)
			},
			giveFilePath: "jss.json",
			want:         jss,
		},
		{
			name:         "failure with non existent file",
			giveFilePath: "non/existent/directory/jss.json",
			wantErr:      "no such file or directory",
		},
		{
			name: "failure with marshal error",
			beforeFunc: func(t *testing.T, rootDir string) {
				t.Helper()

				_, err := os.Create(filepath.Join(rootDir, "jss.json"))
				require.NoError(t, err)
			},
			giveFilePath: "jss.json",
			wantErr:      "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, rootDir)
			}

			got, err := LoadJobSpecs(filepath.Join(rootDir, tt.giveFilePath))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_LoadJobs(t *testing.T) {
	t.Parallel()

	jobs := []cldf.ProposedJob{
		{
			JobID: "job_123",
			Node:  "node1",
			Spec:  testJobSpec{A: "a", B: "b"}.MustMarshal(),
		},
	}

	tests := []struct {
		name         string
		beforeFunc   func(t *testing.T, rootDir string)
		giveFilePath string
		want         []cldf.ProposedJob
		wantErr      string
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, rootDir string) {
				t.Helper()

				err := jsonutils.WriteFile(filepath.Join(rootDir, "jobs.json"), jobs)
				require.NoError(t, err)
			},
			giveFilePath: "jobs.json",
			want:         jobs,
		},
		{
			name:         "failure with non existent file",
			giveFilePath: "non/existent/directory/jobs.json",
			wantErr:      "no such file or directory",
		},
		{
			name: "failure with marshal error",
			beforeFunc: func(t *testing.T, rootDir string) {
				t.Helper()

				_, err := os.Create(filepath.Join(rootDir, "jobs.json"))
				require.NoError(t, err)
			},
			giveFilePath: "jobs.json",
			wantErr:      "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, rootDir)
			}

			got, err := LoadJobs(filepath.Join(rootDir, tt.giveFilePath))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
