package chainsmetadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestAptosRoleFromAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		action       mcmsTypes.TimelockAction
		expectedRole aptos.TimelockRole
		expectError  bool
	}{
		{
			name:         "bypass action returns bypasser role",
			action:       mcmsTypes.TimelockActionBypass,
			expectedRole: aptos.TimelockRoleBypasser,
			expectError:  false,
		},
		{
			name:         "schedule action returns proposer role",
			action:       mcmsTypes.TimelockActionSchedule,
			expectedRole: aptos.TimelockRoleProposer,
			expectError:  false,
		},
		{
			name:         "cancel action returns canceller role",
			action:       mcmsTypes.TimelockActionCancel,
			expectedRole: aptos.TimelockRoleCanceller,
			expectError:  false,
		},
		{
			name:         "unknown action returns error",
			action:       mcmsTypes.TimelockAction("unknown"),
			expectedRole: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := AptosRoleFromAction(tt.action)

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, "unknown timelock action", err.Error())
				assert.Equal(t, tt.expectedRole, role)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
		})
	}
}

func TestAptosRoleFromProposal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		proposal     *mcms.TimelockProposal
		expectedRole aptos.TimelockRole
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "nil proposal returns error",
			proposal:     nil,
			expectedRole: 0,
			expectError:  true,
			errorMsg:     "aptos timelock proposal is needed",
		},
		{
			name: "proposal with bypass action returns bypasser role",
			proposal: &mcms.TimelockProposal{
				Action: mcmsTypes.TimelockActionBypass,
			},
			expectedRole: aptos.TimelockRoleBypasser,
			expectError:  false,
		},
		{
			name: "proposal with schedule action returns proposer role",
			proposal: &mcms.TimelockProposal{
				Action: mcmsTypes.TimelockActionSchedule,
			},
			expectedRole: aptos.TimelockRoleProposer,
			expectError:  false,
		},
		{
			name: "proposal with cancel action returns canceller role",
			proposal: &mcms.TimelockProposal{
				Action: mcmsTypes.TimelockActionCancel,
			},
			expectedRole: aptos.TimelockRoleCanceller,
			expectError:  false,
		},
		{
			name: "proposal with unknown action returns error",
			proposal: &mcms.TimelockProposal{
				Action: mcmsTypes.TimelockAction("unknown"),
			},
			expectedRole: 0,
			expectError:  true,
			errorMsg:     "unknown timelock action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			role, err := AptosRoleFromProposal(tt.proposal)

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
				assert.Equal(t, aptos.TimelockRole(0), role)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
		})
	}
}
