package provider

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
)

// writePrivateKeyToPath writes the provided Solana private key to the specified file path in JSON
// format. The private key is stored as an array of integers, where each integer represents a byte
// of the private key.
func writePrivateKeyToPath(keyPath string, privKey solana.PrivateKey) error {
	b := []byte(privKey)

	// Convert bytes to slice of integers for JSON conversion
	privKeyInts := make([]int, len(b))
	for i, b := range b {
		privKeyInts[i] = int(b)
	}

	// Marshal the integer array to JSON
	privKeyJSON, err := json.Marshal(privKeyInts)
	if err != nil {
		return err
	}

	// Write the JSON to the specified file path
	if err = os.WriteFile(keyPath, privKeyJSON, 0600); err != nil {
		return fmt.Errorf("failed to write keypair to file: %w", err)
	}

	return nil
}
