package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/segmentio/ksuid"

	"github.com/smartcontractkit/mcms"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/fileutils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/jsonutils"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

const (
	// Defines the file extensions for the artifacts
	TOMLExt = "toml"
	JSONExt = "json"
	MDExt   = "md"
	TxtExt  = "txt"

	// Defines the artifact types. These are also used as suffixes for the artifact file names.
	ArtifactAddress                      = "addresses"
	ArtifactDataStore                    = "datastore"
	ArtifactJobSpec                      = "jobspecs"
	ArtifactJobs                         = "jobs"
	ArtifactsDurablePipelineDirName      = "durable_pipelines"
	ArtifactMCMSProposal                 = "mcms_proposal"
	ArtifactMCMSProposalDecoded          = "mcms_proposal_decoded"
	ArtifactsMCMSTimelockProposal        = "mcms_timelock_proposal"
	ArtifactsMCMSTimelockProposalDecoded = "mcms_timelock_proposal_decoded"
)

var (
	// ErrArtifactNotFound is returned when an artifact is not in the filesystem.
	ErrArtifactNotFound = errors.New("artifact not found")
)

// ArtifactsDir represents a directory containing all migration artifacts grouped by the migration
// key. It provides methods to interact with the artifacts in the directory.
type ArtifactsDir struct {
	rootPath           string // rootPath is absolute path of the domains filesystem
	domainKey          string // The key of the domain that the environment belongs to. e.g. "ccip", "keystone"
	envKey             string // The name of the environment. e.g. "mainnet", "testnet", "staging"
	durablePipelineDir string // The directory containing the durable pipeline artifacts
	timestamp          string // The timestamp when the migration started
}

// NewArtifactsDir creates a new Artifacts.
func NewArtifactsDir(rootPath, domainKey, envKey string) *ArtifactsDir {
	return &ArtifactsDir{
		rootPath:  rootPath,
		domainKey: domainKey,
		envKey:    envKey,
	}
}

// ArtifactsDirPath returns the path to the directory containing the artifacts but not proposals.
func (a *ArtifactsDir) ArtifactsDirPath() string {
	return filepath.Join(a.rootPath, a.domainKey, a.envKey, ArtifactsDirName, a.durablePipelineDir)
}

// ProposalsDirPath returns the path to the directory containing the proposals.
func (a *ArtifactsDir) ProposalsDirPath() string {
	return filepath.Join(a.rootPath, a.domainKey, a.envKey, ProposalsDirName)
}

// DecodedProposalsDirPath returns the path to the directory containing the decoded proposals.
func (a *ArtifactsDir) DecodedProposalsDirPath() string {
	return filepath.Join(a.rootPath, a.domainKey, a.envKey, DecodedProposalsDirName)
}

// ArchivedProposalsDirPath returns the path to the directory containing archived proposals.
func (a *ArtifactsDir) ArchivedProposalsDirPath() string {
	return filepath.Join(a.rootPath, a.domainKey, a.envKey, ArchivedProposalsDirName)
}

// OperationsReportsDirPath returns the path to the directory containing the operations reports.
func (a *ArtifactsDir) OperationsReportsDirPath() string {
	return filepath.Join(a.rootPath, a.domainKey, a.envKey, OperationsReportsDirName, a.durablePipelineDir)
}

// SetDurablePipelines sets the directory containing the durable pipeline artifacts and the timestamp for the durable pipelines.
func (a *ArtifactsDir) SetDurablePipelines(timestamp string) error {
	a.durablePipelineDir = filepath.Join(ArtifactsDurablePipelineDirName)

	return a.setDurablePipelinesTimestamp(timestamp)
}

func (a *ArtifactsDir) setDurablePipelinesTimestamp(timestamp string) error {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	if ts <= 0 {
		return errors.New("timestamp must be greater than 0")
	}
	if len(timestamp) != 19 {
		return errors.New("timestamp must be in nanoseconds unix time format (19 digits)")
	}

	a.timestamp = timestamp

	return nil
}

// CreateProposalsDir creates the proposals directory within the artifacts directory if it does not exist.
// It also creates a .gitkeep file within the proposals directory to ensure the directory is tracked by git.
func (a *ArtifactsDir) CreateProposalsDir() error {
	migDirPath := a.getProposalDir()

	if err := os.MkdirAll(migDirPath, 0755); err != nil {
		return err
	}

	_, err := os.Create(filepath.Join(migDirPath, ".gitkeep"))

	return err
}

