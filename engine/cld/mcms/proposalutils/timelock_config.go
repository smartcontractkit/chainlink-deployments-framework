package proposalutils

import (
	"errors"
	"fmt"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/gagliardetto/solana-go"
	ownerhelpers "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/xssnick/tonutils-go/address"

	cldf_aptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	mcmscontracts "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/contracts/mcms"
)

// EVMMCMSWithTimelock adapts EVM MCMS-with-timelock state for TimelockConfig helpers.
type EVMMCMSWithTimelock interface {
	TimelockContracts() MCMSWithTimelockContracts
}

// SolanaMCMSWithTimelock adapts Solana MCMS-with-timelock state for TimelockConfig helpers.
type SolanaMCMSWithTimelock interface {
	TimelockPrograms() MCMSWithTimelockPrograms
}

// MCMSSuiteState holds the state of a single MCMS deployment - currently includes all contract addresses.
type MCMSSuiteState struct {
	// 3x MCMS contracts, each gets a role in the timelock
	Proposer  *address.Address
	Canceller *address.Address
	Bypasser  *address.Address

	// Timelock contract address for this MCMS suite
	Timelock *address.Address
}

// MCMSWithTimelockPrograms holds the Solana program and PDA seed values needed to resolve MCMS role addresses.
type MCMSWithTimelockPrograms struct {
	McmProgram       solana.PublicKey
	ProposerMcmSeed  mcmssolanasdk.PDASeed
	CancellerMcmSeed mcmssolanasdk.PDASeed
	BypasserMcmSeed  mcmssolanasdk.PDASeed
}

// TimelockConfig configures MCMS timelock proposal behavior.
type TimelockConfig struct {
	MinDelay                  time.Duration            `json:"minDelay"` // delay for timelock worker to execute the transfers.
	MCMSAction                mcmstypes.TimelockAction `json:"mcmsAction"`
	OverrideRoot              bool                     `json:"overrideRoot"`                        // if true, override the previous root with the new one.
	TimelockQualifierPerChain map[uint64]string        `json:"timelockQualifierPerChain,omitempty"` // optional qualifier to fetch timelock address from datastore
	ValidDuration             *mcmstypes.Duration      `json:"validDuration" yaml:"validDuration"`
}

func (tc *TimelockConfig) MCMBasedOnActionSolana(s SolanaMCMSWithTimelock) (string, error) {
	// if MCMSAction is not set, default to timelock.Schedule, this is to ensure no breaking changes for existing code
	if tc.MCMSAction == "" {
		tc.MCMSAction = mcmstypes.TimelockActionSchedule
	}

	programs := s.TimelockPrograms()
	switch tc.MCMSAction {
	case mcmstypes.TimelockActionSchedule:
		return mcmssolanasdk.ContractAddress(programs.McmProgram, programs.ProposerMcmSeed), nil
	case mcmstypes.TimelockActionCancel:
		return mcmssolanasdk.ContractAddress(programs.McmProgram, programs.CancellerMcmSeed), nil
	case mcmstypes.TimelockActionBypass:
		return mcmssolanasdk.ContractAddress(programs.McmProgram, programs.BypasserMcmSeed), nil
	default:
		return "", errors.New("invalid MCMS action")
	}
}

func (tc *TimelockConfig) MCMBasedOnActionTon(s *MCMSSuiteState) (string, error) {
	// if MCMSAction is not set, default to timelock.Schedule, this is to ensure no breaking changes for existing code
	if tc.MCMSAction == "" {
		tc.MCMSAction = mcmstypes.TimelockActionSchedule
	}
	switch tc.MCMSAction {
	case mcmstypes.TimelockActionSchedule:
		if s.Proposer == nil {
			return "", errors.New("missing TON proposer")
		}

		return s.Proposer.String(), nil
	case mcmstypes.TimelockActionCancel:
		if s.Canceller == nil {
			return "", errors.New("missing TON canceller")
		}

		return s.Canceller.String(), nil
	case mcmstypes.TimelockActionBypass:
		if s.Bypasser == nil {
			return "", errors.New("missing TON bypasser")
		}

		return s.Bypasser.String(), nil
	default:
		return "", errors.New("invalid MCMS action")
	}
}

