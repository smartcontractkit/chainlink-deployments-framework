package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVersionToPath verifies the semver→directory-path conversion.
func TestVersionToPath(t *testing.T) {
	t.Parallel()
	cases := []struct{ version, want string }{
		{"1.0.0", "v1_0_0"},
		{"1.2.3", "v1_2_3"},
		{"0.0.1", "v0_0_1"},
	}
	for _, tc := range cases {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			if got := versionToPath(tc.version); got != tc.want {
				t.Errorf("versionToPath(%q) = %q, want %q", tc.version, got, tc.want)
			}
		})
	}
}

// TestLoadTemplate_UnknownFamily verifies that loadTemplate returns an error
// for a chain family that has no registered template.
func TestLoadTemplate_UnknownFamily(t *testing.T) {
	t.Parallel()
	_, err := loadTemplate("solana")
	if err == nil {
		t.Error("expected error for unsupported chain family, got nil")
	}
}

func TestCapitalize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input, want string
	}{
		{"", ""},
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"hELLO", "HELLO"},
		{"a", "A"},
		{"1abc", "1abc"}, // non-alpha first char is left as-is
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := capitalize(tc.input); got != tc.want {
				t.Errorf("capitalize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestWriteGoFile_InvalidSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := writeGoFile(filepath.Join(dir, "out.go"), []byte("this is not valid Go }{"))
	if err == nil {
		t.Fatal("expected formatting error for invalid Go source, got nil")
	}
}

func TestWriteGoFile_WritesFormattedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := []byte("package foo\nfunc F(){}\n")
	outPath := filepath.Join(dir, "sub", "out.go")

	if err := writeGoFile(outPath, src); err != nil {
		t.Fatalf("writeGoFile: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty output file")
	}
}
