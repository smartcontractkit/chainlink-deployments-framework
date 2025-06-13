package datastore

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// SimpleContract represents metadata for a deployed contract for the exemplar domain.
type SimpleContract struct {
	// DeployedAt is the timestamp when the contract was deployed.
	DeployedAt time.Time `json:"deployed_at" format:"date-time"`
	// TxHash is the transaction hash of the deployment transaction.
	TxHash common.Hash `json:"tx_hash"`
	// BlockNumber is the block number where the contract was deployed.
	BlockNumber uint64 `json:"block_number"`
	// LastUpdateAt is the timestamp when the contract was last updated.
	LastUpdateAt time.Time `json:"last_updated_at" format:"date-time"`
}