// CreateOperationsReportsDir creates the operations reports directory if it does not exist.
// It also creates a .gitkeep file within the operations reports directory to ensure the directory is tracked by git.
func (a *ArtifactsDir) CreateOperationsReportsDir() error {
	if err := fileutils.MkdirAllGitKeep(a.OperationsReportsDirPath()); err != nil {
		return err
	}

	return nil
}

// CreateDecodedProposalsDir creates the decoded_proposals directory within the artifacts directory if it does not exist.
// It also creates a .gitkeep file within the proposals directory to ensure the directory is tracked by git.
func (a *ArtifactsDir) CreateDecodedProposalsDir() error {
	migDirPath := a.getDecodedProposalDir()

	if err := os.MkdirAll(migDirPath, 0755); err != nil {
		return err
	}

	_, err := os.Create(filepath.Join(migDirPath, ".gitkeep"))

	return err
}

// CreateArchivedProposalsDir creates the proposals directory within the artifacts directory if it does not exist.
// It also creates a .gitkeep file within the proposals directory to ensure the directory is tracked by git.
func (a *ArtifactsDir) CreateArchivedProposalsDir() error {
	migDirPath := a.getArchivedProposalDir()

	if err := os.MkdirAll(migDirPath, 0755); err != nil {
		return err
	}

	_, err := os.Create(filepath.Join(migDirPath, ".gitkeep"))

	return err
}

// DomainKey returns the domain key that the artifacts belong to.
func (a *ArtifactsDir) DomainKey() string {
	return a.domainKey
}

// EnvKey returns the environment key that the artifacts belong to.
func (a *ArtifactsDir) EnvKey() string {
	return a.envKey
}

// ChangesetDirPath returns the path to the directory containing the artifacts for the specified
// changeset key.
func (a *ArtifactsDir) ChangesetDirPath(csKey string) string {
	return filepath.Join(a.ArtifactsDirPath(), csKey, a.timestamp)
}

// CreateChangesetDir creates a new directory within the artifacts directory with the specified
// changeset key. If the directory already exists, it will return nil.
func (a *ArtifactsDir) CreateChangesetDir(csKey string) error {
	csDirPath := a.ChangesetDirPath(csKey)
	if err := os.MkdirAll(csDirPath, 0755); err != nil {
		return err
	}

	_, err := os.Create(filepath.Join(csDirPath, ".gitkeep"))

	return err
}

// RemoveChangesetDir removes the directory containing the artifacts for the specified changeset
// key.
func (a *ArtifactsDir) RemoveChangesetDir(csKey string) error {
	return os.RemoveAll(a.ChangesetDirPath(csKey))
}

// ChangesetDirExists checks if the changeset directory containing the artifacts for the specified
// changeset key exists.
func (a *ArtifactsDir) ChangesetDirExists(csKey string) (bool, error) {
	info, err := os.Stat(a.ChangesetDirPath(csKey))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, errors.New("expected directory, got file")
	}

	return true, nil
}

// OperationsReportsDirExists checks if the operations_reports directory exists.
func (a *ArtifactsDir) OperationsReportsDirExists() (bool, error) {
	info, err := os.Stat(a.OperationsReportsDirPath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, errors.New("expected directory, got file")
	}

	return true, nil
}

// ChangesetOperationsReportsFileExists checks if the operations reports file exists for the specified changeset key.
func (a *ArtifactsDir) ChangesetOperationsReportsFileExists(csKey string) (bool, error) {
	info, err := os.Stat(a.getOperationsReportsFilePath(csKey))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if info.IsDir() {
		return false, errors.New("expected file, got directory")
	}

	return true, nil
}

// ProposalsDirExists checks if the proposals directory exists
func (a *ArtifactsDir) ProposalsDirExists() (bool, error) {
	info, err := os.Stat(a.getProposalDir())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, errors.New("expected directory, got file")
	}

	return true, nil
}

// ArchiveProposalsDirExists checks if the proposals directory exists
func (a *ArtifactsDir) ArchiveProposalsDirExists() (bool, error) {
	info, err := os.Stat(a.getArchivedProposalDir())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, errors.New("expected directory, got file")
	}

	return true, nil
}

