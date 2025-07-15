package catalog

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/internal/protos"
)

const (
	// Default gRPC server address - can be overridden with CATALOG_GRPC_ADDRESS env var
	defaultEnvGRPCAddress = "localhost:8080"
)

// TestEnvMetadata is a concrete type for testing environment metadata
type TestEnvMetadata struct {
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	UUID        string   `json:"uuid,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// setupTestEnvStore creates a test environment metadata store and gRPC connection
func setupTestEnvStore(t *testing.T) (*CatalogEnvMetadataStore, *grpc.ClientConn) {
	t.Helper()
	address := os.Getenv("CATALOG_GRPC_ADDRESS")
	if address == "" {
		address = defaultEnvGRPCAddress
	}

	// Create connection
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("Failed to connect to gRPC server at %s: %v. Skipping integration tests.", address, err)
		return nil, nil
	}

	// Create client
	client := pb.NewDeploymentsDatastoreClient(conn)

	// Test if the gRPC service is actually available by making a simple call
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stream, err := client.DataAccess(ctx)
	if err != nil {
		conn.Close()
		t.Skipf("gRPC service not available at %s: %v. Skipping integration tests.", address, err)

		return nil, nil
	}
	if stream != nil {
		_ = stream.CloseSend()
	}

	// Create store with the standard testing environment name
	store := NewCatalogEnvMetadataStore(CatalogEnvMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing",
		Client:      client,
	})

	return store, conn
}

// requireEnvMetadataEqual compares two EnvMetadata records for equality
// This handles the fact that JSON serialization/deserialization converts concrete types to map[string]interface{}
func requireEnvMetadataEqual(t *testing.T, expected, actual datastore.EnvMetadata) {
	t.Helper()
	// Convert both to JSON and compare to handle type differences from JSON round-trip
	expectedJSON, err := json.Marshal(expected.Metadata)
	require.NoError(t, err, "Failed to marshal expected metadata")

	actualJSON, err := json.Marshal(actual.Metadata)
	require.NoError(t, err, "Failed to marshal actual metadata")

	require.JSONEq(t, string(expectedJSON), string(actualJSON), "EnvMetadata should be equal")
}

func TestCatalogEnvMetadataStore_Get(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		setupRecords []datastore.EnvMetadata
		wantRecord   datastore.EnvMetadata
		wantErr      error
	}{
		{
			name: "success",
			setupRecords: []datastore.EnvMetadata{
				{
					Metadata: TestEnvMetadata{
						Description: "Test environment",
						Version:     "1.0.0",
					},
				},
			},
			wantRecord: datastore.EnvMetadata{
				Metadata: TestEnvMetadata{
					Description: "Test environment",
					Version:     "1.0.0",
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create store for testing
			store, conn := setupTestEnvStore(t)
			defer conn.Close()

			// Setup test data if needed
			for _, record := range tt.setupRecords {
				err := store.Set(record)
				require.NoError(t, err, "Failed to setup record")
			}

			// Test Get operation
			gotRecord, err := store.Get()

			// Verify error
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErr)
					return
				}
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
					return
				}

				return
			}

			require.NoError(t, err, "Unexpected error")

			// Verify record
			requireEnvMetadataEqual(t, tt.wantRecord, gotRecord)
		})
	}
}

// func TestCatalogEnvMetadataStore_Set(t *testing.T) {
// 	t.Parallel()
// 	tests := []struct {
// 		name    string
// 		record  datastore.EnvMetadata
// 		wantErr error
// 	}{
// 		{
// 			name: "success",
// 			record: datastore.EnvMetadata{
// 				Metadata: TestEnvMetadata{
// 					Description: "Test environment",
// 					Version:     "1.0.0",
// 				},
// 			},
// 			wantErr: nil,
// 		},
// 		{
// 			name: "nil_metadata",
// 			record: datastore.EnvMetadata{
// 				Metadata: nil,
// 			},
// 			wantErr: nil,
// 		},
// 		{
// 			name: "complex_metadata",
// 			record: datastore.EnvMetadata{
// 				Metadata: TestEnvMetadata{
// 					Description: "Production environment",
// 					Version:     "2.1.0",
// 					Tags:        []string{"production", "v1.0"},
// 				},
// 			},
// 			wantErr: nil,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			store, conn := setupTestEnvStore(t)
// 			defer conn.Close()

// 			// Test Set operation
// 			err := store.Set(tt.record)

// 			// Verify error
// 			if tt.wantErr != nil {
// 				if err == nil {
// 					t.Errorf("Expected error %v, got nil", tt.wantErr)
// 					return
// 				}
// 				if err.Error() != tt.wantErr.Error() {
// 					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
// 					return
// 				}

// 				return
// 			}

// 			require.NoError(t, err, "Unexpected error")

// 			// Verify record was set by retrieving it
// 			gotRecord, err := store.Get()
// 			require.NoError(t, err, "Failed to retrieve record after Set")

// 			requireEnvMetadataEqual(t, tt.record, gotRecord)
// 		})
// 	}
// }

// func TestCatalogEnvMetadataStore_Set_Update(t *testing.T) {
// 	t.Parallel()
// 	store, conn := setupTestEnvStore(t)
// 	defer conn.Close()

// 	// Set initial record
// 	initialRecord := datastore.EnvMetadata{
// 		Metadata: TestEnvMetadata{
// 			Version: "1.0.0",
// 		},
// 	}
// 	err := store.Set(initialRecord)
// 	require.NoError(t, err, "Failed to set initial record")

// 	// Update the record
// 	updatedRecord := datastore.EnvMetadata{
// 		Metadata: TestEnvMetadata{
// 			Version:     "2.0.0",
// 			Description: "Updated environment",
// 		},
// 	}
// 	err = store.Set(updatedRecord)
// 	require.NoError(t, err, "Failed to update record")

// 	// Verify the record was updated
// 	gotRecord, err := store.Get()
// 	require.NoError(t, err, "Failed to retrieve updated record")

// 	requireEnvMetadataEqual(t, updatedRecord, gotRecord)
// }

// func TestCatalogEnvMetadataStore_Set_StaleVersion(t *testing.T) {
// 	t.Parallel()
// 	t.Skip("Stale version testing for environment metadata requires more complex setup due to singleton nature")
// 	// Environment metadata is a singleton per domain/environment, so testing
// 	// stale version scenarios is more complex than with keyed records.
// 	store1, conn1 := setupTestEnvStore(t)
// 	defer conn1.Close()

// 	// Set initial record with store1
// 	initialRecord := datastore.EnvMetadata{
// 		Metadata: TestEnvMetadata{
// 			Version: "1.0.0",
// 		},
// 	}
// 	err := store1.Set(initialRecord)
// 	require.NoError(t, err, "Failed to set initial record")

// 	// Get the record to sync the version cache
// 	_, err = store1.Get()
// 	require.NoError(t, err, "Store1 failed to get record after initial set")

// 	// Create a second store pointing to the same environment
// 	store2, conn2 := setupTestEnvStore(t)
// 	defer conn2.Close()

// 	// Store2 should be able to read the existing record
// 	_, err = store2.Get()
// 	require.NoError(t, err, "Store2 failed to get record")

// 	// Store1 updates the record (this should succeed and increment server version)
// 	updatedRecord1 := datastore.EnvMetadata{
// 		Metadata: TestEnvMetadata{
// 			Version:     "2.0.0",
// 			Description: "Updated by store1",
// 		},
// 	}
// 	err = store1.Set(updatedRecord1)
// 	require.NoError(t, err, "Store1 failed to update record")

// 	// Store2 tries to update with its cached (now stale) version - this should fail
// 	updatedRecord2 := datastore.EnvMetadata{
// 		Metadata: TestEnvMetadata{
// 			Version:     "2.1.0",
// 			Description: "Updated by store2",
// 		},
// 	}
// 	err = store2.Set(updatedRecord2)
// 	if err == nil {
// 		t.Errorf("Expected stale version error, but Set succeeded")
// 		return
// 	}

// 	// Verify it's the expected stale error
// 	require.Equal(t, datastore.ErrEnvMetadataStale, err, "Expected ErrEnvMetadataStale")

// 	// Verify the record still has store1's update
// 	finalRecord, err := store1.Get()
// 	require.NoError(t, err, "Failed to get final record")

// 	requireEnvMetadataEqual(t, updatedRecord1, finalRecord)
// }

// func TestCatalogEnvMetadataStore_ConversionHelpers(t *testing.T) {
// 	t.Parallel()

// 	tests := []struct {
// 		name string
// 		test func(t *testing.T, store *CatalogEnvMetadataStore)
// 	}{
// 		{
// 			name: "keyToFilter",
// 			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
// 				t.Helper()
// 				filter := store.keyToFilter()
// 				require.NotNil(t, filter, "keyToFilter returned nil")
// 				require.Equal(t, "test-domain", filter.Domain.Value, "Expected domain 'test-domain'")
// 				require.Equal(t, "catalog_testing", filter.Environment.Value, "Expected environment 'catalog_testing'")
// 			},
// 		},
// 		{
// 			name: "protoToEnvMetadata_success",
// 			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
// 				t.Helper()
// 				protoRecord := &pb.EnvironmentMetadata{
// 					Domain:      "test-domain",
// 					Environment: "test-env",
// 					Metadata:    `{"description":"test env","version":"1.0","extra":{"key":"value"}}`,
// 					RowVersion:  1,
// 				}

// 				result, err := store.protoToEnvMetadata(protoRecord)
// 				require.NoError(t, err, "protoToEnvMetadata failed")

// 				// Since JSON unmarshaling returns map[string]interface{}, we need to assert on that
// 				metadata := result.Metadata.(map[string]interface{})
// 				require.Equal(t, "test env", metadata["description"])
// 				require.Equal(t, "1.0", metadata["version"])
// 				require.IsType(t, map[string]interface{}{}, metadata["extra"])
// 			},
// 		},
// 		{
// 			name: "protoToEnvMetadata_invalid_json",
// 			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
// 				t.Helper()
// 				protoRecord := &pb.EnvironmentMetadata{
// 					Domain:      "test-domain",
// 					Environment: "test-env",
// 					Metadata:    `{invalid json}`,
// 					RowVersion:  1,
// 				}

// 				_, err := store.protoToEnvMetadata(protoRecord)
// 				require.Error(t, err, "Expected error for invalid JSON")
// 			},
// 		},
// 		{
// 			name: "envMetadataToProto",
// 			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
// 				t.Helper()
// 				record := datastore.EnvMetadata{
// 					Metadata: TestEnvMetadata{
// 						Description: "test environment",
// 						Version:     "1.0.0",
// 						Tags:        []string{"production", "testing"},
// 					},
// 				}

// 				result := store.envMetadataToProto(record, 5)

// 				require.Equal(t, "test-domain", result.Domain, "Expected domain 'test-domain'")
// 				require.Equal(t, "catalog_testing", result.Environment, "Expected environment 'catalog_testing'")
// 				require.Equal(t, int32(5), result.RowVersion, "Expected version 5")

// 				// Parse the JSON metadata to verify it's correct
// 				var metadata map[string]interface{}
// 				err := json.Unmarshal([]byte(result.Metadata), &metadata)
// 				require.NoError(t, err, "Failed to parse result metadata JSON")

// 				require.Equal(t, "test environment", metadata["description"], "Expected description 'test environment'")
// 				require.Equal(t, "1.0.0", metadata["version"], "Expected version '1.0.0'")
// 				require.IsType(t, []interface{}{}, metadata["tags"], "Expected tags to be an array")
// 			},
// 		},
// 		{
// 			name: "version_handling",
// 			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
// 				t.Helper()
// 				// Test getVersion and setVersion
// 				initialVersion := store.getVersion()
// 				require.Equal(t, int32(0), initialVersion, "Expected initial version 0")

// 				store.setVersion(10)
// 				newVersion := store.getVersion()
// 				require.Equal(t, int32(10), newVersion, "Expected version 10 after setVersion")
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			// Create a fresh store for each test case to avoid concurrency issues
// 			store, conn := setupTestEnvStore(t)
// 			defer conn.Close()

// 			tt.test(t, store)
// 		})
// 	}
// }
