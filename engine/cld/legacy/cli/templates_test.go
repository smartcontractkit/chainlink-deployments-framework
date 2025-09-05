package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LongDesc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "empty string produces empty string",
			give: "",
			want: "",
		},
		{
			name: "single line string produces same string",
			give: "hello world",
			want: "hello world",
		},
		{
			name: "multi line string produces same string",
			give: "hello\nworld",
			want: "hello\nworld",
		},
		{
			name: "multi line string with leading/trailing whitespace trims whitespace",
			give: "  hello\nworld  ",
			want: "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := LongDesc(tt.give)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Examples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "empty string produces empty string",
			give: "",
			want: "",
		},
		{
			name: "single line string indents the line",
			give: "hello world",
			want: "  hello world",
		},
		{
			name: "multi line string indents each line",
			give: "hello\nworld",
			want: "  hello\n  world",
		},
		{
			name: "multi line string with leading/trailing whitespace trims trailing whitespace and indents each line",
			give: "  hello\nworld  ",
			want: "  hello\n  world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Examples(tt.give)

			assert.Equal(t, tt.want, got)
		})
	}
}
