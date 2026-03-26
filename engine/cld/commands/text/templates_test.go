package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLongDesc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple string",
			input:    "This is a description.",
			expected: "This is a description.",
		},
		{
			name:     "string with leading whitespace",
			input:    "   Leading spaces.",
			expected: "Leading spaces.",
		},
		{
			name:     "string with trailing whitespace",
			input:    "Trailing spaces.   ",
			expected: "Trailing spaces.",
		},
		{
			name:     "string with leading and trailing whitespace",
			input:    "   Both ends.   ",
			expected: "Both ends.",
		},
		{
			name:     "multiline string",
			input:    "Line one.\nLine two.\nLine three.",
			expected: "Line one.\nLine two.\nLine three.",
		},
		{
			name: "multiline with leading/trailing newlines",
			input: `
				This is a long description.
				It spans multiple lines.
			`,
			expected: "This is a long description.\n\t\t\t\tIt spans multiple lines.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := LongDesc(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "simple string",
			input:    "myapp command --flag value",
			expected: "  myapp command --flag value",
		},
		{
			name:     "string with leading whitespace",
			input:    "   myapp command",
			expected: "  myapp command",
		},
		{
			name:     "multiline examples",
			input:    "# Example 1\nmyapp cmd1\n\n# Example 2\nmyapp cmd2",
			expected: "  # Example 1\n  myapp cmd1\n  \n  # Example 2\n  myapp cmd2",
		},
		{
			name: "multiline with leading/trailing whitespace",
			input: `
				# Run the command
				myapp run --env staging

				# With additional flags
				myapp run --env staging --verbose
			`,
			expected: "  # Run the command\n  myapp run --env staging\n  \n  # With additional flags\n  myapp run --env staging --verbose",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := Examples(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIndentation(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "  ", Indentation, "Indentation should be two spaces")
}