// SaveChangesetOutput writes the ChangesetOutput as artifacts to the specified changeset directory.
func (a *ArtifactsDir) SaveChangesetOutput(csKey string, output fdeployment.ChangesetOutput) error {
	id := ksuid.New()

	// Create the changeset directory if it doesn't exist
	if err := a.CreateChangesetDir(csKey); err != nil {
		return err
	}

	// Create the proposals directory if it doesn't exist
	if err := a.CreateProposalsDir(); err != nil {
		return err
	}

	// Create the decoded proposals directory if it doesn't exist
	if err := a.CreateDecodedProposalsDir(); err != nil {
		return err
	}

	// Create the proposals directory if it doesn't exist
	if err := a.CreateArchivedProposalsDir(); err != nil {
		return err
	}

	// Write job specs artifact
	//nolint:staticcheck
	if len(output.JobSpecs) > 0 {
		//nolint:staticcheck
		if err := a.saveArtifact(id, csKey, ArtifactJobSpec, output.JobSpecs); err != nil {
			return err
		}
	}

	if len(output.Jobs) > 0 {
		if err := a.saveArtifact(id, csKey, ArtifactJobs, output.Jobs); err != nil {
			return err
		}
	}

	// Write MCMS proposal to proposals directory
	if len(output.MCMSProposals) > 0 {
		for i, p := range output.MCMSProposals {
			if err := a.saveProposalArtifact(csKey, ArtifactMCMSProposal, i, p); err != nil {
				return err
			}
		}
	}

	// Write Timelock proposals to proposals directory
	if len(output.MCMSTimelockProposals) > 0 {
		for i, p := range output.MCMSTimelockProposals {
			// this allows us to upgrade the changesets gradually to the new MCMS lib while maintaining the existing behaviour here
			// until product is ready to switch to the new MCMS proposal format
			hasDecoded := len(output.DescribedTimelockProposals) > i && output.DescribedTimelockProposals[i] != ""

			if err := a.saveProposalArtifact(csKey, ArtifactsMCMSTimelockProposal, i, p); err != nil {
				return err
			}
			if hasDecoded {
				if err := a.saveDecodedProposalArtifact(csKey, ArtifactsMCMSTimelockProposal, i, output.DescribedTimelockProposals[i]); err != nil {
					return err
				}
			}
		}
	}

	// Write address book artifact
	//nolint:staticcheck
	if output.AddressBook != nil {
		addressBook, err := output.AddressBook.Addresses()
		if err != nil {
			return err
		}

		if len(addressBook) > 0 {
			// Sort the address book first by chain IDs numerically, then by address names alphabetically
			sortedBytes, err := marshalIndentAndSort(addressBook)
			if err != nil {
				return err
			}
			if err := a.saveArtifact(id, csKey, ArtifactAddress, json.RawMessage(sortedBytes)); err != nil {
				return err
			}
		}
	}

	// Write data store artifact
	if output.DataStore != nil {
		if err := a.saveArtifact(id, csKey, ArtifactDataStore, output.DataStore); err != nil {
			return err
		}
	}

	return nil
}

