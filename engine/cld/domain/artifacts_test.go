package domain

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmsv2 "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

func Test_Artifacts_DirPath(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/artifacts", arts.ArtifactsDirPath())

	err := arts.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/artifacts/durable_pipelines", arts.ArtifactsDirPath())
}

func Test_Artifacts_ProposalsDirPath(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/proposals", arts.ProposalsDirPath())

	err := arts.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/proposals", arts.ProposalsDirPath())
}

func Test_Artifacts_ArchivedProposalsDirPath(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/archived_proposals", arts.ArchivedProposalsDirPath())

	err := arts.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/archived_proposals", arts.ArchivedProposalsDirPath())
}

func Test_OperationsReports_DirPath(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/operations_reports", arts.OperationsReportsDirPath())

	err := arts.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/operations_reports/durable_pipelines", arts.OperationsReportsDirPath())
}

func Test_Artifacts_DomainKey(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("", "ccip", "staging")

	assert.Equal(t, "ccip", arts.DomainKey())
}

func Test_Artifacts_EnvKey(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("", "ccip", "staging")

	assert.Equal(t, "staging", arts.EnvKey())
}

func Test_Artifacts_MigrationDirPath(t *testing.T) {
	t.Parallel()

	timestamp := "1234567890123456789"

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/artifacts/0001_initial", arts.MigrationDirPath("0001_initial"))

	err := arts.SetDurablePipelines(timestamp)
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/artifacts/durable_pipelines/0001_initial/"+arts.timestamp, arts.MigrationDirPath("0001_initial"))
}

func Test_Artifacts_ProposalDirPath(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/proposals", arts.ProposalsDirPath())

	err := arts.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/proposals", arts.ProposalsDirPath())
}

func Test_Artifacts_ArchivedProposalDirPath(t *testing.T) {
	t.Parallel()

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/archived_proposals", arts.ArchivedProposalsDirPath())

	err := arts.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/archived_proposals", arts.ArchivedProposalsDirPath())
}

func Test_Artifacts_CreateMigrationDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	arts := fixture.artifactsDir

	err := arts.CreateMigrationDir("0001_initial")
	require.NoError(t, err)

	got, err := arts.MigrationDirExists("0001_initial")
	require.NoError(t, err)
	assert.True(t, got)
}

func Test_Artifacts_CreateProposalsDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	arts := fixture.artifactsDir

	tests := []struct {
		name         string
		migrationKey string
	}{
		{
			name:         "create proposals dir",
			migrationKey: "0001_initial",
		},
		{
			name:         "create proposals dir for durable pipelines",
			migrationKey: "initial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := arts.CreateProposalsDir()
			require.NoError(t, err)

			got, err := arts.ProposalsDirExists()
			require.NoError(t, err)
			assert.True(t, got)
		})
	}
}

func Test_Artifacts_CreateArchivedProposalsDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	arts := fixture.artifactsDir

	tests := []struct {
		name string
	}{
		{
			name: "create archived proposals dir",
		},
		{
			name: "create archived proposals dir for durable pipelines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := arts.CreateArchivedProposalsDir()
			require.NoError(t, err)

			got, err := arts.ArchiveProposalsDirExists()
			require.NoError(t, err)
			assert.True(t, got)
		})
	}
}

func Test_Artifacts_CreateOperationsReportsDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	arts := fixture.artifactsDir

	tests := []struct {
		name string
	}{
		{
			name: "create operations reports dir",
		},
		{
			name: "create operations reports dir for durable pipelines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := arts.CreateOperationsReportsDir()
			require.NoError(t, err)

			got, err := arts.OperationsReportsDirExists()
			require.NoError(t, err)
			assert.True(t, got)
		})
	}
}

func Test_Artifacts_RemoveMigrationDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)
	arts := fixture.artifactsDir

	err := arts.CreateMigrationDir("0001_initial")
	require.NoError(t, err)

	got, err := arts.MigrationDirExists("0001_initial")
	require.NoError(t, err)
	assert.True(t, got)

	err = arts.RemoveMigrationDir("0001_initial")
	require.NoError(t, err)

	got, err = arts.MigrationDirExists("0001_initial")
	require.NoError(t, err)
	assert.False(t, got)
}

