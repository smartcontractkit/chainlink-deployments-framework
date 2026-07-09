package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBuiltinReportName(t *testing.T) {
	t.Parallel()

	assert.True(t, IsBuiltinReportName("cld.builtin.timelock_delay.report"))
	assert.False(t, IsBuiltinReportName("cld.builtin.timelock_delay"))
	assert.False(t, IsBuiltinReportName("proposal_timelock_delay"))
	assert.False(t, IsBuiltinReportName(""))
}
