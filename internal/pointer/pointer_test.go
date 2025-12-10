package pointer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_To(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give any
	}{
		{
			name: "string",
			give: "x",
		},
		{
			name: "time",
			give: time.Date(2014, 6, 25, 12, 24, 40, 0, time.UTC),
		},
		{
			name: "int32",
			give: int32(1),
		},
		{
			name: "int64",
			give: int64(1),
		},
		{
			name: "uint",
			give: uint(1),
		},
		{
			name: "uint32",
			give: uint(1),
		},
		{
			name: "int",
			give: int(1),
		},
		{
			name: "float64",
			give: float64(1),
		},
		{
			name: "bool",
			give: true,
		},
		{
			name: "struct",
			give: struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.give, *To(tt.give))
		})
	}
}