func Test_Artifacts_MigrationDirExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		beforeFunc       func(*testing.T, *ArtifactsDir)
		giveMigrationKey string
		want             bool
		wantErr          string
	}{
		{
			name: "exists",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.CreateMigrationDir("0001_initial")
				require.NoError(t, err)
			},
			giveMigrationKey: "0001_initial",
			want:             true,
		},
		{
			name:             "does not exist",
			giveMigrationKey: "0001_initial",
			want:             false,
		},
		{
			name: "is a file",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				f, err := os.Create(filepath.Join(artsDir.ArtifactsDirPath(), "0001_initial"))
				require.NoError(t, err)
				defer f.Close()
			},
			giveMigrationKey: "0001_initial",
			want:             false,
			wantErr:          "expected directory, got file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, artsDir)
			}

			got, err := artsDir.MigrationDirExists(tt.giveMigrationKey)
			assert.Equal(t, tt.want, got)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_Artifacts_OperationsReportsDirExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *ArtifactsDir)
		want       bool
	}{
		{
			name: "exists",
			want: true,
		},
		{
			name: "does not exist",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := os.RemoveAll(artsDir.OperationsReportsDirPath())
				require.NoError(t, err)
			},
			want: false,
		},
		{
			name: "is a file",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()
				err := os.RemoveAll(artsDir.OperationsReportsDirPath())
				require.NoError(t, err)

				f, err := os.Create(filepath.Join(artsDir.ArtifactsDirPath(), "operations_reports"))
				require.NoError(t, err)
				defer f.Close()
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, artsDir)
			}

			got, err := artsDir.OperationsReportsDirExists()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Artifacts_MigrationOperationsReportsFileExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		beforeFunc       func(*testing.T, *ArtifactsDir)
		giveMigrationKey string
		want             bool
		wantErr          string
	}{
		{
			name: "exists",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := os.WriteFile(
					filepath.Join(artsDir.OperationsReportsDirPath(), "0001_initial-reports.json"),
					[]byte(`{}`),

					0600,
				)
				require.NoError(t, err)
			},
			giveMigrationKey: "0001_initial",
			want:             true,
		},
		{
			name:             "does not exist",
			giveMigrationKey: "0001_initial",
			want:             false,
		},
		{
			name: "is a directory",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := os.Mkdir(filepath.Join(artsDir.OperationsReportsDirPath(), "0001_initial-reports.json"), 0755)
				require.NoError(t, err)
			},
			giveMigrationKey: "0001_initial",
			want:             false,
			wantErr:          "expected file, got directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, artsDir)
			}

			got, err := artsDir.MigrationOperationsReportsFileExists(tt.giveMigrationKey)
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

