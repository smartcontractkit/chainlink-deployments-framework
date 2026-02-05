package domain

import (
	"encoding/json"
	"errors"
	"fmt"
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

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
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

func Test_Artifacts_ChangesetDirPath(t *testing.T) {
	t.Parallel()

	timestamp := "1234567890123456789"

	arts := NewArtifactsDir("domains", "ccip", "staging")

	assert.Equal(t, "domains/ccip/staging/artifacts/0001_initial", arts.ChangesetDirPath("0001_initial"))

	err := arts.SetDurablePipelines(timestamp)
	require.NoError(t, err)

	assert.Equal(t, "domains/ccip/staging/artifacts/durable_pipelines/0001_initial/"+arts.timestamp, arts.ChangesetDirPath("0001_initial"))
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

func Test_Artifacts_CreateChangesetDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	arts := fixture.artifactsDir

	err := arts.CreateChangesetDir("0001_initial")
	require.NoError(t, err)

	got, err := arts.ChangesetDirExists("0001_initial")
	require.NoError(t, err)
	assert.True(t, got)
}

func Test_Artifacts_CreateProposalsDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)

	arts := fixture.artifactsDir

	tests := []struct {
		name         string
		changesetKey string
	}{
		{
			name:         "create proposals dir",
			changesetKey: "0001_initial",
		},
		{
			name:         "create proposals dir for durable pipelines",
			changesetKey: "initial",
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

func Test_Artifacts_RemoveChangesetDir(t *testing.T) {
	t.Parallel()

	fixture := setupTestDomainsFS(t)
	arts := fixture.artifactsDir

	err := arts.CreateChangesetDir("0001_initial")
	require.NoError(t, err)

	got, err := arts.ChangesetDirExists("0001_initial")
	require.NoError(t, err)
	assert.True(t, got)

	err = arts.RemoveChangesetDir("0001_initial")
	require.NoError(t, err)

	got, err = arts.ChangesetDirExists("0001_initial")
	require.NoError(t, err)
	assert.False(t, got)
}

func Test_Artifacts_ChangesetDirExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		beforeFunc       func(*testing.T, *ArtifactsDir)
		giveChangesetKey string
		want             bool
		wantErr          string
	}{
		{
			name: "exists",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.CreateChangesetDir("0001_initial")
				require.NoError(t, err)
			},
			giveChangesetKey: "0001_initial",
			want:             true,
		},
		{
			name:             "does not exist",
			giveChangesetKey: "0001_initial",
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
			giveChangesetKey: "0001_initial",
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

			got, err := artsDir.ChangesetDirExists(tt.giveChangesetKey)
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

func Test_Artifacts_ChangesetOperationsReportsFileExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		beforeFunc       func(*testing.T, *ArtifactsDir)
		giveChangesetKey string
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
			giveChangesetKey: "0001_initial",
			want:             true,
		},
		{
			name:             "does not exist",
			giveChangesetKey: "0001_initial",
			want:             false,
		},
		{
			name: "is a directory",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := os.Mkdir(filepath.Join(artsDir.OperationsReportsDirPath(), "0001_initial-reports.json"), 0755)
				require.NoError(t, err)
			},
			giveChangesetKey: "0001_initial",
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

			got, err := artsDir.ChangesetOperationsReportsFileExists(tt.giveChangesetKey)
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

		job1 = fdeployment.ProposedJob{
			JobID: "job_123",
			Node:  "node1",
			Spec:  js1.MustMarshal(),
		}
		job2 = fdeployment.ProposedJob{
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
		giveOutput fdeployment.ChangesetOutput
		want       fdeployment.ChangesetOutput
	}{
		{
			name:       "empty changeset output",
			giveOutput: fdeployment.ChangesetOutput{},
		},
		{
			name: "changeset output with job specs",
			giveOutput: fdeployment.ChangesetOutput{
				JobSpecs: map[string][]string{
					"node1": {js1.MustMarshal()},
					"node2": {js2.MustMarshal()},
				},
			},
			want: fdeployment.ChangesetOutput{
				JobSpecs: map[string][]string{
					"node1": {js1.MustMarshal()},
					"node2": {js2.MustMarshal()},
				},
			},
		},
		{
			name: "changeset output with addresses",
			giveOutput: fdeployment.ChangesetOutput{
				AddressBook: addrBook,
			},
			want: fdeployment.ChangesetOutput{
				AddressBook: fdeployment.NewMemoryAddressBookFromMap(map[uint64]map[string]fdeployment.TypeAndVersion{
					chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
						"0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4": fdeployment.TypeAndVersion{
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
			giveOutput: fdeployment.ChangesetOutput{
				DataStore: dataStore,
			},
			want: fdeployment.ChangesetOutput{
				DataStore: dataStore,
			},
		},
		{
			name: "changeset output with jobs",
			giveOutput: fdeployment.ChangesetOutput{
				Jobs: []fdeployment.ProposedJob{job1, job2},
			},
			want: fdeployment.ChangesetOutput{
				Jobs: []fdeployment.ProposedJob{job1, job2},
			},
		},
		{
			name: "changeset output with mcms proposals",
			giveOutput: fdeployment.ChangesetOutput{
				MCMSProposals: mcmsProposals,
			},
			want: fdeployment.ChangesetOutput{
				MCMSProposals: mcmsProposals,
			},
		},
		{
			name: "changeset output with mcms timelock proposals",
			giveOutput: fdeployment.ChangesetOutput{
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
			want: fdeployment.ChangesetOutput{
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
		},
		{
			name: "changeset output with all proposals",
			giveOutput: fdeployment.ChangesetOutput{
				MCMSProposals:         mcmsProposals,
				MCMSTimelockProposals: mcmsTimelockProposals,
			},
			want: fdeployment.ChangesetOutput{
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

func testArtifactSaveAndLoad(t *testing.T, artsDir *ArtifactsDir, giveOutput fdeployment.ChangesetOutput, want fdeployment.ChangesetOutput) {
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

func Test_Artifacts_LoadAddressBookByChangesetKey(t *testing.T) {
	t.Parallel()

	addrBook := createAddressBookMap(t,
		"Contract", version1_0_0,
		chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector, "0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4",
	)

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *ArtifactsDir)
		giveCsKey  string
		want       fdeployment.AddressBook
		wantErr    string
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_initial", fdeployment.ChangesetOutput{
					AddressBook: addrBook,
				})
				require.NoError(t, err)
			},
			giveCsKey: "0001_initial",
			want: fdeployment.NewMemoryAddressBookFromMap(map[uint64]map[string]fdeployment.TypeAndVersion{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {
					"0xAeeFF49471aB5B3d14D2FeA4079bF075d452E5F4": fdeployment.TypeAndVersion{
						Type:    "Contract",
						Version: version1_0_0,
						Labels:  nil,
					},
				},
			}),
		},
		{
			name:      "migration dir does not exist",
			giveCsKey: "invalid",
			wantErr:   "error finding files",
		},
		{
			name: "artifact does not exist",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_no_address_book", fdeployment.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveCsKey: "0001_no_address_book",
			wantErr:   "no files found matching pattern",
		},
		{
			name: "address book is malformed JSON",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.CreateChangesetDir("0001_malformed")
				require.NoError(t, err)

				err = os.WriteFile(
					filepath.Join(artsDir.ChangesetDirPath("0001_malformed"), "xxx-ccip-staging-0001_malformed_addresses.json"),
					[]byte("malformed"),
					0600,
				)
				require.NoError(t, err)
			},
			giveCsKey: "0001_malformed",
			wantErr:   "failed to unmarshal JSON",
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

			got, err := artsDir.LoadAddressBookByChangesetKey(tt.giveCsKey)

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

func Test_Artifacts_LoadDataStoreByChangesetKey(t *testing.T) {
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
		giveCsKey  string
		want       fdatastore.DataStore
		wantErr    string
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_initial", fdeployment.ChangesetOutput{
					DataStore: dataStore,
				})
				require.NoError(t, err)
			},
			giveCsKey: "0001_initial",
			want:      dataStore.Seal(),
		},
		{
			name:      "migration dir does not exist",
			giveCsKey: "invalid",
			wantErr:   "error finding files",
		},
		{
			name: "artifact does not exist",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.SaveChangesetOutput("0001_no_datastore", fdeployment.ChangesetOutput{})
				require.NoError(t, err)
			},
			giveCsKey: "0001_no_datastore",
			wantErr:   "no files found matching pattern",
		},
		{
			name: "address book is malformed JSON",
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()

				err := artsDir.CreateChangesetDir("0001_malformed")
				require.NoError(t, err)

				err = os.WriteFile(
					filepath.Join(artsDir.ChangesetDirPath("0001_malformed"), "xxx-ccip-staging-0001_malformed_datastore.json"),
					[]byte("malformed"),
					0600,
				)
				require.NoError(t, err)
			},
			giveCsKey: "0001_malformed",
			wantErr:   "failed to unmarshal JSON",
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

			got, err := artsDir.LoadDataStoreByChangesetKey(tt.giveCsKey)

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

	changesetKey := "0001_initial"

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
		giveCsKey   string
		want        []operations.Report[any, any]
		wantLoadErr string
		wantSaveErr string
	}{
		{
			name:      "success save and load",
			giveCsKey: changesetKey,
			want:      []operations.Report[any, any]{report},
		},
		{
			name:      "success save and load - durable pipelines",
			giveCsKey: changesetKey,
			beforeFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()
				err := artsDir.SetDurablePipelines("1749186682460987000")
				require.NoError(t, err)
			},
			want: []operations.Report[any, any]{report},
		},
		{
			name:      "report does not exist - return empty slice",
			giveCsKey: "invalid",
			want:      []operations.Report[any, any]{},
		}, {
			name:      "success - directory does not exist - should create it",
			giveCsKey: changesetKey,
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

			err := artsDir.SaveOperationsReports(changesetKey, []operations.Report[any, any]{report})
			if tt.wantSaveErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantSaveErr)
			} else {
				require.NoError(t, err)
			}

			got, err := artsDir.LoadOperationsReports(tt.giveCsKey)

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

	tests := []struct {
		name               string
		changesetKey       string
		setupFunc          func(t *testing.T, artsDir *ArtifactsDir)
		expectedFilePrefix string
		isDurablePipelines bool
	}{
		{
			name:               "regular migration proposals",
			changesetKey:       "0001_initial",
			expectedFilePrefix: "",
			isDurablePipelines: false,
		},
		{
			name:         "durable pipelines proposals with timestamp prefix",
			changesetKey: "0001_initial",
			setupFunc: func(t *testing.T, artsDir *ArtifactsDir) {
				t.Helper()
				err := artsDir.SetDurablePipelines("1234567890123456789")
				require.NoError(t, err)
			},
			expectedFilePrefix: "1234567890123456789-",
			isDurablePipelines: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixture := setupTestDomainsFS(t)
			artsDir := fixture.artifactsDir

			if tt.setupFunc != nil {
				tt.setupFunc(t, artsDir)
			}

			// Save multiple proposals
			require.NoError(t, artsDir.CreateChangesetDir(tt.changesetKey))
			for i, proposal := range proposals {
				err := artsDir.saveProposalArtifact(tt.changesetKey, ArtifactMCMSProposal, i, proposal)
				require.NoError(t, err)
				err = artsDir.saveDecodedProposalArtifact(tt.changesetKey, ArtifactMCMSProposal, i, "some decoded proposal")
				require.NoError(t, err)
			}

			// Verify proposal files were created with correct indexes and timestamp prefix
			proposalFiles, err := os.ReadDir(artsDir.ProposalsDirPath())
			require.NoError(t, err)
			sort.Slice(proposalFiles, func(i, j int) bool { return proposalFiles[i].Name() < proposalFiles[j].Name() })
			assert.Len(t, proposalFiles, len(proposals))

			// Verify decoded proposal files were created
			decodedFiles, err := os.ReadDir(artsDir.DecodedProposalsDirPath())
			require.NoError(t, err)
			sort.Slice(decodedFiles, func(i, j int) bool { return decodedFiles[i].Name() < decodedFiles[j].Name() })
			assert.Len(t, decodedFiles, len(proposals))

			// Check proposal file naming conventions
			for i, file := range proposalFiles {
				expectedFileName := fmt.Sprintf("%s%s-%s-%s_%s_%d.%s",
					tt.expectedFilePrefix,
					artsDir.DomainKey(),
					artsDir.EnvKey(),
					tt.changesetKey,
					ArtifactMCMSProposal,
					i,
					JSONExt)
				assert.Equal(t, expectedFileName, file.Name())
			}

			// Check decoded proposal file naming conventions
			for i, file := range decodedFiles {
				expectedFileName := fmt.Sprintf("%s%s-%s-%s_%s_%d_decoded.%s",
					tt.expectedFilePrefix,
					artsDir.DomainKey(),
					artsDir.EnvKey(),
					tt.changesetKey,
					ArtifactMCMSProposal,
					i,
					TxtExt)
				assert.Equal(t, expectedFileName, file.Name())
			}

			// Load proposals and verify
			exists, err := artsDir.ChangesetDirExists(tt.changesetKey)
			require.NoError(t, err)
			require.True(t, exists)
			loadedProposals, err := artsDir.LoadChangesetOutput(tt.changesetKey)
			require.NoError(t, err)
			assert.Len(t, loadedProposals.MCMSProposals, len(proposals))

			for i, proposal := range loadedProposals.MCMSProposals {
				assert.Equal(t, proposals[i].Version, proposal.Version)
				assert.Equal(t, proposals[i].ValidUntil, proposal.ValidUntil)
			}
		})
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

	csKey := "0001_sorted_addresses"

	// Create address book with intentionally unsorted entries
	// Higher chain selector first, unsorted addresses
	addrBook := fdeployment.NewMemoryAddressBookFromMap(map[uint64]map[string]fdeployment.TypeAndVersion{
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
	err = artsDir.SaveChangesetOutput(csKey, fdeployment.ChangesetOutput{
		AddressBook: addrBook,
	})
	require.NoError(t, err)

	// Find the address book file
	pattern := "*_" + ArtifactAddress + "." + JSONExt
	addrBookPath, err := artsDir.findArtifactPath(artsDir.ChangesetDirPath(csKey), pattern)
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
	loadedOutput, err := artsDir.LoadChangesetOutput(csKey)
	require.NoError(t, err)
	//nolint:staticcheck
	require.NotNil(t, loadedOutput.AddressBook)
}

func Test_getOperationsReportsFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		csKey              string
		wantFileName       string
		isDurablePipelines bool
	}{
		{
			name:         "migration",
			csKey:        "0001_initial",
			wantFileName: "0001_initial-reports.json",
		},
		{
			name:               "durable pipelines",
			csKey:              "0001_initial",
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

			got := artsDir.getOperationsReportsFilePath(tt.csKey)
			assert.Equal(t, filepath.Join(artsDir.OperationsReportsDirPath(), tt.wantFileName), got)
		})
	}
}
