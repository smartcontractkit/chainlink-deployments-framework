package mcmsutils

import (
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"

	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

//------------------------------------------------------------------------------

var _ InspectorFactory = &solanaInspectorFactory{}

type solanaInspectorFactory struct {
	chain fchainsolana.Chain
}

func newSolanaInspectorFactory(chain fchainsolana.Chain) *solanaInspectorFactory {
	return &solanaInspectorFactory{chain: chain}
}

func (f *solanaInspectorFactory) Make() (mcmssdk.Inspector, error) {
	return mcmssolanasdk.NewInspector(f.chain.Client), nil
}

//------------------------------------------------------------------------------

var _ ConverterFactory = &solanaConverterFactory{}

type solanaConverterFactory struct{}

func newSolanaConverterFactory() *solanaConverterFactory {
	return &solanaConverterFactory{}
}

func (f *solanaConverterFactory) Make() (mcmssdk.TimelockConverter, error) {
	return &mcmssolanasdk.TimelockConverter{}, nil
}

//------------------------------------------------------------------------------

var _ ExecutorFactory = &solanaExecutorFactory{}

type solanaExecutorFactory struct {
	chain   fchainsolana.Chain
	encoder *mcmssolanasdk.Encoder
}

func newSolanaExecutorFactory(
	chain fchainsolana.Chain, encoder *mcmssolanasdk.Encoder,
) *solanaExecutorFactory {
	return &solanaExecutorFactory{
		chain:   chain,
		encoder: encoder,
	}
}

func (f *solanaExecutorFactory) Make() (mcmssdk.Executor, error) {
	return mcmssolanasdk.NewExecutor(f.encoder, f.chain.Client, *f.chain.DeployerKey), nil
}

//------------------------------------------------------------------------------

var _ TimelockExecutorFactory = &solanaTimelockExecutorFactory{}

type solanaTimelockExecutorFactory struct {
	chain fchainsolana.Chain
}

func newSolanaTimelockExecutorFactory(chain fchainsolana.Chain) *solanaTimelockExecutorFactory {
	return &solanaTimelockExecutorFactory{chain: chain}
}

func (f *solanaTimelockExecutorFactory) Make() (mcmssdk.TimelockExecutor, error) {
	return mcmssolanasdk.NewTimelockExecutor(f.chain.Client, *f.chain.DeployerKey), nil
}