func Test_Artifacts_SaveChangesetOutput_LoadChangesetOutput(t *testing.T) {
	t.Parallel()

	var (
		js1 = testJobSpec{A: "1", B: "2"}
		js2 = testJobSpec{A: "3", B: "4"}

		job1 = cldf.ProposedJob{
			JobID: "job_123",
			Node:  "node1",
			Spec:  js1.MustMarshal(),
		}
		job2 = cldf.ProposedJob{
			JobID: "job_234",
			Node:  "node2",
			Spec:  js2.MustMarshal(),
		}

		validUntilUnixTime = uint32(time.Now().Add(time.Hour).Unix()) //nolint:gosec // This won't overflow until 7 Feb 2106, and would also cause MCMS to fail anyway

		mcmsProposals = []mcmsv2.Proposal{
			{
				BaseProposal: mcmsv2.BaseProposal{
					Kind:       mcmstypes.KindProposal,
					Version:    "v1",
					ValidUntil: validUntilUnixTime,
					Signatures: []mcmstypes.Signature{},
					ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
						mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {
							AdditionalFields: []byte("null"),
						},
					},
				},
				Operations: []mcmstypes.Operation{
					{
						ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
						Transaction: mcmstypes.Transaction{
							To:               "0x123",
							Data:             []byte{0x01, 0x02, 0x03},
							AdditionalFields: json.RawMessage(`{}`),
						},
					},
				},
			},
			{
				BaseProposal: mcmsv2.BaseProposal{
					Kind:       mcmstypes.KindProposal,
					Version:    "v1",
					ValidUntil: validUntilUnixTime,
					Signatures: []mcmstypes.Signature{},
					ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
						mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {
							AdditionalFields: []byte("null"),
						},
					},
				},
				Operations: []mcmstypes.Operation{
					{
						ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
						Transaction: mcmstypes.Transaction{
							To:               "0x321",
							Data:             []byte{0x01, 0x02, 0x03},
							AdditionalFields: json.RawMessage(`{}`),
						},
					},
				},
			},
		}

		mcmsTimelockProposals = []mcmsv2.TimelockProposal{
			{
				BaseProposal: mcmsv2.BaseProposal{
					Kind:       mcmstypes.KindTimelockProposal,
					Version:    "v1",
					ValidUntil: validUntilUnixTime,
					Signatures: []mcmstypes.Signature{},
					ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
						mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {
							AdditionalFields: []byte("null"),
						},
					},
				},
				Action: mcmstypes.TimelockActionSchedule,
				TimelockAddresses: map[mcmstypes.ChainSelector]string{
					mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): "0x123",
				},
				Operations: []mcmstypes.BatchOperation{
					{
						ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
						Transactions: []mcmstypes.Transaction{
							{
								To:               "0x123",
								Data:             []byte{0x01, 0x02, 0x03},
								AdditionalFields: json.RawMessage(`{}`),
							},
						},
					},
				},
			},
			{
				BaseProposal: mcmsv2.BaseProposal{
					Kind:       mcmstypes.KindTimelockProposal,
					Version:    "v1",
					ValidUntil: validUntilUnixTime,
					Signatures: []mcmstypes.Signature{},
					ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
						mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {
							AdditionalFields: []byte("null"),
						},
					},
				},
				Action: mcmstypes.TimelockActionSchedule,
				TimelockAddresses: map[mcmstypes.ChainSelector]string{
					mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): "0x123",
				},
				Operations: []mcmstypes.BatchOperation{
					{
						ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
						Transactions: []mcmstypes.Transaction{
							{
								To:               "0x124",
								Data:             []byte{0x01, 0x02, 0x03},
								AdditionalFields: json.RawMessage(`{}`),
							},
						},
					},
				},
			},
		}

		addrBook = createAddressBookMap(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, "0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4",
		)

		dataStore = createDataStore(t,
			"Contract", version1_0_0,
			chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
			"0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4",
			"qtest1",
		)
	)

	tests := []struct {
		name       string
		giveOutput cldf.ChangesetOutput
		want       cldf.ChangesetOutput
	}{
		{
			name:       "empty changeset output",
			giveOutput: cldf.ChangesetOutput{},
		},
		{
			name: "changeset output with job specs",
			giveOutput: cldf.ChangesetOutput{
				JobSpecs: map[string][]string{
					"node1": {js1.MustMarshal()},
					"node2": {js2.MustMarshal()},
				},
			},
			want: cldf.ChangesetOutput{
				JobSpecs: map[string][]string{
					"node1": {js1.MustMarshal()},
					"node2": {js2.MustMarshal()},
				},
			},
		},
		{
			name: "changeset output with addresses",
			giveOutput: cldf.ChangesetOutput{
				AddressBook: addrBook,
			},
			want: cldf.ChangesetOutput{
				AddressBook: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
					chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
						"0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4": cldf.TypeAndVersion{
							Type:    "Contract",
							Version: version1_0_0,
							Labels:  nil,
						},
					},
				}),
			},
		},
		{
			name: "changeset output with datastore",
			giveOutput: cldf.ChangesetOutput{
				DataStore: dataStore,
			},
			want: cldf.ChangesetOutput{
				DataStore: dataStore,
			},
		},
		{
			name: "changeset output with jobs",
			giveOutput: cldf.ChangesetOutput{
				Jobs: []cldf.ProposedJob{job1, job2},
			},
			want: cldf.ChangesetOutput{
				Jobs: []cldf.ProposedJob{job1, job2},
			},
		},
		{
			name: "changeset output with mcms proposals",
			giveOutput: cldf.ChangesetOutput{
				MCMSProposals: mcmsProposals,
			},
			want: cldf.ChangesetOutput{
				MCMSProposals: mcmsProposals,
			},
		},
		{
			name: "changeset output with mcms timelock proposals",
			giveOutput: cldf.ChangesetOutput{
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
			want: cldf.ChangesetOutput{
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
		},
		{
			name: "changeset output with all proposals",
			giveOutput: cldf.ChangesetOutput{
				MCMSProposals:         mcmsProposals,
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
			want: cldf.ChangesetOutput{
				MCMSProposals:         mcmsProposals,
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
		},
	}

	for _, tt := range tests {
		t.Run("migrations "+tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)

			artsDir := fixture.artifactsDir

			// Test saving and loading artifacts for migrations
			testArtifactSaveAndLoad(t, artsDir, tt.giveOutput, tt.want)
		})
	}

	for _, tt := range tests {
		t.Run("durable pipelines "+tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)

			artsDir := fixture.artifactsDir

			err := artsDir.SetDurablePipelines("1234567890123456789")
			require.NoError(t, err)

			// Test saving and loading artifacts for durable pipelines
			testArtifactSaveAndLoad(t, artsDir, tt.giveOutput, tt.want)
		})
	}
}

