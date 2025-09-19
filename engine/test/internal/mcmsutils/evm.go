package mcmsutils

import (
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"

	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

var _ InspectorFactory = &evmInspectorFactory{}

// evmInspectorFactory is a factory for creating EVM-specific MCMS inspectors.
// It implements the InspectorFactory interface and is responsible for creating
// inspectors that can examine the state of MCMS and Timelock contracts on EVM-compatible blockchains.
type evmInspectorFactory struct {
	chain fchainevm.Chain // The EVM chain configuration and client
}

// newEVMInspectorFactory creates a new EVM inspector factory.
func newEVMInspectorFactory(chain fchainevm.Chain) *evmInspectorFactory {
	return &evmInspectorFactory{chain: chain}
}

// Make creates and returns a new EVM MCMS inspector.
func (f *evmInspectorFactory) Make() (mcmssdk.Inspector, error) {
	return mcmsevmsdk.NewInspector(f.chain.Client), nil
}

//------------------------------------------------------------------------------

var _ ConverterFactory = &evmConverterFactory{}

// evmConverterFactory is a factory for creating EVM-specific timelock converters.
// It implements the ConverterFactory interface and creates converters that can
// transform MCMS timelock proposals into a standard MCMS proposal.
type evmConverterFactory struct{}

// newEVMConverterFactory creates a new EVM converter factory.
func newEVMConverterFactory() *evmConverterFactory {
	return &evmConverterFactory{}
}

// Make creates and returns a new EVM timelock converter.
func (f *evmConverterFactory) Make() (mcmssdk.TimelockConverter, error) {
	return &mcmsevmsdk.TimelockConverter{}, nil
}

//------------------------------------------------------------------------------

var _ ExecutorFactory = &evmExecutorFactory{}

// evmExecutorFactory is a factory for creating EVM-specific MCMS executors.
// It implements the ExecutorFactory interface and creates executors that can
// execute MCMS operations on EVM-compatible blockchains.
type evmExecutorFactory struct {
	chain   fchainevm.Chain     // The EVM chain configuration and client
	encoder *mcmsevmsdk.Encoder // The encoder for creating EVM-specific transaction data
}

// newEVMExecutorFactory creates a new EVM executor factory.
func newEVMExecutorFactory(
	chain fchainevm.Chain, encoder *mcmsevmsdk.Encoder,
) *evmExecutorFactory {
	return &evmExecutorFactory{
		chain:   chain,
		encoder: encoder,
	}
}

// Make creates and returns a new EVM MCMS executor.
func (f *evmExecutorFactory) Make() (mcmssdk.Executor, error) {
	return mcmsevmsdk.NewExecutor(f.encoder, f.chain.Client, f.chain.DeployerKey), nil
}

//------------------------------------------------------------------------------

var _ TimelockExecutorFactory = &evmTimelockExecutorFactory{}

// evmTimelockExecutorFactory is a factory for creating EVM-specific timelock executors.
// It implements the TimelockExecutorFactory interface and creates executors specifically
// designed for executing Timelock operations on EVM-compatible blockchains.
type evmTimelockExecutorFactory struct {
	chain fchainevm.Chain // The EVM chain configuration and client
}

// newEVMTimelockExecutorFactory creates a new EVM timelock executor factory.
func newEVMTimelockExecutorFactory(chain fchainevm.Chain) *evmTimelockExecutorFactory {
	return &evmTimelockExecutorFactory{chain: chain}
}

// Make creates and returns a new EVM timelock executor.
func (f *evmTimelockExecutorFactory) Make() (mcmssdk.TimelockExecutor, error) {
	return mcmsevmsdk.NewTimelockExecutor(f.chain.Client, f.chain.DeployerKey), nil
}
