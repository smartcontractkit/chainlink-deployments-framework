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

	// Create store with a unique environment name per test to ensure isolation
	store := NewCatalogEnvMetadataStore(CatalogEnvMetadataStoreConfig{
		Domain:      "test-domain",
		Environment: "catalog_testing", // Use static environment name
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

// createMergeUpdater creates a custom updater that merges TestEnvMetadata fields intelligently
func tagsAppendMerger() datastore.MetadataUpdaterF {
	return func(latest any, incoming any) (any, error) {
		// Convert latest to TestEnvMetadata
		latestMeta, err := datastore.As[TestEnvMetadata](latest)
		if err != nil {
			return nil, err
		}
		// Convert incoming to TestEnvMetadata
		incomingMeta, err := datastore.As[TestEnvMetadata](incoming)
		if err != nil {
			return nil, err
		}

		// append tags from incoming to latest
		latestMeta.Tags = append(latestMeta.Tags, incomingMeta.Tags...)

		return latestMeta, nil
	}
}

func TestCatalogEnvMetadataStore_Get(t *testing.T) {
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
			// Create store for testing
			store, conn := setupTestEnvStore(t)
			defer conn.Close()

			// Setup test data if needed
			for _, record := range tt.setupRecords {
				err := store.Set(context.Background(), record.Metadata)
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

func TestCatalogEnvMetadataStore_Set(t *testing.T) {
	tests := []struct {
		name    string
		record  datastore.EnvMetadata
		wantErr error
	}{
		{
			name: "success",
			record: datastore.EnvMetadata{
				Metadata: TestEnvMetadata{
					Description: "Test environment",
					Version:     "1.0.0",
				},
			},
			wantErr: nil,
		},
		{
			name: "nil_metadata",
			record: datastore.EnvMetadata{
				Metadata: nil,
			},
			wantErr: nil,
		},
		{
			name: "complex_metadata",
			record: datastore.EnvMetadata{
				Metadata: TestEnvMetadata{
					Description: "Production environment",
					Version:     "2.1.0",
					Tags:        []string{"production", "v1.0"},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, conn := setupTestEnvStore(t)
			defer conn.Close()

			// Test Set operation
			err := store.Set(context.Background(), tt.record.Metadata)

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

			// Verify record was set by retrieving it
			gotRecord, err := store.Get()
			require.NoError(t, err, "Failed to retrieve record after Set")

			requireEnvMetadataEqual(t, tt.record, gotRecord)
		})
	}
}

func TestCatalogEnvMetadataStore_Set_Update(t *testing.T) {
	store, conn := setupTestEnvStore(t)
	defer conn.Close()

	// Set initial record
	initialMetadata := TestEnvMetadata{
		Description: "Initial environment",
		Version:     "2.0.0",
		Tags:        []string{"initial"},
	}
	err := store.Set(context.Background(), initialMetadata)
	require.NoError(t, err, "Failed to set initial record")

	// Create a custom updater that merges tags and updates version
	mergeUpdater := tagsAppendMerger()

	// Update the record using the custom updater
	updateMetadata := TestEnvMetadata{
		Tags: []string{"updated", "v2"},
	}
	err = store.Set(context.Background(), updateMetadata, mergeUpdater)
	require.NoError(t, err, "Failed to update record with custom updater")

	// Verify the record was updated with merged data
	gotRecord, err := store.Get()
	require.NoError(t, err, "Failed to retrieve updated record")

	// Expected result should have:
	// - Description from initial record (preserved)
	// - Version from update (2.0.0)
	// - Tags merged from both (initial, updated, v2)
	expectedMeta := TestEnvMetadata{
		Description: "Initial environment",                // Preserved from original
		Version:     "2.0.0",                              // Updated
		Tags:        []string{"initial", "updated", "v2"}, // Merged (order may vary)
	}

	// Check individual fields since tag order might vary
	actualMeta, err := datastore.As[TestEnvMetadata](gotRecord.Metadata)
	require.NoError(t, err, "Failed to convert metadata")

	require.Equal(t, expectedMeta.Description, actualMeta.Description, "Description should be preserved")
	require.Equal(t, expectedMeta.Version, actualMeta.Version, "Version should be updated")

	// Check that all expected tags are present (order doesn't matter)
	require.Len(t, actualMeta.Tags, len(expectedMeta.Tags), "Should have correct number of tags")
	for _, expectedTag := range expectedMeta.Tags {
		require.Contains(t, actualMeta.Tags, expectedTag, "Should contain expected tag: %s", expectedTag)
	}
}

func TestCatalogEnvMetadataStore_Set_ConcurrentUpdates(t *testing.T) {
	// Create two stores pointing to the same environment
	store1, conn1 := setupTestEnvStore(t)
	defer conn1.Close()

	store2, conn2 := setupTestEnvStore(t)
	defer conn2.Close()

	// Set initial record with store1
	initialMetadata := TestEnvMetadata{
		Description: "Initial environment",
		Version:     "1.0.0",
	}
	err := store1.Set(context.Background(), initialMetadata)
	require.NoError(t, err, "Failed to set initial record")

	// Both stores get the record to sync their version caches
	_, err = store1.Get()
	require.NoError(t, err, "Store1 failed to get record after initial set")

	_, err = store2.Get()
	require.NoError(t, err, "Store2 failed to get record")

	// Store1 updates the record (this should succeed and increment server version)
	updatedMetadata1 := TestEnvMetadata{
		Description: "Updated by store1",
		Version:     "2.0.0",
	}
	err = store1.Set(context.Background(), updatedMetadata1)
	require.NoError(t, err, "Store1 failed to update record")

	// Store2 also updates successfully (both should succeed with UPSERT semantics)
	updatedMetadata2 := TestEnvMetadata{
		Description: "Updated by store2",
		Version:     "2.1.0",
	}
	err = store2.Set(context.Background(), updatedMetadata2)
	require.NoError(t, err, "Store2 failed to update record")

	// Verify the record has store2's update (the last one wins)
	finalRecord, err := store2.Get()
	require.NoError(t, err, "Failed to get final record")

	expectedRecord := datastore.EnvMetadata{
		Metadata: updatedMetadata2,
	}
	requireEnvMetadataEqual(t, expectedRecord, finalRecord)
}

func TestCatalogEnvMetadataStore_ConversionHelpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T, store *CatalogEnvMetadataStore)
	}{
		{
			name: "keyToFilter",
			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
				t.Helper()
				filter := store.keyToFilter()
				require.NotNil(t, filter, "keyToFilter returned nil")
				require.Equal(t, "test-domain", filter.Domain.Value, "Expected domain 'test-domain'")
				require.Equal(t, "catalog_testing", filter.Environment.Value, "Expected environment 'catalog_testing'")
			},
		},
		{
			name: "protoToEnvMetadata_success",
			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
				t.Helper()
				protoRecord := &pb.EnvironmentMetadata{
					Domain:      "test-domain",
					Environment: "test-env",
					Metadata:    `{"description":"test env","version":"1.0","extra":{"key":"value"}}`,
					RowVersion:  1,
				}

				result, err := store.protoToEnvMetadata(protoRecord)
				require.NoError(t, err, "protoToEnvMetadata failed")

				// Since JSON unmarshaling returns map[string]interface{}, we need to assert on that
				metadata := result.Metadata.(map[string]interface{})
				require.Equal(t, "test env", metadata["description"])
				require.Equal(t, "1.0", metadata["version"])
				require.IsType(t, map[string]interface{}{}, metadata["extra"])
			},
		},
		{
			name: "protoToEnvMetadata_invalid_json",
			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
				t.Helper()
				protoRecord := &pb.EnvironmentMetadata{
					Domain:      "test-domain",
					Environment: "test-env",
					Metadata:    `{invalid json}`,
					RowVersion:  1,
				}

				_, err := store.protoToEnvMetadata(protoRecord)
				require.Error(t, err, "Expected error for invalid JSON")
			},
		},
		{
			name: "envMetadataToProto",
			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
				t.Helper()
				record := datastore.EnvMetadata{
					Metadata: TestEnvMetadata{
						Description: "test environment",
						Version:     "1.0.0",
						Tags:        []string{"production", "testing"},
					},
				}

				result := store.envMetadataToProto(record, 5)

				require.Equal(t, "test-domain", result.Domain, "Expected domain 'test-domain'")
				require.Equal(t, "catalog_testing", result.Environment, "Expected environment 'catalog_testing'")
				require.Equal(t, int32(5), result.RowVersion, "Expected version 5")

				// Parse the JSON metadata to verify it's correct
				var metadata map[string]interface{}
				err := json.Unmarshal([]byte(result.Metadata), &metadata)
				require.NoError(t, err, "Failed to parse result metadata JSON")

				require.Equal(t, "test environment", metadata["description"], "Expected description 'test environment'")
				require.Equal(t, "1.0.0", metadata["version"], "Expected version '1.0.0'")
				require.IsType(t, []interface{}{}, metadata["tags"], "Expected tags to be an array")
			},
		},
		{
			name: "version_handling",
			test: func(t *testing.T, store *CatalogEnvMetadataStore) {
				t.Helper()
				// Test getVersion and setVersion
				initialVersion := store.getVersion()
				require.Equal(t, int32(0), initialVersion, "Expected initial version 0")

				store.setVersion(10)
				newVersion := store.getVersion()
				require.Equal(t, int32(10), newVersion, "Expected version 10 after setVersion")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh store for each test case to avoid concurrency issues
			store, conn := setupTestEnvStore(t)
			defer conn.Close()

			tt.test(t, store)
		})
	}
}
