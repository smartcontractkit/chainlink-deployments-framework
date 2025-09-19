// Package mcmsutils provides utilities for working with Multi-Chain Multi-Sig (MCMS) operations
// across different blockchain networks. This package serves as a bridge between the chainlink
// deployments framework and the MCMS SDK, offering factory patterns to create blockchain-specific
// instances of MCMS components and other utility functions.
//
// MCMS enables secure multi-signature operations across multiple blockchain networks through
// timelock contracts and multi-signature wallets. This package abstracts the complexity of
// creating the appropriate MCMS components for different blockchain families (EVM, Solana, Aptos).
//
// # Factory Pattern Overview
//
// This package implements the factory pattern to create blockchain-specific MCMS components:
//
//   - InspectorFactory: Creates inspectors for inspecting the MCMS contract
//   - ConverterFactory: Creates converters for transforming timelock proposals
//   - ExecutorFactory: Creates executors for executing the MCMS operations
//   - TimelockExecutorFactory: Creates executors for executing the Timelock contract operations
//
// # Supported Blockchains
//
// Different operations are supported on different blockchain families:
//
//   - EVM: All operations (inspection, conversion, execution, timelock execution)
//   - Solana: All operations (inspection, conversion, execution, timelock execution)
//   - Aptos: Limited to timelock operations (timelock inspection, conversion, execution)
//
// # Usage Example
//
//	// Create an inspector for a blockchain
//	inspectorFactory, err := GetInspectorFactory(blockchain)
//	if err != nil {
//		return fmt.Errorf("failed to get inspector factory: %w", err)
//	}
//	inspector, err := inspectorFactory.Make()
//	if err != nil {
//		return fmt.Errorf("failed to create inspector: %w", err)
//	}
package mcmsutils
