package memory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProposalID(t *testing.T) {
	t.Parallel()

	id := newProposalID()

	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "prop_"))
}

func TestNewJobID(t *testing.T) {
	t.Parallel()

	id := newJobID()

	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "job_"))
}

func TestNewNodeID(t *testing.T) {
	t.Parallel()

	id := newNodeID()

	assert.NotEmpty(t, id)
	assert.True(t, strings.HasPrefix(id, "node_"))
}

func TestNewID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prefix     string
		wantPrefix string
	}{
		{
			name:       "empty prefix",
			prefix:     "",
			wantPrefix: "_",
		},
		{
			name:       "single character prefix",
			prefix:     "a",
			wantPrefix: "a_",
		},
		{
			name:       "multi character prefix",
			prefix:     "test",
			wantPrefix: "test_",
		},
		{
			name:       "prefix with numbers",
			prefix:     "test123",
			wantPrefix: "test123_",
		},
		{
			name:       "prefix with special characters",
			prefix:     "test-123",
			wantPrefix: "test-123_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id := newID(tt.prefix)

			assert.NotEmpty(t, id)
			assert.True(t, strings.HasPrefix(id, tt.wantPrefix))
		})
	}
}
