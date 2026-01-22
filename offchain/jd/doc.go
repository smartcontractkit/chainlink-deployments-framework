/*
Package jd provides a comprehensive framework for interacting with Job Distributor (JD) services
in the Chainlink Deployments Framework ecosystem.

# Overview

The JD package enables seamless integration with
Job Distributor services through gRPC communication. It supports multiple authentication
mechanisms and provides a unified interface for job management operations.

# Architecture

The package consists of three main components:

1. JD Client (client.go) - Core gRPC client implementation
2. Client Provider (provider/client_provider.go) - Connects to existing JD services
3. CTF Provider (provider/ctf_provider.go) - Creates and manages JD Docker containers for testing

# Basic Usage

# Simple Connection

For basic connectivity without authentication:

	import (
		"context"
		"google.golang.org/grpc/credentials/insecure"
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
	)

	config := jd.JDConfig{
		GRPC: "localhost:9090",
	}

	client, err := jd.NewJDClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Use client for operations
	import csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	keypairs, err := client.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})

# Provider Interface

	import (
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
	)

	providerConfig := provider.ClientOffchainProviderConfig{
		GRPC:  "localhost:9090",
		Creds: insecure.NewCredentials(),
	}

	prov, err := provider.NewClientOffchainProvider(providerConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize with environment
	err = prov.Initialize(env)
	if err != nil {
		log.Fatal(err)
	}

	// Get client
	client, err := prov.OffchainClient()

	// client is compatible with the Offchain field in the Environment struct

# CTF Provider (Testing)

For testing scenarios where you need to spin up JD Docker containers:

	import (
		"testing"
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
	)

	func TestWithJD(t *testing.T) {
		config := provider.CTFOffchainProviderConfig{
			Image: "job-distributor:latest",
			// OR use environment variable: CTF_JD_IMAGE

			// Optional PostgreSQL configuration
			PostgresPort:      5432,
			PostgresHost:      "localhost",
			PostgresUser:      "chainlink",
			PostgresPassword:  "chainlink",
			PostgresDBName:    "chainlink",

			// Optional JD configuration
			GRPCPort:           14231,
			WebSocketRPCPort:   8080,
			CSAEncryptionKey:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			JDSQLDumpPath:      "./migrations.sql", // Optional
		}

		// Create CTF provider
		jdProvider := provider.NewCTFOffchainProvider(t, config)

		// Initialize (starts Docker containers with health check)
		ctx := context.Background()
		client, err := jdProvider.Initialize(ctx)
		if err != nil {
			t.Fatalf("Failed to initialize JD: %v", err)
		}

		// Use client for testing
		keypairs, err := client.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
		if err != nil {
			t.Fatalf("Failed to list keypairs: %v", err)
		}

		t.Logf("JD service ready with %d keypairs", len(keypairs.Keypairs))
	}

The CTF provider automatically:
- Starts PostgreSQL container with proper schema
- Starts JD container with correct configuration
- Performs health checks using retry logic
- Cleans up containers when tests complete

# Environment Variable Configuration

You can specify the JD Docker image via environment variable:

	export CTF_JD_IMAGE=localhost:5001/job-distributor:latest

	config := provider.CTFOffchainProviderConfig{
		// Image field can be omitted when CTF_JD_IMAGE is set
	}

# Health Check

The CTF provider includes built-in health checking that retries `GetKeypair` calls:
- 10 retry attempts with 2-second delays
- Ensures JD service is fully ready before returning

# Authentication

The package supports three authentication mechanisms:

# 1. No Authentication

For development or internal networks:

	config := jd.JDConfig{
		GRPC:  "localhost:9090",
		Creds: insecure.NewCredentials(),
	}

# 2. OAuth2 Authentication

For services requiring OAuth2 Bearer tokens:

	import "golang.org/x/oauth2"

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "your-access-token",
		TokenType:   "Bearer",
	})

	config := jd.JDConfig{
		GRPC:  "secure.jobdistributor.com:443",
		Auth:  tokenSource,
		Creds: credentials.NewTLS(&tls.Config{}),
	}

# Client Operations

The JD client supports various operations through gRPC service interfaces:

# CSA Key Management

	import csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"

	// List CSA keypairs
	keypairs, err := client.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
	if err != nil {
		log.Printf("Failed to list CSA keypairs: %v", err)
	}

	for _, keypair := range keypairs.Keypairs {
		log.Printf("CSA Public Key: %s", keypair.PublicKey)
	}

# Job Management

	import jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"

	// List existing jobs
	jobs, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{})
	if err != nil {
		log.Printf("Failed to list jobs: %v", err)
	}

	for _, job := range jobs.Jobs {
		log.Printf("Job ID: %s", job.Id)
	}

	// Propose a new job
	jobSpec := &jobv1.ProposeJobRequest{
		NodeIds: []string{"node-1", "node-2"},
		Spec:    "job specification here",
	}

	response, err := client.ProposeJob(ctx, jobSpec)
	if err != nil {
		log.Printf("Failed to propose job: %v", err)
	}

# Node Management

	import nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	// List registered nodes
	nodes, err := client.ListNodes(ctx, &nodev1.ListNodesRequest{})
	if err != nil {
		log.Printf("Failed to list nodes: %v", err)
	}

	for _, node := range nodes.Nodes {
		log.Printf("Node ID: %s", node.Id)
	}

# Configuration Validation

The provider automatically validates configurations:

	config := provider.ClientOffchainProviderConfig{
		GRPC: "", // Invalid - will cause validation error
	}

	prov, err := provider.NewClientOffchainProvider(config)
	if err != nil {
		// Handle validation error
		log.Printf("Invalid configuration: %v", err)
	}

# Dry Run Mode

The package includes a dry run client that provides safe testing capabilities without
affecting real Job Distributor operations:

	import (
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
		"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
	)

	// Create a real client for read operations
	realClient, err := jd.NewJDClient(jd.JDConfig{
		GRPC: "localhost:9090",
		Creds: insecure.NewCredentials(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Wrap with dry run client
	dryRunClient := jd.NewDryRunJobDistributor(realClient, logger.DefaultLogger)

	// Read operations work normally (forwarded to real backend)
	jobs, err := dryRunClient.ListJobs(ctx, &jobv1.ListJobsRequest{})

	// Write operations are simulated (logged but not executed)
	response, err := dryRunClient.ProposeJob(ctx, &jobv1.ProposeJobRequest{
		NodeId: "test-node",
		Spec:   "test job spec",
	})
	// Returns mock response without actually proposing the job

# Dry Run with Provider (Recommended)

For a cleaner approach, use the provider's functional option:

	import (
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
		"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
	)

	// Create provider with dry run mode enabled
	jdProvider := provider.NewClientOffchainProvider(
		provider.ClientOffchainProviderConfig{
			GRPC: "localhost:9090",
			Creds: insecure.NewCredentials(),
		},
		provider.WithDryRun(logger),
	)

	// Initialize - returns a dry run client automatically
	client, err := jdProvider.Initialize(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// All operations now use dry run mode
	jobs, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{})        // Read: forwarded
	response, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{  // Write: simulated
		NodeId: "test-node",
		Spec:   "test job spec",
	})
*/
package jd