func testArtifactSaveAndLoad(t *testing.T, artsDir *ArtifactsDir, giveOutput cldf.ChangesetOutput, want cldf.ChangesetOutput) {
	t.Helper()

	// Save the changeset output as artifacts
	err := artsDir.SaveChangesetOutput("0001_initial", giveOutput)
	require.NoError(t, err)

	// Load the changeset output from the artifacts
	got, err := artsDir.LoadChangesetOutput("0001_initial")
	require.NoError(t, err)

	// Compare the loaded changeset output with the original
	assert.Equal(t, want, got)
}

func Test_Artifacts_LoadAddressBookByMigrationKey(t *testing.T) {
	t.Parallel()

	addrBook := createAddressBookMap(t,
		"Contract", version1_0_0,
		chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, "0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4",
	)

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *ArtifactsDir)
		giveMigKey string
		want       cldf.AddressBook
		wantErr    string
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					AddressBook: addrBook,
				})
				require.NoError(t, err)
			},
			giveMigKey: "0001_initial",
			want: cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					"0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4": cldf.TypeAndVersion{
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
			}),
		},
		{
			name:       "migration dir does not exist",
			giveMigKey: "invalid",
			wantErr:    "error finding files",
		},
		{
			name: "artifact does not exist",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_no_address_book", cldf.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveMigKey: "0001_no_address_book",
			wantErr:    "no files found matching pattern",
		},
		{
			name: "address book is malformed JSON",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.CreateMigrationDir("0001_malformed")
				require.NoError(t, err)

				err = os.WriteFile(
					filepath.Join(artsDir.MigrationDirPath("0001_malformed"), "xxx-ccip-staging-0001_malformed_addresses.json"),
					[]byte("malformed"),
					0600,
				)
				require.NoError(t, err)
			},
			giveMigKey: "0001_malformed",
			wantErr:    "failed to unmarshal JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, artsDir)
			}

			got, err := artsDir.LoadAddressBookByMigrationKey(tt.giveMigKey)

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