// LoadChangesetOutput reads the artifacts from the specified changeset directory and returns the ChangesetOutput.
func (a *ArtifactsDir) LoadChangesetOutput(csKey string) (fdeployment.ChangesetOutput, error) {
	changesetDir := a.ChangesetDirPath(csKey)
	proposalsDir := a.getProposalDir()

	artifactEntries, err := os.ReadDir(changesetDir)
	if err != nil {
		return fdeployment.ChangesetOutput{}, err
	}

	proposalEntries, err := os.ReadDir(proposalsDir)
	if err != nil {
		return fdeployment.ChangesetOutput{}, err
	}

	jobSpecsRgx := regexp.MustCompile(fmt.Sprintf(`^[a-zA-Z0-9_-]+_%s\.json$`, ArtifactJobSpec))
	jobsRgx := regexp.MustCompile(fmt.Sprintf(`^[a-zA-Z0-9_-]+_%s\.json$`, ArtifactJobs))
	addressesRgx := regexp.MustCompile(fmt.Sprintf(`^[a-zA-Z0-9_-]+_%s\.json$`, ArtifactAddress))
	datastoreRgx := regexp.MustCompile(fmt.Sprintf(`^[a-zA-Z0-9_-]+_%s\.json$`, ArtifactDataStore))
	mcmsTimelockProposalRgx := regexp.MustCompile(
		fmt.Sprintf(`^[a-zA-Z0-9_-]+_%s_\d+\.json$`, ArtifactsMCMSTimelockProposal),
	)
	mcmsProposalRgx := regexp.MustCompile(
		fmt.Sprintf(`^[a-zA-Z0-9_-]+_%s_\d+\.json$`, ArtifactMCMSProposal),
	)

	var output fdeployment.ChangesetOutput
	for _, entry := range artifactEntries {
		// Shortcircuit to ignore directories and the .gitkeep file
		if entry.IsDir() || entry.Name() == ".gitkeep" {
			continue
		}

		// Determine the path of the artifact file
		entryPath := filepath.Join(changesetDir, entry.Name())

		switch name := entry.Name(); {
		case jobSpecsRgx.MatchString(name):
			jss, err1 := LoadJobSpecs(entryPath)
			if err1 != nil {
				return fdeployment.ChangesetOutput{}, err1
			}
			//nolint:staticcheck
			output.JobSpecs = jss
		case jobsRgx.MatchString(name):
			jobs, err1 := LoadJobs(entryPath)
			if err1 != nil {
				return fdeployment.ChangesetOutput{}, err1
			}
			output.Jobs = jobs
		case addressesRgx.MatchString(name):
			ab, err1 := a.loadAddressBook(entryPath)
			if err1 != nil {
				return fdeployment.ChangesetOutput{}, err1
			}
			//nolint:staticcheck
			output.AddressBook = ab
		case datastoreRgx.MatchString(name):
			ds, err1 := a.loadMutableDataStore(entryPath)
			if err1 != nil {
				return fdeployment.ChangesetOutput{}, err1
			}
			output.DataStore = ds
		}
	}

	// Process proposal files from proposals directory
	for _, entry := range proposalEntries {
		if entry.IsDir() || entry.Name() == ".gitkeep" {
			continue
		}

		entryPath := filepath.Join(proposalsDir, entry.Name())

		switch name := entry.Name(); {
		case mcmsTimelockProposalRgx.MatchString(name):
			p, err1 := a.loadMCMSTimelockProposal(entryPath)
			if err1 != nil {
				return fdeployment.ChangesetOutput{}, err1
			}
			output.MCMSTimelockProposals = append(output.MCMSTimelockProposals, *p)
		case mcmsProposalRgx.MatchString(name):
			p, err1 := a.loadMCMSProposal(entryPath)
			if err1 != nil {
				return fdeployment.ChangesetOutput{}, err1
			}
			output.MCMSProposals = append(output.MCMSProposals, *p)
		default:
			// Ignore unknown files
			continue
		}
	}

	return output, nil
}

// LoadAddressBookByChangesetKey searches for an address book file in the changeset directory and
// returns the address book as an AddressBookMap.
//
// The search will look for a address book file with a matching name as the domain, env and
// changeset key, returning the first matching file. An error is returned if no matches are found
// or if an error occurs during the search.
//
// Pattern format: "*-<domain>-<env>-<csKey>_addresses.json".
func (a *ArtifactsDir) LoadAddressBookByChangesetKey(csKey string) (*fdeployment.AddressBookMap, error) {
	csDirPath := a.ChangesetDirPath(csKey)
	pattern := fmt.Sprintf("*-%s-%s-%s_%s",
		a.DomainKey(), a.EnvKey(), csKey, AddressBookFileName,
	)

	addrBookPath, err := a.findArtifactPath(csDirPath, pattern)
	if err != nil {
		return nil, err
	}

	return a.loadAddressBook(addrBookPath)
}

// LoadOperationsReports reads the reports from the operations reports directory for the specified changeset key.
func (a *ArtifactsDir) LoadOperationsReports(csKey string) ([]foperations.Report[any, any], error) {
	exists, err := a.ChangesetOperationsReportsFileExists(csKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return []foperations.Report[any, any]{}, nil
	}

	file, err := os.ReadFile(a.getOperationsReportsFilePath(csKey))
	if err != nil {
		return nil, err
	}

	var reports []foperations.Report[json.RawMessage, json.RawMessage]
	err = json.Unmarshal(file, &reports)
	if err != nil {
		return nil, err
	}

	anyReports := make([]foperations.Report[any, any], 0, len(reports))
	for _, r := range reports {
		anyReports = append(anyReports, r.ToGenericReport())
	}

	return anyReports, nil
}

