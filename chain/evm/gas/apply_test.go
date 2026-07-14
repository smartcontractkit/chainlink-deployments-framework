package gas_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
)

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		client     gas.Client
		cfg        gas.Config
		setupMock  func(t *testing.T) gas.Client
		wantErr    string
		assertOpts func(t *testing.T, opts *bind.TransactOpts)
	}{
		{
			name: "legacy chain",
			cfg: gas.Config{
				DefaultGasPriceWei: 50_000_000_000,
				DefaultGasLimit:    5_000_000,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: nil}, nil)

				return mockClient
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Equal(t, big.NewInt(50_000_000_000), opts.GasPrice)
				require.Equal(t, uint64(5_000_000), opts.GasLimit)
				require.Nil(t, opts.GasTipCap)
				require.Nil(t, opts.GasFeeCap)
			},
		},
		{
			name: "EIP-1559 chain",
			cfg: gas.Config{
				DefaultGasPriceWei: 50_000_000_000,
				DefaultGasLimit:    5_000_000,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: big.NewInt(100)}, nil)
				mockClient.EXPECT().SuggestGasTipCap(mock.Anything).
					Return(big.NewInt(1_000_000_000), nil)

				return mockClient
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Nil(t, opts.GasPrice)
				require.Equal(t, big.NewInt(1_000_000_000), opts.GasTipCap)
				require.Equal(t, big.NewInt(50_000_000_000), opts.GasFeeCap)
				require.Equal(t, uint64(5_000_000), opts.GasLimit)
			},
		},
		{
			name: "gas limit only",
			cfg:  gas.Config{DefaultGasLimit: 10_000_000},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				return evm.NewMockOnchainClient(t)
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Equal(t, uint64(10_000_000), opts.GasLimit)
				require.Nil(t, opts.GasPrice)
			},
		},
		{
			name: "caps default gas limit at max tx gas limit",
			cfg: gas.Config{
				DefaultGasLimit: 20_000_000,
				MaxTxGasLimit:   gas.EIP7825MaxTxGasLimit,
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Equal(t, gas.EIP7825MaxTxGasLimit, opts.GasLimit)
			},
		},
		{
			name: "EIP-1559 with tip override",
			cfg: gas.Config{
				DefaultGasPriceWei:  100_000_000_000,
				DefaultGasTipCapWei: 40_000_000_000,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: big.NewInt(100)}, nil)

				return mockClient
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Nil(t, opts.GasPrice)
				require.Equal(t, big.NewInt(40_000_000_000), opts.GasTipCap)
				require.Equal(t, big.NewInt(100_000_000_000), opts.GasFeeCap)
			},
		},
		{
			name: "EIP-1559 caps tip override to fee cap minus base fee",
			cfg: gas.Config{
				DefaultGasPriceWei:  50_000_000_000,
				DefaultGasTipCapWei: 100_000_000_000,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: big.NewInt(100)}, nil)

				return mockClient
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Nil(t, opts.GasPrice)
				require.Equal(t, big.NewInt(49_999_999_900), opts.GasTipCap)
				require.Equal(t, big.NewInt(50_000_000_000), opts.GasFeeCap)
			},
		},
		{
			name: "EIP-1559 caps suggested tip to fee cap minus base fee",
			cfg: gas.Config{
				DefaultGasPriceWei: 50_000_000_000,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: big.NewInt(100)}, nil)
				mockClient.EXPECT().SuggestGasTipCap(mock.Anything).
					Return(big.NewInt(100_000_000_000), nil)

				return mockClient
			},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Equal(t, big.NewInt(49_999_999_900), opts.GasTipCap)
				require.Equal(t, big.NewInt(50_000_000_000), opts.GasFeeCap)
			},
		},
		{
			name: "EIP-1559 rejects fee cap below base fee",
			cfg: gas.Config{
				DefaultGasPriceWei: 50,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: big.NewInt(100)}, nil)
				mockClient.EXPECT().SuggestGasTipCap(mock.Anything).
					Return(big.NewInt(1), nil)

				return mockClient
			},
			wantErr: "below current base fee",
		},
		{
			name: "EIP-1559 rejects nil suggested tip",
			cfg: gas.Config{
				DefaultGasPriceWei: 50_000_000_000,
			},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(&types.Header{BaseFee: big.NewInt(100)}, nil)
				mockClient.EXPECT().SuggestGasTipCap(mock.Anything).
					Return(nil, nil)

				return mockClient
			},
			wantErr: "gas tip cap is nil",
		},
		{
			name:    "nil client requires RPC for price",
			client:  nil,
			cfg:     gas.Config{DefaultGasPriceWei: 50_000_000_000},
			wantErr: "gas client is required",
		},
		{
			name: "nil header",
			cfg:  gas.Config{DefaultGasPriceWei: 50_000_000_000},
			setupMock: func(t *testing.T) gas.Client {
				t.Helper()
				mockClient := evm.NewMockOnchainClient(t)
				mockClient.EXPECT().HeaderByNumber(mock.Anything, (*big.Int)(nil)).
					Return(nil, nil)

				return mockClient
			},
			wantErr: "latest block header is nil",
		},
		{
			name:   "nil client allows limit only",
			client: nil,
			cfg:    gas.Config{DefaultGasLimit: 10_000_000},
			assertOpts: func(t *testing.T, opts *bind.TransactOpts) {
				t.Helper()
				require.Equal(t, uint64(10_000_000), opts.GasLimit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := tt.client
			if tt.setupMock != nil {
				client = tt.setupMock(t)
			}

			opts := &bind.TransactOpts{}
			err := gas.ApplyDefaults(t.Context(), client, opts, tt.cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			if tt.assertOpts != nil {
				tt.assertOpts(t, opts)
			}
		})
	}
}

func TestIsEIP1559Header(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header *types.Header
		want   bool
	}{
		{name: "nil header", header: nil},
		{name: "legacy header", header: &types.Header{}},
		{name: "EIP-1559 header", header: &types.Header{BaseFee: big.NewInt(1)}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, gas.IsEIP1559Header(tt.header))
		})
	}
}
