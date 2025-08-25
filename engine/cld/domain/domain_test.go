package domain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type tempProj struct {
	root        string
	domains     string
	nestedLevel string // a subdir inside the project to test upward search
}

func mkTempProject(t *testing.T) tempProj {
	t.Helper()

	root := t.TempDir()
	domains := filepath.Join(root, "domains")
	require.NoError(t, os.MkdirAll(domains, 0o755))

	// add at least one domain so GetDomain/MustGetDomain can succeed
	require.NoError(t, os.MkdirAll(filepath.Join(domains, "alpha"), 0o755))

	// create nested structure: <root>/x/y/z
	nested := filepath.Join(root, "x", "y", "z")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	return tempProj{
		root:        root,
		domains:     domains,
		nestedLevel: nested,
	}
}

// setGlobals points the package globals at our temp project.
func setGlobals(t *testing.T, p tempProj) {
	t.Helper()
	rootResolved := resolve(t, p.root)
	ProjectRoot = rootResolved
	DomainsRoot = resolve(t, filepath.Join(ProjectRoot, "domains"))
}

// resolve cleans and resolves symlinks for path-safe comparisons
func resolve(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	require.NoError(t, err)

	return r
}

//nolint:paralleltest // tests modify global ProjectRoot/DomainsRoot, cannot run in parallel
func Test_ProjectRoot(t *testing.T) {
	p := mkTempProject(t)
	setGlobals(t, p)
	require.True(t, isValidProjectRoot(ProjectRoot), "ProjectRoot should have a domains/ subdirectory")
}

//nolint:paralleltest // tests modify global ProjectRoot/DomainsRoot, cannot run in parallel
func Test_getProjectRoot_finds_project_root_via_upward_search_from_working_directory(t *testing.T) {
	p := mkTempProject(t)

	// run in nested dir so getProjectRoot's CWD strategy can find our root
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(p.nestedLevel))

	got := getProjectRoot()
	require.Equal(t,
		resolve(t, p.root),
		resolve(t, got),
		"getProjectRoot should walk upward to temp project root",
	)
}

func Test_searchUpwardForProjectRoot_finds_project_root_when_starting_from_nested_directory(t *testing.T) {
	t.Parallel()
	p := mkTempProject(t)
	got := searchUpwardForProjectRoot(p.nestedLevel)
	require.Equal(t, resolve(t, p.root), resolve(t, got))
}

func Test_searchUpwardForProjectRoot_returns_empty_string_when_no_project_root_found(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no domains/ inside
	got := searchUpwardForProjectRoot(dir)
	require.Empty(t, got)
}

func Test_isValidProjectRoot_returns_true_for_directory_with_domains_subdirectory(t *testing.T) {
	t.Parallel()
	p := mkTempProject(t)
	require.True(t, isValidProjectRoot(p.root))
}

func Test_isValidProjectRoot_returns_false_for_directory_without_domains_subdirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.False(t, isValidProjectRoot(dir))
}

func Test_isValidProjectRoot_returns_false_for_non_existent_directory(t *testing.T) {
	t.Parallel()
	require.False(t, isValidProjectRoot(filepath.Join(t.TempDir(), "does-not-exist")))
}

//nolint:paralleltest // tests modify global ProjectRoot/DomainsRoot, cannot run in parallel
func Test_GetDomain_success_in_domains_root(t *testing.T) {
	p := mkTempProject(t)
	setGlobals(t, p)

	d, err := GetDomain("alpha")
	require.NoError(t, err)
	require.Equal(t,
		resolve(t, filepath.Join(p.domains, "alpha")),
		resolve(t, d.DirPath()),
	)
	require.Equal(t, "alpha", d.Key())
}

