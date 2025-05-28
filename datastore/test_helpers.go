package datastore

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// SimpleContract represents metadata for a deployed contract for the exemplar domain.
type TestContractMetadata struct {
	// DeployedAt is the timestamp when the contract was deployed.
	DeployedAt time.Time `json:"deployed_at" format:"date-time"`
	// TxHash is the transaction hash of the deployment transaction.
	TxHash common.Hash `json:"tx_hash"`
	// BlockNumber is the block number where the contract was deployed.
	BlockNumber uint64 `json:"block_number"`
}

type TestEnvMetadata struct {
	// EnvName is the name of the environment.
	EnvName string `json:"env_name"`
	// EnvID is the unique identifier for the environment.
	EnvID string `json:"env_id"`
	// CreatedAt is the timestamp when the environment was created.
	CreatedAt time.Time `json:"created_at" format:"date-time"`
}
