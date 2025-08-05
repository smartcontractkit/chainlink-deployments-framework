/*
Package jd provides a comprehensive framework for interacting with Job Distributor (JD) services
in the Chainlink Deployments Framework ecosystem.

# Overview

The JD package enables seamless integration with
Job Distributor services through gRPC communication. It supports multiple authentication
mechanisms and provides a unified interface for job management operations.

# Architecture

The package consists of two main components:

1. **JD Client** (`client.go`) - Core gRPC client implementation
2. **Provider Interface** (`provider/`) - A wrapper around the JD client that provides a standarized interface

# Basic Usage

## Simple Connection

For basic connectivity without authentication:

	import (
		"context"
		"google.golang.org/grpc/credentials/insecure"
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
	)

	config := jd.JDConfig{
		GRPC:  "localhost:9090",
		WSRPC: "ws://localhost:9091"
	}

	client, err := jd.NewJDClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Use client for operations
	pubKey, err := client.GetCSAPublicKey(ctx)

## Provider Interface

	import (
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
	)

	providerConfig := provider.ClientOffchainProviderConfig{
		GRPC:  "localhost:9090",
		WSRPC: "ws://localhost:9091",
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

# Authentication

The package supports three authentication mechanisms:

## 1. No Authentication

For development or internal networks:

	config := jd.JDConfig{
		GRPC:  "localhost:9090",
		Creds: insecure.NewCredentials(),
	}

## 2. OAuth2 Authentication

For services requiring OAuth2 Bearer tokens:

	import "golang.org/x/oauth2"

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: "your-access-token",
		TokenType:   "Bearer",
	})

	config := jd.JDConfig{
		GRPC:        "secure.jobdistributor.com:443",
		WSRPC:       "wss://secure.jobdistributor.com:443/ws",
		OAuth2:      tokenSource,
		Creds:       credentials.NewTLS(&tls.Config{}),
	}

# Client Operations

The JD client supports various operations:

## CSA Key Management

	// Get CSA public key
	pubKey, err := client.GetCSAPublicKey(ctx)
	if err != nil {
		log.Printf("Failed to get CSA key: %v", err)
	}

## Job Management

	import "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"

	jobSpec := &job.ProposeJobRequest{
		NodeIds: []string{"node-1", "node-2"},
		Spec:    "job specification here",
	}

	response, err := client.ProposeJob(ctx, jobSpec)
	if err != nil {
		log.Printf("Failed to propose job: %v", err)
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
*/
package jd