func Test_Artifacts_LoadDataStoreByMigrationKey(t *testing.T) {
	t.Parallel()

	dataStore := createDataStore(t,
		"Contract", version1_0_0,
		chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
		"0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4",
		"qtest1",
	)

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *ArtifactsDir)
		giveMigKey string
		want       datastore.DataStore
		wantErr    string
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_initial", cldf.ChangesetOutput{
					DataStore: dataStore,
				})
				require.NoError(t, err)
			},
			giveMigKey: "0001_initial",
			want:       dataStore.Seal(),
		},
		{
			name:       "migration dir does not exist",
			giveMigKey: "invalid",
			wantErr:    "error finding files",
		},
		{
			name: "artifact does not exist",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_no_datastore", cldf.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveMigKey: "0001_no_datastore",
			wantErr:    "no files found matching pattern",
		},
		{
			name: "address book is malformed JSON",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.CreateMigrationDir("0001_malformed")
				require.NoError(t, err)

				err = os.WriteFile(
					filepath.Join(artsDir.MigrationDirPath("0001_malformed"), "xxx-ccip-staging-0001_malformed_datastore.json"),
					[]byte("malformed"),
					0600,
				)
				require.NoError(t, err)
			},
			giveMigKey: "0001_malformed",
			wantErr:    "failed to unmarshal JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, artsDir)
			}

			got, err := artsDir.LoadDataStoreByMigrationKey(tt.giveMigKey)

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

func Test_Artifacts_SaveAndLoadOperationsReport(t *testing.T) {
	t.Parallel()

	migrationKey := "0001_initial"

	report := operations.NewReport[any, any](
		operations.Definition{
			ID:          "test",
			Version:     semver.MustParse("1.0.0"),
			Description: "test description",
		},
		chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
		2,
		errors.New("test error"),
		"123",
	)

	tests := []struct {
		name        string
		beforeFunc  func(*testing.T, *ArtifactsDir)
		giveMigKey  string
		want        []operations.Report[any, any]
		wantLoadErr string
		wantSaveErr string
	}{
		{
			name:       "success save and load",
			giveMigKey: migrationKey,
			want:       []operations.Report[any, any]{report},
		},
		{
			name:       "success save and load - durable pipelines",
			giveMigKey: migrationKey,
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()
				err := artsDir.SetDurablePipelines("1749186682460987000")
				require.NoError(t, err)
			},
			want: []operations.Report[any, any]{report},
		},
		{
			name:       "report does not exist - return empty slice",
			giveMigKey: "invalid",
			want:       []operations.Report[any, any]{},
		}, {
			name:       "success - directory does not exist - should create it",
			giveMigKey: migrationKey,
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()
				err := os.RemoveAll(artsDir.OperationsReportsDirPath())
				require.NoError(t, err)
			},
			want: []operations.Report[any, any]{report},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, artsDir)
			}

			err := artsDir.SaveOperationsReports(migrationKey, []operations.Report[any, any]{report})
			if tt.wantSaveErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantSaveErr)
			} else {
				require.NoError(t, err)
			}

			got, err := artsDir.LoadOperationsReports(tt.giveMigKey)

			if tt.wantLoadErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantLoadErr)
			} else {
				require.NoError(t, err)
				assert.Len(t, got, len(tt.want))
				if len(tt.want) > 0 {
					assert.Equal(t, report.ID, got[0].ID)
					assert.Equal(t, report.Def, got[0].Def)
					assert.Equal(t, report.Timestamp.Format(time.RFC3339Nano), got[0].Timestamp.Format(time.RFC3339Nano))
					assert.Equal(t,
						strconv.FormatUint(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, 10),
						string(got[0].Input.(json.RawMessage)))
					assert.Equal(t, "2", string(got[0].Output.(json.RawMessage)))
					assert.Equal(t, report.Err, got[0].Err)
					assert.Equal(t, report.ChildOperationReports, tt.want[0].ChildOperationReports)
				}
			}
		})
	}
}

