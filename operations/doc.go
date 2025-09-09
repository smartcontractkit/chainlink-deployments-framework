/*
Package operations provides the Operations API for managing and executing deployment operations
in a structured, reliable, and traceable manner.

# Operations API

The Operations API enables:
- Defining reusable deployment operations with versioning
- Executing operations with retry logic and error handling
- Tracking operation results and generating reports
- Sequencing multiple operations with dependencies

# Core Components

Operation:
  - Defines a single deployment operation with inputs, dependencies, and outputs
  - Includes versioning, validation, and execution logic
  - Supports generic typing for type-safe operation definitions

Registry:
  - Stores and retrieves operations by ID and version
  - Enables operation lookup and reuse across deployments
  - Provides centralized operation management

Executor:
  - Executes operations with configurable retry policies
  - Handles operation failures and recovery strategies
  - Supports input hooks for dynamic parameter adjustment

Sequence:
  - Orchestrates multiple operations in dependency order
  - Manages operation execution flow and error propagation
  - Provides sequence-level reporting and validation

Reporter:
  - Tracks operation execution results and metadata
  - Generates detailed reports for audit and debugging
  - Supports custom reporting formats and outputs

# Basic Usage

	// Define an operation
	op := operations.NewOperation(
		operations.OperationDef{ID: "deploy-contract", Version: "1.0.0"},
		handler,
	)

	// Execute the operation
	bundle := operations.NewBundle(logger, reporter)
	result, err := operations.ExecuteOperation(bundle, op, input, deps)
*/
package operations
