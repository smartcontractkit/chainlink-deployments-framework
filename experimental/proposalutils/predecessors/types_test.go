package predecessors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMcmOpData_EndingOpCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data McmOpData
		want uint64
	}{
		{
			name: "standard case",
			data: McmOpData{
				StartingOpCount: 10,
				OpsCount:        5,
			},
			want: 15,
		},
		{
			name: "zero ops count",
			data: McmOpData{
				StartingOpCount: 20,
				OpsCount:        0,
			},
			want: 20,
		},
		{
			name: "zero starting op count",
			data: McmOpData{
				StartingOpCount: 0,
				OpsCount:        7,
			},
			want: 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.data.EndingOpCount()
			require.Equal(t, tt.want, got)
		})
	}
}
