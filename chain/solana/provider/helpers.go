package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gagliardetto/solana-go"
)

// isValidFilepath checks if the provided file path exists and is absolute.
func isValidFilepath(fp string) error {
	_, err := os.Stat(fp)
	if os.IsNotExist(err) {
		return fmt.Errorf("required file does not exist: %s", fp)
	}

	if !filepath.IsAbs(fp) {
		return fmt.Errorf("required file is not absolute: %s", fp)
	}

	return nil
}

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
