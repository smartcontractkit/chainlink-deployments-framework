/*
Package chain provides the core blockchain abstraction layer for the Chainlink Deployments Framework,
supporting multiple blockchain families through a unified interface.

# Overview

The chain package serves as the foundation for blockchain interactions in the Chainlink Deployments Framework.
It defines the core BlockChain interface that all blockchain implementations must satisfy, and provides
a powerful collection system for managing and querying multiple chains across different blockchain families.

# Architecture

The package consists of several key components:

 1. BlockChain Interface (blockchain.go) - Core abstraction for all blockchains
 2. BlockChains Collection (blockchain.go) - Container for managing multiple chains
 3. Provider Interface (provider.go) - Abstraction for blockchain providers
 4. Chain Family Support - Type-safe access to specific blockchain implementations

# Basic Usage

# Core BlockChain Interface

Every blockchain implementation satisfies the BlockChain interface:

	import "github.com/smartcontractkit/chainlink-deployments-framework/chain"

	// BlockChain interface provides basic chain information
	type BlockChain interface {
		String() string           // Human-readable chain info
		Name() string            // Chain name
		ChainSelector() uint64   // Unique chain identifier
		Family() string          // Blockchain family (evm, solana, etc.)
	}

	// Example usage with any blockchain
	func printChainInfo(bc chain.BlockChain) {
		fmt.Printf("Chain: %s\n", bc.String())         // "Ethereum Mainnet (1)"
		fmt.Printf("Name: %s\n", bc.Name())            // "Ethereum Mainnet"
		fmt.Printf("Selector: %d\n", bc.ChainSelector()) // 1
		fmt.Printf("Family: %s\n", bc.Family())        // "evm"
	}

# BlockChains Collection

The BlockChains collection provides powerful querying and management capabilities:

	import (
		"github.com/smartcontractkit/chainlink-deployments-framework/chain"
		"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
		"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	)

	// Create a collection of chains
	:= chain.NewBlockChains(map[uint64]chain.BlockChain{
		1:                evmMainnet,     // Ethereum mainnet
		1151111081099710: solanaMainnet, // Solana mainnet
		42161:           arbChain,       // Arbitrum
	})

	// Check if chains exist
	if chains.Exists(1) {
		fmt.Println("Ethereum mainnet is available")
	}

	// Check multiple chains at once
	if chains.ExistsN(1, 42161) {
		fmt.Println("Both Ethereum and Arbitrum are available")
	}

# Family-Specific Chain Access

Retrieve chains by their blockchain family with type safety:

	// Get all EVM chains
	evmChains := chains.EVMChains()
	for selector, evmChain := range evmChains {
		fmt.Printf("EVM Chain %d: %s\n", selector, evmChain.String())
		// evmChain is typed as evm.Chain
	}

	// Get all Solana chains
	solanaChains := chains.SolanaChains()
	for selector, solChain := range solanaChains {
		fmt.Printf("Solana Chain %d: %s\n", selector, solChain.String())
		// solChain is typed as solana.Chain
	}

	// Similarly for other families
	aptosChains := chains.AptosChains()
	suiChains := chains.SuiChains()
	tonChains := chains.TonChains()
	tronChains := chains.TronChains()
	cantonChains := chains.CantonChains()

# Iterating Over All Chains

Use the iterator interface to process all chains:

	// Iterate over all chains
	for selector, blockchain := range chains.All() {
		fmt.Printf("Processing chain %d (%s) from family %s\n",
			selector, blockchain.Name(), blockchain.Family())

		// Handle each chain based on its family
		switch blockchain.Family() {
		case "evm":
			// Handle EVM-specific logic
		case "solana":
			// Handle Solana-specific logic
		case "aptos":
			// Handle Aptos-specific logic
		}

# Advanced Querying

# Chain Selector Filtering

The package provides flexible filtering options for chain selectors:

	// Get all chain selectors
	allSelectors := chains.ListChainSelectors()
	fmt.Printf("All chains: %v\n", allSelectors)

	// Filter by blockchain family
	evmSelectors := chains.ListChainSelectors(
		chain.WithFamily("evm"),
	)
	fmt.Printf("EVM chains: %v\n", evmSelectors)

	// Exclude specific chains
	filteredSelectors := chains.ListChainSelectors(
		chain.WithFamily("evm"),
		chain.WithChainSelectorsExclusion([]uint64{42161}), // Exclude Arbitrum
	)
	fmt.Printf("EVM chains excluding Arbitrum: %v\n", filteredSelectors)

	// Combine multiple families
	testnetSelectors := chains.ListChainSelectors(
		chain.WithFamily("evm"),
		chain.WithFamily("solana"),
		chain.WithChainSelectorsExclusion([]uint64{1, 1151111081099710}), // Exclude mainnets
	)

# Creating Collections from Slices

	// Create from slice of chains
	chainList := []chain.BlockChain{evmChain, solanaChain, aptosChain}
	chainsFromSlice := chain.NewBlockChainsFromSlice(chainList)

	// Equivalent to map-based creation
	chainsFromMap := chain.NewBlockChains(map[uint64]chain.BlockChain{
		evmChain.ChainSelector():    evmChain,
		solanaChain.ChainSelector(): solanaChain,
		aptosChain.ChainSelector():  aptosChain,
	})

# Provider System

The Provider interface enables pluggable blockchain implementations:

	import "context"

	// Provider interface for blockchain providers
	type Provider interface {
		Initialize(ctx context.Context) (BlockChain, error)
		Name() string
		ChainSelector() uint64
		BlockChain() BlockChain
	}

	// Example provider usage
	func setupChain(provider chain.Provider) (chain.BlockChain, error) {
		ctx := context.Background()

		// Initialize the provider
		blockchain, err := provider.Initialize(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize %s: %w",
				provider.Name(), err)
		}

		fmt.Printf("Initialized %s chain with selector %d\n",
			provider.Name(), provider.ChainSelector())

		return blockchain, nil
	}

# Integration with Deployment Framework

The chain package integrates seamlessly with the broader deployment framework:

	import "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	// Create deployment environment with multiple chains
	env := &deployment.Environment{
		Chains: chains, // BlockChains collection
		// ... other environment fields
	}

	// Access chains in deployment operations
	evmChains := env.Chains.EVMChains()
	solanaChains := env.Chains.SolanaChains()
*/
package chain
