package mcmsutils

import (
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"

	fchainevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

var _ InspectorFactory = &evmInspectorFactory{}

type evmInspectorFactory struct {
	chain fchainevm.Chain
}

func newEVMInspectorFactory(chain fchainevm.Chain) *evmInspectorFactory {
	return &evmInspectorFactory{chain: chain}
}

func (f *evmInspectorFactory) Make() (mcmssdk.Inspector, error) {
	return mcmsevmsdk.NewInspector(f.chain.Client), nil
}

//------------------------------------------------------------------------------

var _ ConverterFactory = &evmConverterFactory{}

type evmConverterFactory struct{}

func newEVMConverterFactory() *evmConverterFactory {
	return &evmConverterFactory{}
}

func (f *evmConverterFactory) Make() (mcmssdk.TimelockConverter, error) {
	return &mcmsevmsdk.TimelockConverter{}, nil
}

//------------------------------------------------------------------------------

var _ ExecutorFactory = &evmExecutorFactory{}

type evmExecutorFactory struct {
	chain   fchainevm.Chain
	encoder *mcmsevmsdk.Encoder
}

func newEVMExecutorFactory(
	chain fchainevm.Chain, encoder *mcmsevmsdk.Encoder,
) *evmExecutorFactory {
	return &evmExecutorFactory{
		chain:   chain,
		encoder: encoder,
	}
}

func (f *evmExecutorFactory) Make() (mcmssdk.Executor, error) {
	return mcmsevmsdk.NewExecutor(f.encoder, f.chain.Client, f.chain.DeployerKey), nil
}

//------------------------------------------------------------------------------

var _ TimelockExecutorFactory = &evmTimelockExecutorFactory{}

type evmTimelockExecutorFactory struct {
	chain fchainevm.Chain
}

func newEVMTimelockExecutorFactory(chain fchainevm.Chain) *evmTimelockExecutorFactory {
	return &evmTimelockExecutorFactory{chain: chain}
}

func (f *evmTimelockExecutorFactory) Make() (mcmssdk.TimelockExecutor, error) {
	return mcmsevmsdk.NewTimelockExecutor(f.chain.Client, f.chain.DeployerKey), nil
}