func Test_Artifacts_SaveAndLoadMultipleProposals(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)
	migrationKey := "0001_initial"
	artsDir := fixture.artifactsDir

	validUntilUnixTime := uint32(time.Now().Add(time.Hour).Unix()) //nolint:gosec // This won't overflow until 7 Feb 2106, and would also cause MCMS to fail anyway

	proposals := []mcmsv2.Proposal{
		{
			BaseProposal: mcmsv2.BaseProposal{
				Kind:       mcmstypes.KindProposal,
				Version:    "v1",
				ValidUntil: validUntilUnixTime,
				Signatures: []mcmstypes.Signature{},
				ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
					mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {},
				},
			},
			Operations: []mcmstypes.Operation{
				{
					ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transaction: mcmstypes.Transaction{
						To:               "0x321",
						Data:             []byte{0x01, 0x02, 0x03},
						AdditionalFields: json.RawMessage(`{}`),
					},
				},
			},
		},
		{
			BaseProposal: mcmsv2.BaseProposal{
				Kind:       mcmstypes.KindProposal,
				Version:    "v1",
				ValidUntil: validUntilUnixTime,
				Signatures: []mcmstypes.Signature{},
				ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
					mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {},
				},
			},
			Operations: []mcmstypes.Operation{
				{
					ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
					Transaction: mcmstypes.Transaction{
						To:               "0x321",
						Data:             []byte{0x01, 0x02, 0x03},
						AdditionalFields: json.RawMessage(`{}`),
					},
				},
			},
		},
	}

	// Save multiple proposals
	require.NoError(t, artsDir.CreateMigrationDir(migrationKey))
	for i, proposal := range proposals {
		err := artsDir.saveProposalArtifact(migrationKey, ArtifactMCMSProposal, i, proposal)
		require.NoError(t, err)
		err = artsDir.saveDecodedProposalArtifact(migrationKey, ArtifactMCMSProposal, i, "some decoded proposal")
		require.NoError(t, err)
	}

	// Verify proposal files were created with correct indexes
	files, err := os.ReadDir(artsDir.ProposalsDirPath())
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	require.NoError(t, err)
	assert.Len(t, files, len(proposals))

	for i, file := range files {
		assert.Contains(t, file.Name(), migrationKey)
		assert.Contains(t, file.Name(), ArtifactMCMSProposal)
		assert.Contains(t, file.Name(), "_"+strconv.Itoa(i))
	}

	// Load proposals and verify
	exists, err := artsDir.MigrationDirExists(migrationKey)
	require.NoError(t, err)
	require.True(t, exists)
	loadedProposals, err := artsDir.LoadChangesetOutput(migrationKey)
	require.NoError(t, err)
	assert.Len(t, loadedProposals.MCMSProposals, len(proposals))

	for i, proposal := range loadedProposals.MCMSProposals {
		assert.Equal(t, proposals[i].Version, proposal.Version)
		assert.Equal(t, proposals[i].ValidUntil, proposal.ValidUntil)
	}
}

func Test_Artifacts_SetDurablePipelinesTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		timestamp string
		wantErr   string
	}{
		{
			name:      "valid timestamp",
			timestamp: "1234567890123456789",
		},
		{
			name:      "invalid timestamp precision",
			timestamp: "123456789",
			wantErr:   "timestamp must be in nanoseconds",
		},
		{
			name:      "invalid timestamp format",
			timestamp: "abc",
			wantErr:   "invalid timestamp",
		},
		{
			name:      "no timestamp",
			timestamp: "",
			wantErr:   "invalid timestamp",
		},
		{
			name:      "zero timestamp",
			timestamp: "0",
			wantErr:   "timestamp must be greater than 0",
		},
		{
			name:      "negative timestamp",
			timestamp: "-1234567890",
			wantErr:   "timestamp must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			arts := NewArtifactsDir("domains", "exemplar", "testnet")

			err := arts.SetDurablePipelines(tt.timestamp)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.timestamp, arts.timestamp)
			}
		})
	}
}

