/*
Package offchain provides client abstractions and provider interfaces for interacting with
off-chain services in the Chainlink Deployments Framework ecosystem.

# Overview

The offchain package serves as the main entry point for managing off-chain client connections
and providers. It provides unified interfaces for connecting to various off-chain services
such as Job Distributors (JD) and other Chainlink infrastructure components.

# Architecture

The package consists of two main components:

 1. Client (offchain.go) - Client interface for off-chain service interactions
 2. Provider Interface (provider.go) - Abstraction for different off-chain service providers

# Basic Usage

# Provider Interface

The Provider interface allows for pluggable off-chain service implementations:

	import (
		"context"
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	)

	// Example provider usage
	func connectToOffchainService(provider offchain.Provider) error {
		ctx := context.Background()

		// Initialize the provider
		client, err := provider.Initialize(ctx)
		if err != nil {
			return err
		}

		// Get provider name for logging
		providerName := provider.Name()
		log.Printf("Connected to provider: %s", providerName)

		// Use the client for operations
		offchainClient := provider.OffchainClient()
		// ... perform operations with offchainClient

		return nil
	}

# Available Providers

The offchain package includes several specialized providers:

# Job Distributor (JD) Provider

Located in the jd subpackage, providing:

  - Client Provider - Connects to existing JD services

  - CTF Provider - Creates JD Docker containers for testing

# Provider Implementation

To implement a custom off-chain provider, satisfy the Provider interface:

	type CustomProvider struct {
		client offchain.Client
		name   string
	}

	func (p *CustomProvider) Initialize(ctx context.Context) (offchain.Client, error) {
		// Initialize your custom client
		// ... setup logic
		return p.client, nil
	}

	func (p *CustomProvider) Name() string {
		return p.name
	}

	func (p *CustomProvider) OffchainClient() offchain.Client {
		return p.client
	}

# Integration with Deployment Framework

The offchain package integrates seamlessly with the deployment framework:

	import (
		"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
		"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	)

	// Create environment with offchain client
	env := &deployment.Environment{
		Offchain: offchainClient, // offchain.Client is compatible
		// ... other environment fields
	}

This compatibility ensures that existing deployment workflows can leverage
the new offchain provider abstractions without breaking changes.
*/
package offchain
