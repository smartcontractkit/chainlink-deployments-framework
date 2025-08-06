package provider_test

import (
	"testing"

	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
)

// TestCTFOffchainProvider_WithLocalImage demonstrates how to use the CTF provider
// with a JD Docker image from AWS ECR.

// To authenticate with ECR before running:
// 1. AWS CLI configured with access to ECR: use aws sso login to AWS Account which has the ECR repository containing the JD image.
// 2. aws ecr get-login-password --region <region> | docker login --username AWS --password-stdin <ECR_REPO_URL>
//
// Alternatively, you can build the JD image locally and use it directly.
// Run this test with: go test -v -run TestCTFOffchainProvider_WithLocalImage
func TestCTFOffchainProvider_WithLocalImage(t *testing.T) {
	t.Parallel()

	// Skip by default - uncomment the next line to run integration tests
	t.Skip("Integration test with ECR Docker image - enable manually")

	config := provider.CTFOffchainProviderConfig{
		// Use ECR image from private AWS account
		Image: "<IMAGE_TAG>", // UPDATE THIS

		// Use default ports (optional to specify)
		GRPCPort:  "14231",
		WSRPCPort: "8080",

		// Valid 64-character hex CSA encryption key
		CSAEncryptionKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",

		// Optional: Custom database configuration
		// DBInput: &postgres.Input{
		//     Image: "postgres:16",
		//     Port:  15432, // Use different port to avoid conflicts
		//     Name:  "jd-test-db",
		// },
	}

	// Create the CTF provider
	jdProvider := provider.NewCTFOffchainProvider(t, config)

	// Initialize the provider (this starts Docker containers)
	t.Log("Initializing JD CTF provider with ECR image...")
	jdClient, err := jdProvider.Initialize(t.Context())
	require.NoError(t, err, "Failed to initialize JD CTF provider")
	require.NotNil(t, jdClient, "JD client should not be nil")

	t.Log("JD CTF provider initialized successfully - health check passed")

	// Test 1: Node Service - List nodes
	t.Run("Node_ListNodes", func(t *testing.T) {
		t.Parallel()

		nodes, err := jdClient.ListNodes(t.Context(), &nodev1.ListNodesRequest{})
		require.NoError(t, err, "Failed to list nodes")
		require.NotNil(t, nodes, "Nodes response should not be nil")

		t.Logf("Found %d nodes", len(nodes.Nodes))

		// Log node information if any exist
		for i, node := range nodes.Nodes {
			t.Logf("Node %d: ID=%s", i, node.Id)
		}
	})

	// Test 2: Job Service - List jobs
	t.Run("Job_ListJobs", func(t *testing.T) {
		t.Parallel()

		jobs, err := jdClient.ListJobs(t.Context(), &jobv1.ListJobsRequest{})
		require.NoError(t, err, "Failed to list jobs")
		require.NotNil(t, jobs, "Jobs response should not be nil")

		t.Logf("Found %d jobs", len(jobs.Jobs))

		// Log job information if any exist
		for i, job := range jobs.Jobs {
			t.Logf("Job %d: ID=%s", i, job.Id)
		}
	})

	t.Log("All JD integration tests completed successfully")
}
