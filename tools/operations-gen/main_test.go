package main

import (
	"bytes"
	"strings"
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

// TestVersionFlag verifies that the -version flag prints version metadata to
// stdout and returns exit code 0 without requiring a config file.
func TestVersionFlag(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"-version"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"version=" + version,
		"commit=" + commit,
		"date=" + date,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("-version output missing %q; got: %s", want, output)
		}
	}
}
