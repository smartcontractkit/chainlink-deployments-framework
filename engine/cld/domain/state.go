package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
)

// SaveViewState saves the view state as JSON to a file at the filePath.
func SaveViewState(filePath string, v json.Marshaler) error {
	// Ensure that the directory that the file will be written to exists.
	dirPath := path.Dir(filePath)
	if _, err := os.Stat(dirPath); err != nil {
		return fmt.Errorf("failed to stat %s: %w", dirPath, err)
	}

	b, err := v.MarshalJSON()
	if err != nil {
		return fmt.Errorf("unable to marshal state: %w", err)
	}

	if err = os.WriteFile(filePath, b, 0600); err != nil {
		return fmt.Errorf("failed to write state file '%s': %w", filePath, err)
	}

	return nil
}
