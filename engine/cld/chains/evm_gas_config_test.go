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

func Test_builtInEVMGasConfig(t *testing.T) {
	t.Parallel()

	t.Run("base mainnet buffer", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(chainsel.ETHEREUM_MAINNET_BASE_1.Selector)
		require.Equal(t, BaseGasLimitBufferBps, cfg.gasLimitBufferBps)
		require.Equal(t, fevm.EIP7825MaxTxGasLimit, cfg.maxTxGasLimit)
		require.Equal(t, uint64(0), cfg.deployerGasLimit)
	})

	t.Run("base sepolia buffer", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector)
		require.Equal(t, BaseGasLimitBufferBps, cfg.gasLimitBufferBps)
	})

	t.Run("optimism mainnet buffer", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(chainsel.ETHEREUM_MAINNET_OPTIMISM_1.Selector)
		require.Equal(t, BaseGasLimitBufferBps, cfg.gasLimitBufferBps)
		require.Equal(t, fevm.EIP7825MaxTxGasLimit, cfg.maxTxGasLimit)
		require.Equal(t, uint64(0), cfg.deployerGasLimit)
	})

	t.Run("optimism sepolia buffer", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(chainsel.ETHEREUM_TESTNET_SEPOLIA_OPTIMISM_1.Selector)
		require.Equal(t, BaseGasLimitBufferBps, cfg.gasLimitBufferBps)
	})

	t.Run("hedera fixed gas", func(t *testing.T) {
		t.Parallel()

		for _, selector := range []uint64{
			chainsel.HEDERA_MAINNET.Selector,
			chainsel.HEDERA_TESTNET.Selector,
		} {
			cfg := builtInEVMGasConfig(selector)
			require.Equal(t, uint64(10_000_000), cfg.deployerGasLimit)
			require.Equal(t, HederaDeployerGasPriceWei, *cfg.deployerGasPrice)
			require.Equal(t, uint64(0), cfg.gasLimitBufferBps)
		}
	})

	t.Run("edge fixed gas limit only", func(t *testing.T) {
		t.Parallel()

		for _, selector := range []uint64{
			chainsel.EDGE_MAINNET.Selector,
			chainsel.EDGE_TESTNET.Selector,
		} {
			cfg := builtInEVMGasConfig(selector)
			require.Equal(t, uint64(25_000_000), cfg.deployerGasLimit)
			require.Nil(t, cfg.deployerGasPrice)
		}
	})

	t.Run("bittensor fixed gas", func(t *testing.T) {
		t.Parallel()

		for _, selector := range []uint64{
			chainsel.BITTENSOR_MAINNET.Selector,
			chainsel.BITTENSOR_TESTNET.Selector,
		} {
			cfg := builtInEVMGasConfig(selector)
			require.Equal(t, uint64(10_000_000), cfg.deployerGasLimit)
			require.Equal(t, uint64(10_000_000_000), *cfg.deployerGasPrice)
		}
	})

	t.Run("mind fixed gas limit only", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(chainsel.MIND_MAINNET.Selector)
		require.Equal(t, uint64(8_000_000), cfg.deployerGasLimit)
		require.Nil(t, cfg.deployerGasPrice)
	})

	t.Run("ronin mainnet legacy gas price", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(chainsel.RONIN_MAINNET.Selector)
		require.Equal(t, uint64(100_000_000_000), *cfg.deployerGasPrice)
	})

	t.Run("ronin testnets legacy gas price", func(t *testing.T) {
		t.Parallel()

		for _, selector := range []uint64{
			chainsel.RONIN_TESTNET_SAIGON.Selector,
			chainsel.ETHEREUM_TESTNET_SEPOLIA_RONIN_1.Selector,
		} {
			cfg := builtInEVMGasConfig(selector)
			require.Equal(t, uint64(50_000_000_000), *cfg.deployerGasPrice)
		}
	})

	t.Run("testnet-only gas overrides", func(t *testing.T) {
		t.Parallel()

		gnosis := builtInEVMGasConfig(chainsel.GNOSIS_CHAIN_TESTNET_CHIADO.Selector)
		require.Equal(t, uint64(10_000_000), gnosis.deployerGasLimit)
		require.Equal(t, uint64(0), gnosis.maxTxGasLimit)

		ink := builtInEVMGasConfig(chainsel.INK_TESTNET_SEPOLIA.Selector)
		require.Equal(t, uint64(7_500_000), ink.deployerGasLimit)
		require.Equal(t, fevm.EIP7825MaxTxGasLimit, ink.maxTxGasLimit)

		zora := builtInEVMGasConfig(chainsel.ZORA_TESTNET.Selector)
		require.Equal(t, uint64(7_500_000), zora.deployerGasLimit)
		require.Equal(t, fevm.EIP7825MaxTxGasLimit, zora.maxTxGasLimit)
	})

	t.Run("eip-7825 cap on op stack fixed overrides", func(t *testing.T) {
		t.Parallel()

		for _, selector := range []uint64{
			chainsel.METAL_MAINNET.Selector,
			chainsel.BITCOIN_MAINNET_BOB_1.Selector,
		} {
			cfg := builtInEVMGasConfig(selector)
			require.Equal(t, fevm.EIP7825MaxTxGasLimit, cfg.maxTxGasLimit)
		}
	})

	t.Run("non eip-7825 overrides have no tx gas cap", func(t *testing.T) {
		t.Parallel()

		for _, selector := range []uint64{
			chainsel.HEDERA_MAINNET.Selector,
			chainsel.EDGE_MAINNET.Selector,
			chainsel.MEGAETH_MAINNET.Selector,
			chainsel.MIND_MAINNET.Selector,
		} {
			cfg := builtInEVMGasConfig(selector)
			require.Equal(t, uint64(0), cfg.maxTxGasLimit)
		}
	})

	t.Run("applyEIP7825GasCapIfConfigured caps high fixed limits", func(t *testing.T) {
		t.Parallel()

		cfg := applyEIP7825GasCapIfConfigured(
			chainsel.METAL_MAINNET.Selector,
			evmGasConfig{deployerGasLimit: 25_000_000},
		)
		require.Equal(t, fevm.EIP7825MaxTxGasLimit, cfg.maxTxGasLimit)
		require.Equal(t, fevm.EIP7825MaxTxGasLimit, cfg.deployerGasLimit)
	})

	t.Run("testnets match mainnet gas overrides", func(t *testing.T) {
		t.Parallel()

		pairs := []struct {
			mainnet uint64
			testnet uint64
		}{
			{chainsel.METAL_MAINNET.Selector, chainsel.METAL_TESTNET.Selector},
			{chainsel.BITCOIN_MAINNET_BOB_1.Selector, chainsel.BITCOIN_TESTNET_SEPOLIA_BOB_1.Selector},
			{chainsel.MEGAETH_MAINNET.Selector, chainsel.MEGAETH_TESTNET.Selector},
			{chainsel.BITTENSOR_MAINNET.Selector, chainsel.BITTENSOR_TESTNET.Selector},
		}

		for _, pair := range pairs {
			require.Equal(t, builtInEVMGasConfig(pair.mainnet), builtInEVMGasConfig(pair.testnet))
		}

		require.Equal(t,
			builtInEVMGasConfig(chainsel.MEGAETH_MAINNET.Selector),
			builtInEVMGasConfig(chainsel.MEGAETH_TESTNET_2.Selector),
		)
	})

	t.Run("wemix testnet uses higher gas limit", func(t *testing.T) {
		t.Parallel()

		mainnet := builtInEVMGasConfig(chainsel.WEMIX_MAINNET.Selector)
		testnet := builtInEVMGasConfig(chainsel.WEMIX_TESTNET.Selector)

		require.Equal(t, uint64(7_500_000), mainnet.deployerGasLimit)
		require.Equal(t, uint64(10_000_000), testnet.deployerGasLimit)
		require.Equal(t, *mainnet.deployerGasPrice, *testnet.deployerGasPrice)
	})

	t.Run("unknown chain has no defaults", func(t *testing.T) {
		t.Parallel()

		cfg := builtInEVMGasConfig(12345)
		require.Equal(t, evmGasConfig{}, cfg)
	})
}

