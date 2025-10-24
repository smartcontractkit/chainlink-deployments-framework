package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

func TestMatchesSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		labels   map[string]*string
		selector *ptypes.Selector
		want     bool
	}{
		{
			name:   "EQ operation - matching value",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: true,
		},
		{
			name:   "EQ operation - non-matching value",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("test"),
			},
			want: false,
		},
		{
			name:   "EQ operation - missing key",
			labels: map[string]*string{"type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name:   "EQ operation - nil value",
			labels: map[string]*string{"environment": pointer.To("prod")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: nil,
			},
			want: false,
		},
		{
			name:   "EQ operation - nil label value",
			labels: map[string]*string{"environment": nil, "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name:   "NOT_EQ operation - different value",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_NOT_EQ,
				Value: pointer.To("test"),
			},
			want: true,
		},
		{
			name:   "NOT_EQ operation - same value",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_NOT_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name:   "NOT_EQ operation - missing key",
			labels: map[string]*string{"type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_NOT_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name:   "IN operation - matching value in comma-separated list",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_IN,
				Value: pointer.To("prod,test,dev"),
			},
			want: true,
		},
		{
			name:   "IN operation - non-matching value in comma-separated list",
			labels: map[string]*string{"environment": pointer.To("staging"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_IN,
				Value: pointer.To("prod,test,dev"),
			},
			want: false,
		},
		{
			name:   "IN operation - matching value with spaces",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_IN,
				Value: pointer.To(" prod , test , dev "),
			},
			want: true,
		},
		{
			name:   "NOT_IN operation - value not in comma-separated list",
			labels: map[string]*string{"environment": pointer.To("staging"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_NOT_IN,
				Value: pointer.To("prod,test,dev"),
			},
			want: true,
		},
		{
			name:   "NOT_IN operation - value in comma-separated list",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_NOT_IN,
				Value: pointer.To("prod,test,dev"),
			},
			want: false,
		},
		{
			name:   "EXIST operation - key exists",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key: "environment",
				Op:  ptypes.SelectorOp_EXIST,
			},
			want: true,
		},
		{
			name:   "EXIST operation - key does not exist",
			labels: map[string]*string{"type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key: "environment",
				Op:  ptypes.SelectorOp_EXIST,
			},
			want: false,
		},
		{
			name:   "EXIST operation - key exists with nil value",
			labels: map[string]*string{"environment": nil, "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key: "environment",
				Op:  ptypes.SelectorOp_EXIST,
			},
			want: true,
		},
		{
			name:   "NOT_EXIST operation - key does not exist",
			labels: map[string]*string{"type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key: "environment",
				Op:  ptypes.SelectorOp_NOT_EXIST,
			},
			want: true,
		},
		{
			name:   "NOT_EXIST operation - key exists",
			labels: map[string]*string{"environment": pointer.To("prod"), "type": pointer.To("ocr")},
			selector: &ptypes.Selector{
				Key: "environment",
				Op:  ptypes.SelectorOp_NOT_EXIST,
			},
			want: false,
		},
		{
			name:   "unknown operation",
			labels: map[string]*string{"environment": pointer.To("prod")},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp(999), // Unknown operation
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name:   "empty labels map",
			labels: map[string]*string{},
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
		{
			name:   "nil labels map",
			labels: nil,
			selector: &ptypes.Selector{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("prod"),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := matchesSelector(tt.labels, tt.selector)
			assert.Equal(t, tt.want, result)
		})
	}
}