func (tc *TimelockConfig) MCMBasedOnAction(s EVMMCMSWithTimelock) (*ownerhelpers.ManyChainMultiSig, error) {
	// if MCMSAction is not set, default to timelock.Schedule, this is to ensure no breaking changes for existing code
	if tc.MCMSAction == "" {
		tc.MCMSAction = mcmstypes.TimelockActionSchedule
	}

	contracts := s.TimelockContracts()
	switch tc.MCMSAction {
	case mcmstypes.TimelockActionSchedule:
		if contracts.ProposerMcm == nil {
			return nil, errors.New("missing proposerMcm")
		}

		return contracts.ProposerMcm, nil
	case mcmstypes.TimelockActionCancel:
		if contracts.CancellerMcm == nil {
			return nil, errors.New("missing cancellerMcm")
		}

		return contracts.CancellerMcm, nil
	case mcmstypes.TimelockActionBypass:
		if contracts.BypasserMcm == nil {
			return nil, errors.New("missing bypasserMcm")
		}

		return contracts.BypasserMcm, nil
	default:
		return nil, errors.New("invalid MCMS action")
	}
}

func (tc *TimelockConfig) validateCommon() error {
	// if MCMSAction is not set, default to timelock.Schedule
	if tc.MCMSAction == "" {
		tc.MCMSAction = mcmstypes.TimelockActionSchedule
	}
	if tc.MCMSAction != mcmstypes.TimelockActionSchedule &&
		tc.MCMSAction != mcmstypes.TimelockActionCancel &&
		tc.MCMSAction != mcmstypes.TimelockActionBypass {
		return fmt.Errorf("invalid MCMS type %s", tc.MCMSAction)
	}

	return nil
}

func (tc *TimelockConfig) Validate(chain cldf_evm.Chain, s EVMMCMSWithTimelock) error {
	err := tc.validateCommon()
	if err != nil {
		return err
	}

	contracts := s.TimelockContracts()
	if contracts.Timelock == nil {
		return fmt.Errorf("missing timelock on %s", chain)
	}
	if tc.MCMSAction == mcmstypes.TimelockActionSchedule && contracts.ProposerMcm == nil {
		return fmt.Errorf("missing proposerMcm on %s", chain)
	}
	if tc.MCMSAction == mcmstypes.TimelockActionCancel && contracts.CancellerMcm == nil {
		return fmt.Errorf("missing cancellerMcm on %s", chain)
	}
	if tc.MCMSAction == mcmstypes.TimelockActionBypass && contracts.BypasserMcm == nil {
		return fmt.Errorf("missing bypasserMcm on %s", chain)
	}
	if contracts.Timelock == nil {
		return fmt.Errorf("missing timelock on %s", chain)
	}
	if contracts.CallProxy == nil {
		return fmt.Errorf("missing callProxy on %s", chain)
	}

	return nil
}

func (tc *TimelockConfig) ValidateSolana(e cldf.Environment, chainSelector uint64) error {
	err := tc.validateCommon()
	if err != nil {
		return err
	}

	validateContract := func(contractType cldf.ContractType) error {
		timelockID, searchErr := cldf.SearchAddressBook(e.ExistingAddresses, chainSelector, contractType) //nolint:staticcheck // preserve AddressBook compatibility from the core helper.
		if searchErr != nil {
			return fmt.Errorf("%s not present on the chain %w", contractType, searchErr)
		}
		// Make sure addresses are correctly parsed. Format is: "programID.PDASeed"
		_, _, parseErr := mcmssolanasdk.ParseContractAddress(timelockID)
		if parseErr != nil {
			return fmt.Errorf("failed to parse timelock address: %w", parseErr)
		}

		return nil
	}

	err = validateContract(mcmscontracts.RBACTimelock)
	if err != nil {
		return err
	}

	switch tc.MCMSAction {
	case mcmstypes.TimelockActionSchedule:
		err = validateContract(mcmscontracts.ProposerManyChainMultisig)
		if err != nil {
			return err
		}
	case mcmstypes.TimelockActionCancel:
		err = validateContract(mcmscontracts.CancellerManyChainMultisig)
		if err != nil {
			return err
		}
	case mcmstypes.TimelockActionBypass:
		err = validateContract(mcmscontracts.BypasserManyChainMultisig)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid MCMS action %s", tc.MCMSAction)
	}

	return nil
}

func (tc *TimelockConfig) ValidateAptos(chain cldf_aptos.Chain, mcmsAddress aptos.AccountAddress) error {
	if err := tc.validateCommon(); err != nil {
		return err
	}
	if mcmsAddress == (aptos.AccountAddress{}) {
		return fmt.Errorf("aptos MCMS contract not present on chain %s", chain)
	}

	return nil
}
