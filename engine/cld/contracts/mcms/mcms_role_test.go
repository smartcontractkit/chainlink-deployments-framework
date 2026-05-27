package mcms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMCMSRole_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role MCMSRole
		want string
	}{
		{name: "proposer", role: ProposerRole, want: "PROPOSER"},
		{name: "bypasser", role: BypasserRole, want: "BYPASSER"},
		{name: "canceller", role: CancellerRole, want: "CANCELLER"},
		{name: "custom role", role: MCMSRole("CUSTOM"), want: "CUSTOM"},
		{name: "empty", role: MCMSRole(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.role.String())
		})
	}
}
