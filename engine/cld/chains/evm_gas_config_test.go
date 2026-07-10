package chains

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	fevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	evmclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func TestBuiltInEVMGasConfig(t *testing.T) {
	t.Parallel()

	eip7825 := fevm.EIP7825MaxTxGasLimit
	tests := []struct {
		name      string
		selectors []uint64
		want      evmGasConfig
	}{
		{"buffered estimates", []uint64{
			chainsel.ETHEREUM_MAINNET_BASE_1.Selector,
			chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector,
			chainsel.ETHEREUM_MAINNET_OPTIMISM_1.Selector,
			chainsel.ETHEREUM_TESTNET_SEPOLIA_OPTIMISM_1.Selector,
		}, evmGasConfig{gasLimitBufferBps: estimateGasBufferBps, maxTxGasLimit: eip7825}},
		{"hedera", []uint64{
			chainsel.HEDERA_MAINNET.Selector,
			chainsel.HEDERA_TESTNET.Selector,
		}, evmGasConfig{deployerGasLimit: 10_000_000, deployerGasPrice: hederaDeployerGasPriceWei}},
		{"bob", []uint64{
			chainsel.BITCOIN_MAINNET_BOB_1.Selector,
			chainsel.BITCOIN_TESTNET_SEPOLIA_BOB_1.Selector,
		}, evmGasConfig{maxTxGasLimit: eip7825, deployerGasLimit: 7_500_000, deployerGasPrice: 2_000_000}},
		{"wemix mainnet", []uint64{chainsel.WEMIX_MAINNET.Selector},
			evmGasConfig{deployerGasLimit: 10_000_000, deployerGasPrice: 120_000_000_000}},
		{"wemix testnet", []uint64{chainsel.WEMIX_TESTNET.Selector},
			evmGasConfig{deployerGasLimit: 10_000_000, deployerGasPrice: 120_000_000_000}},
		{"megaeth", []uint64{
			chainsel.MEGAETH_MAINNET.Selector,
			chainsel.MEGAETH_TESTNET.Selector,
			chainsel.MEGAETH_TESTNET_2.Selector,
		}, evmGasConfig{deployerGasLimit: 400_000_000, deployerGasPrice: 1_000_000}},
		{"edge", []uint64{
			chainsel.EDGE_MAINNET.Selector,
			chainsel.EDGE_TESTNET.Selector,
		}, evmGasConfig{deployerGasLimit: 25_000_000}},
		{"bittensor", []uint64{
			chainsel.BITTENSOR_MAINNET.Selector,
			chainsel.BITTENSOR_TESTNET.Selector,
		}, evmGasConfig{deployerGasLimit: 10_000_000, deployerGasPrice: 10_000_000_000}},
		{"mind", []uint64{chainsel.MIND_MAINNET.Selector},
			evmGasConfig{deployerGasLimit: 8_000_000}},
		{"ronin mainnet", []uint64{chainsel.RONIN_MAINNET.Selector},
			evmGasConfig{deployerGasPrice: 100_000_000_000}},
		{"ronin testnets", []uint64{
			chainsel.RONIN_TESTNET_SAIGON.Selector,
			chainsel.ETHEREUM_TESTNET_SEPOLIA_RONIN_1.Selector,
		}, evmGasConfig{deployerGasPrice: 50_000_000_000}},
		{"gnosis chiado", []uint64{chainsel.GNOSIS_CHAIN_TESTNET_CHIADO.Selector},
			evmGasConfig{deployerGasLimit: 10_000_000}},
		{"ink sepolia", []uint64{chainsel.INK_TESTNET_SEPOLIA.Selector},
			evmGasConfig{maxTxGasLimit: eip7825, deployerGasLimit: 7_500_000}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, selector := range tt.selectors {
				require.Equal(t, tt.want, builtInEVMGasConfig(selector))
			}
		})
	}

	require.Equal(t, evmGasConfig{}, builtInEVMGasConfig(12345))
}

func TestEVMClientOptsFromGasConfig(t *testing.T) {
	t.Parallel()

	mc := &evmclient.MultiClient{}
	for _, opt := range evmClientOptsFromGasConfig(withEIP7825Cap(evmGasConfig{
		gasLimitBufferBps: estimateGasBufferBps,
	})) {
		opt(mc)
	}

	require.Equal(t, estimateGasBufferBps, mc.GasLimitBufferBps())
	require.Equal(t, fevm.EIP7825MaxTxGasLimit, mc.MaxTxGasLimit())
}

func TestEVMSignerWithGasConfig(t *testing.T) {
	t.Parallel()

	gen := evmSignerWithGasConfig(stubSignerGenerator{}, evmGasConfig{
		maxTxGasLimit:    fevm.EIP7825MaxTxGasLimit,
		deployerGasLimit: 25_000_000,
		deployerGasPrice: 5_000_000,
	})

	opts, err := gen.Generate(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, fevm.EIP7825MaxTxGasLimit, opts.GasLimit)
	require.Equal(t, uint64(5_000_000), opts.GasPrice.Uint64())
	require.Nil(t, opts.GasFeeCap)
	require.Nil(t, opts.GasTipCap)
}

type stubSignerGenerator struct{}

func (stubSignerGenerator) Generate(_ *big.Int) (*bind.TransactOpts, error) {
	return &bind.TransactOpts{}, nil
}

func (stubSignerGenerator) SignHash(_ []byte) ([]byte, error) {
	return nil, nil
}
