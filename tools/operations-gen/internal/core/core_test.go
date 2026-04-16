package core

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVersionToPath verifies the semver-to-directory-path conversion.
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
			if got := VersionToPath(tc.version); got != tc.want {
				t.Errorf("VersionToPath(%q) = %q, want %q", tc.version, got, tc.want)
			}
		})
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
			if got := Capitalize(tc.input); got != tc.want {
				t.Errorf("Capitalize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestWriteGoFileInvalidSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := WriteGoFile(filepath.Join(dir, "out.go"), []byte("this is not valid Go }{"))
	if err == nil {
		t.Fatal("expected formatting error for invalid Go source, got nil")
	}
}

func TestWriteGoFileWritesFormattedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := []byte("package foo\nfunc F(){}\n")
	outPath := filepath.Join(dir, "sub", "out.go")

	if err := WriteGoFile(outPath, src); err != nil {
		t.Fatalf("WriteGoFile: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty output file")
	}
}
