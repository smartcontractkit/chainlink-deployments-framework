package proposalutils

import (
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

func GetAptosRoleFromAction(action mcmstypes.TimelockAction) (mcmsaptossdk.TimelockRole, error) {
	if action == "" {
		return mcmsaptossdk.TimelockRoleProposer, nil
	}

	return mcmsaptossdk.AptosRoleFromAction(action)
}
