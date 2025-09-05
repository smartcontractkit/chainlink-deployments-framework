package sui

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/block-vision/sui-go-sdk/sui"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents an Sui chain.
type Chain struct {
	ChainMetadata
	Client sui.ISuiAPI
	Signer SuiSigner
	URL    string
	// TODO: Implement ConfirmTransaction. Current tooling relies on node local execution
}

// AddressToBytes converts a Sui address string to bytes.
// Sui addresses are hex strings typically prefixed with "0x" (32 bytes).
func (c Chain) AddressToBytes(address string) ([]byte, error) {
	// Remove "0x" prefix if present
	address = strings.TrimPrefix(address, "0x")

	// Sui addresses should be 64 hex characters (32 bytes)
	if len(address) != 64 {
		return nil, fmt.Errorf("invalid Sui address format: expected 64 hex characters, got %d", len(address))
	}

	addressBytes, err := hex.DecodeString(address)
	if err != nil {
		return nil, fmt.Errorf("invalid Sui address format: %s, error: %w", address, err)
	}

	return addressBytes, nil
}
