package mcmsutils

import (
	"errors"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fchainaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

// InspectorFactory creates MCMS Inspector instances for MCMS contract inspection.
type InspectorFactory interface {
	Make() (mcmssdk.Inspector, error)
}

// GetInspectorFactory returns a blockchain-specific InspectorFactory based on the provided
// blockchain's type for use in performing MCMS operations.
//
// Note: Aptos chains only support timelock-specific inspection (use GetTimelockInspectorFactory
// when performing inspection against a Timelock proposal).
//
// Returns an error if the blockchain family is not supported.
func GetInspectorFactory(
	blockchain fchain.BlockChain,
) (InspectorFactory, error) {
	switch c := blockchain.(type) {
	case fchainevm.Chain:
		return newEVMInspectorFactory(c), nil
	case fchainsolana.Chain:
		return newSolanaInspectorFactory(c), nil
	case fchainaptos.Chain:
		return nil, errors.New("aptos does not support inspection on non-timelock proposals")
	default:
		return nil, errFamilyNotSupported(c.Family())
	}
}

// GetTimelockInspectorFactory returns a blockchain-specific InspectorFactory based on the provided
// blockchain's type for use in performing MCMS Timelock operations.
//
// Returns an error if the blockchain family is not supported.
func GetTimelockInspectorFactory(
	blockchain fchain.BlockChain,
	action mcmstypes.TimelockAction,
) (InspectorFactory, error) {
	switch c := blockchain.(type) {
	case fchainevm.Chain:
		return newEVMInspectorFactory(c), nil
	case fchainsolana.Chain:
		return newSolanaInspectorFactory(c), nil
	case fchainaptos.Chain:
		return newAptosInspectorFactory(c, action), nil
	default:
		return nil, errFamilyNotSupported(c.Family())
	}
}

// ConverterFactory creates MCMS TimelockConverter instances for converting a Timelock proposal to
// an MCMS proposal.
type ConverterFactory interface {
	Make() (mcmssdk.TimelockConverter, error)
}

// GetConverterFactory returns a blockchain family specific ConverterFactory.
//
// Returns an error if the blockchain family is not supported.
func GetConverterFactory(family string) (ConverterFactory, error) {
	switch family {
	case chainselectors.FamilyEVM:
		return newEVMConverterFactory(), nil
	case chainselectors.FamilySolana:
		return newSolanaConverterFactory(), nil
	case chainselectors.FamilyAptos:
		return newAptosConverterFactory(), nil
	default:
		return nil, errFamilyNotSupported(family)
	}
}

// ExecutorFactory creates MCMS Executor instances for executing MCMS contract operations.
type ExecutorFactory interface {
	Make() (mcmssdk.Executor, error)
}

// GetExecutorFactory returns a blockchain-specific ExecutorFactory with the appropriate encoder.
//
// The provided encoder must match the blockchain type for proper transaction encoding and execution.
//
// Returns an error if the blockchain family is not supported.
func GetExecutorFactory(
	blockchain fchain.BlockChain,
	encoder mcmssdk.Encoder,
) (ExecutorFactory, error) {
	switch c := blockchain.(type) {
	case fchainevm.Chain:
		encoder, ok := encoder.(*mcmsevmsdk.Encoder)
		if !ok {
			return nil, errEncoderNotFound(c.Selector)
		}

		return newEVMExecutorFactory(c, encoder), nil
	case fchainsolana.Chain:
		encoder, ok := encoder.(*mcmssolanasdk.Encoder)
		if !ok {
			return nil, errEncoderNotFound(c.Selector)
		}

		return newSolanaExecutorFactory(c, encoder), nil
	case fchainaptos.Chain:
		encoder, ok := encoder.(*mcmsaptossdk.Encoder)
		if !ok {
			return nil, errEncoderNotFound(c.Selector)
		}

		return newAptosExecutorFactory(c, encoder), nil
	default:
		return nil, errFamilyNotSupported(c.Family())
	}
}

// TimelockExecutorFactory creates MCMS TimelockExecutor instances for executing the Timelock
// contract operations.
type TimelockExecutorFactory interface {
	Make() (mcmssdk.TimelockExecutor, error)
}

// GetTimelockExecutorFactory returns a blockchain-specific TimelockExecutorFactory based on the
// provided blockchain type.
//
// Returns an error if the blockchain family is not supported.
func GetTimelockExecutorFactory(
	blockchain fchain.BlockChain,
) (TimelockExecutorFactory, error) {
	switch c := blockchain.(type) {
	case fchainevm.Chain:
		return newEVMTimelockExecutorFactory(c), nil
	case fchainsolana.Chain:
		return newSolanaTimelockExecutorFactory(c), nil
	case fchainaptos.Chain:
		return newAptosTimelockExecutorFactory(c), nil
	default:
		return nil, errFamilyNotSupported(c.Family())
	}
}
