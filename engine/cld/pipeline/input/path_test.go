package input

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

//nolint:paralleltest
func TestResolveDurablePipelineYamlPath(t *testing.T) {
	tests := []struct {
		name       string
		inputFile  string
		domKey     string
		envKey     string
		wantErr    string
		wantSuffix string
	}{
		{
			name:      "rejects full path",
			inputFile: "/some/path/file.yaml",
			domKey:    "test",
			envKey:    "testnet",
			wantErr:   "only filenames are supported, not full paths: /some/path/file.yaml",
		},
		{
			name:      "rejects relative path",
			inputFile: "subdir/file.yaml",
			domKey:    "test",
			envKey:    "testnet",
			wantErr:   "only filenames are supported, not full paths: subdir/file.yaml",
		},
		{
			name:       "success with filename",
			inputFile:  "pipeline.yaml",
			domKey:     "mydomain",
			envKey:     "testnet",
			wantSuffix: filepath.Join("domains", "mydomain", "testnet", "durable_pipelines", "inputs", "pipeline.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains", tt.domKey, tt.envKey, "durable_pipelines", "inputs"), 0o755))
			originalWd, _ := os.Getwd()
			require.NoError(t, os.Chdir(dir))
			t.Cleanup(func() { _ = os.Chdir(originalWd) })

			dom := domain.NewDomain(dir, tt.domKey)
			got, err := ResolveDurablePipelineYamlPath(tt.inputFile, dom, tt.envKey)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			expect := filepath.Join(dir, tt.wantSuffix)
			expectCanon, _ := filepath.EvalSymlinks(expect)
			gotCanon, _ := filepath.EvalSymlinks(got)
			require.Equal(t, expectCanon, gotCanon)
		})
	}
}
