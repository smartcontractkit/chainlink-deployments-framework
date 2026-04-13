package main

import (
	"testing"
)

// TestLoadTemplate_UnknownFamily verifies that loadTemplate returns an error
// for a chain family that has no registered template.
func TestLoadTemplate_UnknownFamily(t *testing.T) {
	t.Parallel()
	_, err := loadTemplate("solana")
	if err == nil {
		t.Error("expected error for unsupported chain family, got nil")
	}
}