//nolint:paralleltest // tests modify global ProjectRoot/DomainsRoot, cannot run in parallel
func Test_GetDomain_failure_with_invalid_domain_key(t *testing.T) {
	p := mkTempProject(t)
	setGlobals(t, p)

	_, err := GetDomain("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "domain not found: invalid")
}

//nolint:paralleltest // tests modify global ProjectRoot/DomainsRoot, cannot run in parallel
func Test_MustGetDomain(t *testing.T) {
	p := mkTempProject(t)
	setGlobals(t, p)

	require.NotPanics(t, func() {
		_ = MustGetDomain("alpha")
	})
}

func Test_Domain_DirPath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip", d.DirPath())
}

func Test_Domain_Key(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "ccip", d.Key())
}

func Test_Domain_String(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "ccip", d.String())
}

func Test_Domain_EnvDir(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	denv := d.EnvDir("staging")
	assert.Equal(t, "domains/ccip/staging", denv.DirPath())
}

func Test_Domain_AddressBookByEnv(t *testing.T) {
	t.Parallel()
	fixture := setupTestDomainsFS(t)
	d := NewDomain(fixture.rootDirPath, "ccip")
	got, err := d.AddressBookByEnv("staging")
	require.NoError(t, err)
	want := cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{})
	assert.Equal(t, want, got)
}

func Test_Domain_DataStoreByEnv(t *testing.T) {
	t.Parallel()
	fixture := setupTestDomainsFS(t)
	d := NewDomain(fixture.rootDirPath, "ccip")
	got, err := d.DataStoreByEnv("staging")
	require.NoError(t, err)
	want := datastore.NewMemoryDataStore().Seal()
	assert.Equal(t, want, got)
}

func Test_Domain_ArtifactsByEnv(t *testing.T) {
	t.Parallel()
	fixture := setupTestDomainsFS(t)
	d := NewDomain(fixture.rootDirPath, "ccip")
	got := d.ArtifactsDirByEnv("staging")
	assert.Equal(t, fixture.artifactsDir, got)
}

func Test_Domain_CmdDirPath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/cmd", d.CmdDirPath())
}

func Test_Domain_ConfigDirPath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config", d.ConfigDirPath())
}

func Test_Domain_ConfigLocalDirPath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config/local", d.ConfigLocalDirPath())
}

func Test_Domain_ConfigLocalFileName(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config/local/config.staging.yaml", d.ConfigLocalFilePath("staging"))
}

func Test_Domain_ConfigNetworksFilePath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config/networks", d.ConfigNetworksDirPath())
}

func Test_Domain_ConfigCIDirPath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config/ci", d.ConfigCIDirPath())
}

func Test_Domain_ConfigCICommonFilePath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config/ci/common.env", d.ConfigCICommonFilePath())
}

func Test_Domain_ConfigCIEnvFilePath(t *testing.T) {
	t.Parallel()
	d := NewDomain("domains", "ccip")
	assert.Equal(t, "domains/ccip/.config/ci/staging.env", d.ConfigCIEnvFilePath("staging"))
}

// todo: uncomment after moving migration registry over to cldf
// func Test_EnvDir_LatestExecutedMigration(t *testing.T) {
// 	t.Parallel()
//
// 	var (
// 		fixture = setupTestDomainsFS(t)
// 		envdir  = fixture.envDir
// 		reg     = NewMigrationsRegistry()
// 	)
//
// 	reg.Add("0001_initial", nil)
// 	reg.Add("0002_second", nil)
//
// 	_, err := envdir.LatestExecutedMigration(reg)
// 	require.EqualError(t, err, "no migrations have been executed")
//
// 	// Simulate a migration being executed by creating a migration artifacts group
// 	err = envdir.ArtifactsDir().CreateMigrationDir("0001_initial")
// 	require.NoError(t, err)
//
// 	got, err := envdir.LatestExecutedMigration(reg)
// 	require.NoError(t, err)
// 	require.Equal(t, "0001_initial", got)
//
// 	// Create another migration artifacts group
// 	err = envdir.ArtifactsDir().CreateMigrationDir("0002_second")
// 	require.NoError(t, err)
//
// 	got, err = envdir.LatestExecutedMigration(reg)
// 	require.NoError(t, err)
// 	require.Equal(t, "0002_second", got)
// }