// SaveOperationsReports writes an operations report as JSON to the operations reports directory for
// the specified changeset key.
// if the directory does not exist, it will be created.
// if the file already exists, it will be overwritten.
func (a *ArtifactsDir) SaveOperationsReports(csKey string, reports []foperations.Report[any, any]) error {
	found, err := a.OperationsReportsDirExists()
	if err != nil {
		return err
	}
	if !found {
		err := a.CreateOperationsReportsDir()
		if err != nil {
			return err
		}
	}

	return jsonutils.WriteFile(filepath.Join(a.getOperationsReportsFilePath(csKey)), reports)
}

func (a *ArtifactsDir) getOperationsReportsFilePath(csKey string) string {
	fileName := fmt.Sprintf("%s-reports.%s", csKey, JSONExt)

	return filepath.Join(a.OperationsReportsDirPath(), fileName)
}

// findArtifactPath searches for a file in the specified directory that matches the given pattern.
func (a *ArtifactsDir) findArtifactPath(dirPath string, pattern string) (string, error) {
	var artifactPath string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error finding files: %w", err)
		}

		if !info.IsDir() {
			// We ignore the error here because the only possible returned error is ErrBadPattern,
			// when pattern is malformed.
			//
			// https://pkg.go.dev/path/filepath#Match
			match, _ := filepath.Match(pattern, filepath.Base(path))

			if match {
				artifactPath = path

				return nil
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	if artifactPath == "" {
		return "", fmt.Errorf("%w: no files found matching pattern %s", ErrArtifactNotFound, pattern)
	}

	return artifactPath, nil
}

// loadMCMSProposal reads an MCMS proposal file and returns the proposal.
func (a *ArtifactsDir) loadMCMSProposal(filePath string) (*mcms.Proposal, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Note this does not take into account predecessors
	return mcms.NewProposal(f)
}

// loadMCMSTimelockProposal reads an MCMS timelock proposal file and returns the proposal.
func (a *ArtifactsDir) loadMCMSTimelockProposal(filePath string) (*mcms.TimelockProposal, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return mcms.NewTimelockProposal(f)
}

// loadAddressBook reads an address book file and returns the address book as an AddressBookMap.
func (a *ArtifactsDir) loadAddressBook(addrBookPath string) (*fdeployment.AddressBookMap, error) {
	b, err := os.ReadFile(addrBookPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	addressesByChain := make(map[uint64]map[string]fdeployment.TypeAndVersion)
	if err = json.Unmarshal(b, &addressesByChain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON for address book from path %s: %w",
			addrBookPath, err,
		)
	}

	return fdeployment.NewMemoryAddressBookFromMap(addressesByChain), nil
}

// loadDataStore reads a datastore file and returns the datastore as read-only.
func (a *ArtifactsDir) loadDataStore(dataStorePath string) (fdatastore.DataStore, error) {
	b, err := os.ReadFile(dataStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dataStore fdatastore.MemoryDataStore
	if err = json.Unmarshal(b, &dataStore); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON for datastore from path %s: %w",
			dataStorePath, err,
		)
	}

	return dataStore.Seal(), nil
}

func (a *ArtifactsDir) loadMutableDataStore(dataStorePath string) (fdatastore.MutableDataStore, error) {
	b, err := os.ReadFile(dataStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dataStore fdatastore.MemoryDataStore
	if err = json.Unmarshal(b, &dataStore); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON for datastore from path %s: %w",
			dataStorePath, err,
		)
	}

	return &dataStore, nil
}

// saveArtifact writes an artifact as JSON to the specified changeset directory.
func (a *ArtifactsDir) saveArtifact(k ksuid.KSUID, csKey, name string, v any) error {
	filename := fmt.Sprintf("%s-%s-%s-%s_%s.%s",
		k.String(), a.DomainKey(), a.EnvKey(), csKey, name, JSONExt,
	)

	return jsonutils.WriteFile(filepath.Join(a.ChangesetDirPath(csKey), filename), v)
}

