package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func Test_ScaffoldDomain(t *testing.T) {
	t.Parallel()

	var (
		rootDir = t.TempDir()
		domKey  = "brontosaurus"
	)

	dom := domain.NewDomain(rootDir, domKey)
	err := ScaffoldDomain(dom)
	require.NoError(t, err)

	info, err := os.Stat(dom.DirPath())
	require.NoError(t, err)

	assert.True(t, info.IsDir())

	info, err = os.Stat(dom.InternalDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(dom.LibDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = os.Stat(filepath.Join(dom.CmdDirPath(), "main.go"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dom.CmdDirPath(), "internal", "cli", "app.go"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dom.DirPath(), ".config", "networks", "mainnet.yaml"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dom.DirPath(), ".config", "networks", "testnet.yaml"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dom.DirPath(), ".config", "local"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dom.DirPath(), ".config", "ci", "common.env"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dom.DirPath(), ".config", "domain.yaml"))
	require.NoError(t, err)

	// Check that go.mod file is created
	goModPath := filepath.Join(dom.DirPath(), "go.mod")
	_, err = os.Stat(goModPath)
	require.NoError(t, err)

	// Verify go.mod content has correct module name
	goModContent, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	repoName := filepath.Base(filepath.Dir(rootDir))
	expectedModuleLine := fmt.Sprintf("module github.com/smartcontractkit/%s/domains/%s", repoName, domKey)
	assert.Contains(t, string(goModContent), expectedModuleLine)

	err = ScaffoldDomain(dom)
	require.Error(t, err)
	assert.ErrorContains(t, err, os.ErrExist.Error())
}

func Test_ScaffoldEnvDir(t *testing.T) {
	t.Parallel()

	var (
		rootDir = t.TempDir()
		domKey  = "ccip"
		envKey  = "staging"
	)

	dom := domain.NewDomain(rootDir, domKey)
	err := ScaffoldDomain(dom)
	require.NoError(t, err)

	envdir := domain.NewEnvDir(rootDir, domKey, envKey)
	err = ScaffoldEnvDir(envdir)
	require.NoError(t, err)

	info, err := os.Stat(envdir.DirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = os.Stat(envdir.AddressBookFilePath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.AddressRefsFilePath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.ChainMetadataFilePath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.ContractMetadataFilePath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.EnvMetadataFilePath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.NodesFilePath())
	require.NoError(t, err)

	_, err = os.Stat(envdir.ViewStateFilePath())
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(envdir.DirPath(), "pipelines.go"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(envdir.DirPath(), "pipelines_test.go"))
	require.NoError(t, err)

	info, err = os.Stat(envdir.ArtifactsDir().ArtifactsDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(envdir.ArtifactsDir().ProposalsDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(envdir.ArtifactsDir().ArchivedProposalsDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(envdir.ArtifactsDir().DecodedProposalsDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(envdir.ArtifactsDir().OperationsReportsDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(envdir.DataStoreDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check if durable_pipelines directory is created
	info, err = os.Stat(envdir.DurablePipelinesDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check if durable_pipelines/inputs directory is created
	info, err = os.Stat(envdir.DurablePipelinesInputsDirPath())
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	err = ScaffoldEnvDir(envdir)
	assert.ErrorContains(t, err, os.ErrExist.Error())
}
func Test_getRepositoryName(t *testing.T) {
	t.Parallel()

	rootDir := domain.NewDomain("my/root/dir/repo_name/domains", "dummy").RootPath()
	result := getRepositoryName(rootDir)
	assert.Equal(t, "repo_name", result)
}

func Test_ScaffoldEnvDir_WithHyphens(t *testing.T) {
	t.Parallel()

	var (
		rootDir = t.TempDir()
		domKey  = "ccip"
		envKey  = "my-test-env"
	)

	// First scaffold the domain
	dom := domain.NewDomain(rootDir, domKey)
	err := ScaffoldDomain(dom)
	require.NoError(t, err)

	// Then scaffold the environment
	envdir := domain.NewEnvDir(rootDir, domKey, envKey)
	err = ScaffoldEnvDir(envdir)
	require.NoError(t, err)

	// Check that pipelines.go uses sanitized package name
	pipelinesPath := filepath.Join(envdir.DirPath(), "pipelines.go")
	pipelinesContent, err := os.ReadFile(pipelinesPath)
	require.NoError(t, err)

	// Should contain sanitized package name (underscores instead of hyphens)
	assert.Contains(t, string(pipelinesContent), "my_test_env")
	assert.NotContains(t, string(pipelinesContent), "my-test-env")
}

func Test_sanitizePackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no hyphens",
			input:    "mypackage",
			expected: "mypackage",
		},
		{
			name:     "single hyphen",
			input:    "my-package",
			expected: "my_package",
		},
		{
			name:     "multiple hyphens",
			input:    "my-complex-package-name",
			expected: "my_complex_package_name",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only hyphens",
			input:    "---",
			expected: "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := sanitizePackageName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
