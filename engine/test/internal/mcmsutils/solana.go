package mcmsutils

import (
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"

	fchainsolana "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
)

//------------------------------------------------------------------------------

var _ InspectorFactory = &solanaInspectorFactory{}

// solanaInspectorFactory is a factory for creating Solana-specific MCMS inspectors.
// It implements the InspectorFactory interface and is responsible for creating
// inspectors that can examine the state of MCMS and Timelock contracts on the Solana blockchain.
type solanaInspectorFactory struct {
	chain fchainsolana.Chain // The Solana chain configuration and client
}

// newSolanaInspectorFactory creates a new Solana inspector factory.
func newSolanaInspectorFactory(chain fchainsolana.Chain) *solanaInspectorFactory {
	return &solanaInspectorFactory{chain: chain}
}

// Make creates and returns a new Solana MCMS inspector.
func (f *solanaInspectorFactory) Make() (mcmssdk.Inspector, error) {
	return mcmssolanasdk.NewInspector(f.chain.Client), nil
}

//------------------------------------------------------------------------------

var _ ConverterFactory = &solanaConverterFactory{}

// solanaConverterFactory is a factory for creating Solana-specific timelock converters.
// It implements the ConverterFactory interface and creates converters that can
// transform MCMS timelock proposals into a standard MCMS proposal.
type solanaConverterFactory struct{}

// newSolanaConverterFactory creates a new Solana converter factory.
func newSolanaConverterFactory() *solanaConverterFactory {
	return &solanaConverterFactory{}
}

// Make creates and returns a new Solana timelock converter.
func (f *solanaConverterFactory) Make() (mcmssdk.TimelockConverter, error) {
	return &mcmssolanasdk.TimelockConverter{}, nil
}

//------------------------------------------------------------------------------

var _ ExecutorFactory = &solanaExecutorFactory{}

// solanaExecutorFactory is a factory for creating Solana-specific MCMS executors.
// It implements the ExecutorFactory interface and creates executors that can
// execute MCMS operations on the Solana blockchain.
type solanaExecutorFactory struct {
	chain   fchainsolana.Chain     // The Solana chain configuration and client
	encoder *mcmssolanasdk.Encoder // The encoder for creating Solana-specific transaction data
}

// newSolanaExecutorFactory creates a new Solana executor factory.
func newSolanaExecutorFactory(
	chain fchainsolana.Chain, encoder *mcmssolanasdk.Encoder,
) *solanaExecutorFactory {
	return &solanaExecutorFactory{
		chain:   chain,
		encoder: encoder,
	}
}

// Make creates and returns a new Solana MCMS executor.
func (f *solanaExecutorFactory) Make() (mcmssdk.Executor, error) {
	return mcmssolanasdk.NewExecutor(f.encoder, f.chain.Client, *f.chain.DeployerKey), nil
}

//------------------------------------------------------------------------------

var _ TimelockExecutorFactory = &solanaTimelockExecutorFactory{}

// solanaTimelockExecutorFactory is a factory for creating Solana-specific timelock executors.
// It implements the TimelockExecutorFactory interface and creates executors specifically
// designed for executing Timelock operations on the Solana blockchain.
type solanaTimelockExecutorFactory struct {
	chain fchainsolana.Chain // The Solana chain configuration and client
}

// newSolanaTimelockExecutorFactory creates a new Solana timelock executor factory.
func newSolanaTimelockExecutorFactory(chain fchainsolana.Chain) *solanaTimelockExecutorFactory {
	return &solanaTimelockExecutorFactory{chain: chain}
}

// Make creates and returns a new Solana timelock executor.
func (f *solanaTimelockExecutorFactory) Make() (mcmssdk.TimelockExecutor, error) {
	return mcmssolanasdk.NewTimelockExecutor(f.chain.Client, *f.chain.DeployerKey), nil
}
