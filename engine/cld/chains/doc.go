// Package chains provides multi-blockchain chain loading and management capabilities
// for the Chainlink Deployments Framework (CLD) engine.
//
// This package implements a unified interface for loading and initializing blockchain
// connections across multiple blockchain families, including EVM, Solana, Aptos, Sui,
// TON, and Tron networks. It provides concurrent chain loading with proper error handling
// and configuration-based chain selection.
//
// # Core Components
//
// The package is built around several key components:
//
//   - ChainLoader interface: Defines the contract for loading blockchain connections
//   - LoadChains function: Main entry point for concurrent chain loading
//   - Chain-specific loaders: Individual implementations for each blockchain family
//   - Configuration-based initialization: Uses network configs and secrets for setup
//
// # Chain Loading Process
//
// The chain loading process follows these steps:
//
//  1. Filter requested chain selectors based on available loaders and configuration
//  2. Create appropriate chain loaders for each supported blockchain family
//  3. Load chains concurrently using goroutines for optimal performance
//  4. Collect results and handle any loading errors
//  5. Return a unified BlockChains collection or aggregated error information
//
// # Usage Example
//
//	ctx := context.Background()
//
//	// Load configuration
//	cfg := &config.Config{
//		Networks: networkConfig,
//		Env: envConfig,
//	}
//
//	// Specify which chains to load
//	chainSelectors := []uint64{
//		chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
//		chainsel.POLYGON_TESTNET_MUMBAI.Selector,
//	}
//
//	// Load chains concurrently
//	chains, err := LoadChains(ctx, logger, cfg, chainSelectors)
//	if err != nil {
//		// Handle loading errors
//		log.Fatal(err)
//	}
//
//	// Use loaded chains for deployments
//	for _, chain := range chains.All() {
//		// Perform blockchain operations
//	}
package chains
