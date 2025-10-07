package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseTask_ID(t *testing.T) {
	t.Parallel()

	task := baseTask{
		id: "test-id",
	}
	assert.Equal(t, "test-id", task.ID())
}
