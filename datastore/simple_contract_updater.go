package datastore

import (
	"fmt"
	"time"
)

// SimpleContractUpdater is a specialized updater function for ContractMetadata with SimpleContract metadata.
// It updates the block number and records the update time.
func SimpleContractUpdater(update ContractMetadata, orig ContractMetadata) (ContractMetadata, error) {
	// Extract the original SimpleContract from metadata
	origSimpleContract, err := As[SimpleContract](orig.Metadata)
	if err != nil {
		return ContractMetadata{}, fmt.Errorf("failed to convert original metadata to SimpleContract: %w", err)
	}

	// Extract the update SimpleContract from metadata
	updateSimpleContract, err := As[SimpleContract](update.Metadata)
	if err != nil {
		return ContractMetadata{}, fmt.Errorf("failed to convert update metadata to SimpleContract: %w", err)
	}

	// Apply custom update logic - preserve the original transaction hash and deployment time
	// but update the block number from the update record and set a current timestamp
	result := SimpleContract{
		DeployedAt:   origSimpleContract.DeployedAt,    // Preserve original deployment time
		TxHash:       origSimpleContract.TxHash,        // Preserve original transaction hash
		BlockNumber:  updateSimpleContract.BlockNumber, // Use the new block number
		LastUpdateAt: time.Now(),                       // Add the current time as update time
	}

	// Create the updated record with the new SimpleContract as metadata
	updatedRecord := ContractMetadata{
		ChainSelector: update.ChainSelector,
		Address:       update.Address,
		Metadata:      result,
	}

	return updatedRecord, nil
}
