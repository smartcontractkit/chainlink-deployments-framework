package common

import (
	"errors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	"github.com/smartcontractkit/mcms/types"
)

const (
	proposalPathFlag  = "proposal"
	proposalTypeFlag  = "proposalType"
	environmentFlag   = "environment"
	chainSelectorFlag = "selector"
)

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func aptosRoleFromProposal(proposal *mcms.TimelockProposal) (*aptos.TimelockRole, error) {
	if proposal == nil {
		return nil, errors.New("aptos timelock proposal is needed")
	}

	switch proposal.Action {
	case types.TimelockActionBypass:
		role := aptos.TimelockRoleBypasser
		return &role, nil
	case types.TimelockActionSchedule:
		role := aptos.TimelockRoleProposer
		return &role, nil
	case types.TimelockActionCancel:
		role := aptos.TimelockRoleCanceller
		return &role, nil
	default:
		return nil, errors.New("unknown timelock action")
	}
}