func Test_evmClientOptsFromGasConfig(t *testing.T) {
	t.Parallel()

	opts := evmClientOptsFromGasConfig(builtInEVMGasConfig(chainsel.ETHEREUM_MAINNET_BASE_1.Selector))
	require.Len(t, opts, 2)

	mc := &evmclient.MultiClient{}
	opts[0](mc)
	opts[1](mc)
	require.Equal(t, uint64(2500), mc.GasLimitBufferBps())
	require.Equal(t, fevm.EIP7825MaxTxGasLimit, mc.MaxTxGasLimit())
}

func Test_evmSignerWithGasConfig(t *testing.T) {
	t.Parallel()

	gen := evmSignerWithGasConfig(
		stubSignerGenerator{},
		builtInEVMGasConfig(chainsel.METAL_MAINNET.Selector),
	)

	opts, err := gen.Generate(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, uint64(10_000_000), opts.GasLimit)
	require.Equal(t, uint64(5_000_000), opts.GasPrice.Uint64())
	require.Nil(t, opts.GasFeeCap)
	require.Nil(t, opts.GasTipCap)
}

func Test_evmSignerWithGasConfig_capsFixedLimitAtEIP7825(t *testing.T) {
	t.Parallel()

	gen := evmSignerWithGasConfig(
		stubSignerGenerator{},
		applyEIP7825GasCapIfConfigured(
			chainsel.METAL_MAINNET.Selector,
			evmGasConfig{deployerGasLimit: 25_000_000},
		),
	)

	opts, err := gen.Generate(big.NewInt(1))
	require.NoError(t, err)
	require.Equal(t, fevm.EIP7825MaxTxGasLimit, opts.GasLimit)
}

type stubSignerGenerator struct{}

func (stubSignerGenerator) Generate(_ *big.Int) (*bind.TransactOpts, error) {
	return &bind.TransactOpts{}, nil
}

func (stubSignerGenerator) SignHash(_ []byte) ([]byte, error) {
	return nil, nil
}
