package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_deleteFromSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		slice []string
		item  string
		want  []string
	}{
		{
			name:  "remove first of one match at index zero",
			slice: []string{"a", "b", "c"},
			item:  "a",
			want:  []string{"b", "c"},
		},
		{
			name:  "remove first of two matches",
			slice: []string{"a", "b", "a", "c"},
			item:  "a",
			want:  []string{"b", "a", "c"},
		},
		{
			name:  "remove last of one match",
			slice: []string{"a", "b", "c"},
			item:  "c",
			want:  []string{"a", "b"},
		},
		{
			name:  "remove none",
			slice: []string{"a", "b", "a", "c"},
			item:  "d",
			want:  []string{"a", "b", "a", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deleteFromSlice(tt.slice, tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}