func Test_Artifacts_SaveAddressBookInSortedOrder(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)
	artsDir := fixture.artifactsDir

	// Set timestamp for durable pipelines directory structure
	err := artsDir.SetDurablePipelines("1234567890123456789")
	require.NoError(t, err)

	migKey := "0001_sorted_addresses"

	// Create address book with intentionally unsorted entries
	// Higher chain selector first, unsorted addresses
	addrBook := cldf.NewMemoryAddressBookFromMap(map[uint64]map[string]cldf.TypeAndVersion{
		// Chain selector 456 (should appear second)
		456: {
			"0xC3A1B2D4E5F6789012345678901234567890ABCD": {Type: "ContractC", Version: version1_0_0},
			"0xA1B2C3D4E5F6789012345678901234567890ABCD": {Type: "ContractA", Version: version1_0_0},
			"0xB2C3D4E5F6789012345678901234567890ABCDEF": {Type: "ContractB", Version: version1_0_0},
		},
		// Chain selector 123 (should appear first)
		123: {
			"0xZ9F8E7D6C5B4A3210987654321098765432109AB": {Type: "ContractZ", Version: version1_0_0},
			"0xX1A2B3C4D5E6F7890123456789012345678901BC": {Type: "ContractX", Version: version1_0_0},
			"0xY5F4E3D2C1B0A9876543210987654321098765CD": {Type: "ContractY", Version: version1_0_0},
		},
	})

	// Save address book
	err = artsDir.SaveChangesetOutput(migKey, cldf.ChangesetOutput{
		AddressBook: addrBook,
	})
	require.NoError(t, err)

	// Find the address book file
	pattern := "*_" + ArtifactAddress + "." + JSONExt
	addrBookPath, err := artsDir.findArtifactPath(artsDir.MigrationDirPath(migKey), pattern)
	require.NoError(t, err)

	// Read the file contents directly
	fileContents, err := os.ReadFile(addrBookPath)
	require.NoError(t, err)

	fileStr := string(fileContents)

	// Verify the JSON structure is sorted correctly
	// Chain selectors should be in numeric order: 123, 456
	// Addresses within each chain should be in alphabetical order
	expected := `{
  "123": {
    "0xX1A2B3C4D5E6F7890123456789012345678901BC": {
      "Type": "ContractX",
      "Version": "1.0.0"
    },
    "0xY5F4E3D2C1B0A9876543210987654321098765CD": {
      "Type": "ContractY",
      "Version": "1.0.0"
    },
    "0xZ9F8E7D6C5B4A3210987654321098765432109AB": {
      "Type": "ContractZ",
      "Version": "1.0.0"
    }
  },
  "456": {
    "0xA1B2C3D4E5F6789012345678901234567890ABCD": {
      "Type": "ContractA",
      "Version": "1.0.0"
    },
    "0xB2C3D4E5F6789012345678901234567890ABCDEF": {
      "Type": "ContractB",
      "Version": "1.0.0"
    },
    "0xC3A1B2D4E5F6789012345678901234567890ABCD": {
      "Type": "ContractC",
      "Version": "1.0.0"
    }
  }
}`

	assert.Equal(t, expected, fileStr)

	// Also verify the saved file can be loaded back correctly
	loadedOutput, err := artsDir.LoadChangesetOutput(migKey)
	require.NoError(t, err)
	//nolint:staticcheck
	require.NotNil(t, loadedOutput.AddressBook)
}

func Test_getOperationsReportsFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		migKey             string
		wantFileName       string
		isDurablePipelines bool
	}{
		{
			name:         "migration",
			migKey:       "0001_initial",
			wantFileName: "0001_initial-reports.json",
		},
		{
			name:               "durable pipelines",
			migKey:             "0001_initial",
			isDurablePipelines: true,
			wantFileName:       "0001_initial-reports.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.isDurablePipelines {
				err := artsDir.SetDurablePipelines("1234567890123456789")
				require.NoError(t, err)
			}

			got := artsDir.getOperationsReportsFilePath(tt.migKey)
			assert.Equal(t, filepath.Join(artsDir.OperationsReportsDirPath(), tt.wantFileName), got)
		})
	}
}