// saveProposalArtifact writes a proposal artifact as JSON to the proposals directory.
func (a *ArtifactsDir) saveProposalArtifact(csKey string, name string, index int, v any) error {
	filename := fmt.Sprintf("%s-%s-%s_%s_%d.%s", a.DomainKey(), a.EnvKey(), csKey, name, index, JSONExt)
	if a.timestamp != "" {
		filename = fmt.Sprintf("%s-%s", a.timestamp, filename)
	}

	return jsonutils.WriteFile(filepath.Join(a.getProposalDir(), filename), v)
}

// saveDecodedProposalArtifact writes a decoded proposal artifact as JSON to the decoded proposals directory.
func (a *ArtifactsDir) saveDecodedProposalArtifact(csKey string, name string, index int, data string) error {
	filename := fmt.Sprintf("%s-%s-%s_%s_%d_decoded.%s", a.DomainKey(), a.EnvKey(), csKey, name, index, TxtExt)
	if a.timestamp != "" {
		filename = fmt.Sprintf("%s-%s", a.timestamp, filename)
	}

	return os.WriteFile(filepath.Join(a.getDecodedProposalDir(), filename), []byte(data), 0600)
}

// getDirectoryPath returns the directory path for changesets and durable pipelines.
func (a *ArtifactsDir) getDirectoryPath(basePathFunc func() string) string {
	return basePathFunc()
}

// getProposalDir returns the directory path for proposals.
func (a *ArtifactsDir) getProposalDir() string {
	return a.getDirectoryPath(a.ProposalsDirPath)
}

// getDecodedProposalDir returns the directory path for proposals.
func (a *ArtifactsDir) getDecodedProposalDir() string {
	return a.getDirectoryPath(a.DecodedProposalsDirPath)
}

// getArchivedProposalDir returns the directory path for archived proposals.
func (a *ArtifactsDir) getArchivedProposalDir() string {
	return a.getDirectoryPath(a.ArchivedProposalsDirPath)
}

// marshalIndentAndSort marshals a map of addresses to their types and versions into a sorted JSON object format.
// This is a workaround to ensure that the JSON output is deterministic and sorted by chain selector and address which
// helps to avoid merge conflicts in git when multiple migrations are run.
func marshalIndentAndSort(addrMap map[uint64]map[string]fdeployment.TypeAndVersion) ([]byte, error) {
	// Sort the outer map keys (chain selectors)
	chainSelectors := make([]uint64, 0, len(addrMap))
	for k := range addrMap {
		chainSelectors = append(chainSelectors, k)
	}
	sort.Slice(chainSelectors, func(i, j int) bool { return chainSelectors[i] < chainSelectors[j] })

	var buf bytes.Buffer
	buf.WriteString("{\n")

	// Iterate through sorted chain selectors
	for i, chainSelector := range chainSelectors {
		// Write indentation for chain selector
		buf.WriteString("  ")

		// Write chain selector as JSON string
		buf.WriteString(strconv.Quote(strconv.FormatUint(chainSelector, 10)))
		buf.WriteString(": {\n")

		// Get the inner map for this chain selector
		innerMap := addrMap[chainSelector]

		// Sort the inner map keys (addresses)
		addresses := make([]string, 0, len(innerMap))
		for addr := range innerMap {
			addresses = append(addresses, addr)
		}
		sort.Strings(addresses)

		// Iterate through sorted addresses
		for j, addr := range addresses {
			// Write indentation for address
			buf.WriteString("    ")

			// Write address as JSON string
			buf.WriteString(strconv.Quote(addr))
			buf.WriteString(": ")

			// Marshal the TypeAndVersion value
			valBytes, err := json.Marshal(innerMap[addr])
			if err != nil {
				return nil, err
			}

			// Indent the TypeAndVersion JSON
			var indentedVal bytes.Buffer
			err = json.Indent(&indentedVal, valBytes, "    ", "  ")
			if err != nil {
				return nil, err
			}
			buf.Write(indentedVal.Bytes())

			// Comma between inner entries
			if j < len(addresses)-1 {
				buf.WriteString(",\n")
			} else {
				buf.WriteString("\n")
			}
		}

		// Close the inner object with proper indentation
		buf.WriteString("  }")

		// Comma between outer entries
		if i < len(chainSelectors)-1 {
			buf.WriteString(",\n")
		} else {
			buf.WriteString("\n")
		}
	}

	buf.WriteString("}")

	return buf.Bytes(), nil
}
