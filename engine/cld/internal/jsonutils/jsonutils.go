package jsonutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
)

// WriteFile marshals data into pretty JSON and writes it at path.
func WriteFile(path string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0600)
}

// LoadFromFS loads a JSON file from the filesystem, instantiates and unmarshals it into T.
func LoadFromFS[T any](fs fs.ReadFileFS, path string) (T, error) {
	var v T

	f, err := fs.ReadFile(path)
	if err != nil {
		return v, fmt.Errorf("failed to read %s: %w", path, err)
	}

	dec := json.NewDecoder(bytes.NewReader(f))
	dec.UseNumber()
	if err = dec.Decode(&v); err != nil {
		return v, fmt.Errorf("failed to unmarshal JSON at path %s: %w", path, err)
	}

	return v, nil
}
