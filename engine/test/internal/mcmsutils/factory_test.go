package mcmsutils

import (
	"fmt"
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/testutils"
)

func TestGetInspectorFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chain    fchain.BlockChain
		wantType any
		wantErr  string
	}{
		{
			name:     "EVM chain success",
			chain:    stubEVMChain(),
			wantType: &evmInspectorFactory{},
		},
		{
			name:     "Solana chain success",
			chain:    stubSolanaChain(),
			wantType: &solanaInspectorFactory{},
		},
		{
			name:    "Aptos chain should fail",
			chain:   stubAptosChain(),
			wantErr: "aptos does not support inspection on non-timelock proposals",
		},
		{
			name:    "unsupported chain family",
			chain:   testutils.NewStubChain(999999),
			wantErr: "chain family not supported: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory, err := GetInspectorFactory(tt.chain)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, factory)
			} else {
				require.NoError(t, err)
				require.NotNil(t, factory)

				// Verify we can create an inspector
				inspector, err := factory.Make()
				require.NoError(t, err)
				assert.NotNil(t, inspector)
				assert.IsType(t, tt.wantType, factory)
			}
		})
	}
}

func TestGetTimelockInspectorFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chain    fchain.BlockChain
		action   mcmstypes.TimelockAction
		wantType any
		wantErr  string
	}{
		{
			name:     "EVM chain",
			chain:    stubEVMChain(),
			action:   mcmstypes.TimelockActionSchedule,
			wantType: &evmInspectorFactory{},
		},
		{
			name:     "Solana chain",
			chain:    stubSolanaChain(),
			action:   mcmstypes.TimelockActionCancel,
			wantType: &solanaInspectorFactory{},
		},
		{
			name:     "Aptos chain",
			chain:    stubAptosChain(),
			action:   mcmstypes.TimelockActionBypass,
			wantType: &aptosInspectorFactory{},
		},
		{
			name:    "unsupported chain family",
			chain:   testutils.NewStubChain(999999),
			action:  mcmstypes.TimelockActionSchedule,
			wantErr: "chain family not supported: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory, err := GetTimelockInspectorFactory(tt.chain, tt.action)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, factory)
			} else {
				require.NoError(t, err)
				require.NotNil(t, factory)

				// Verify we can create an inspector
				inspector, err := factory.Make()
				require.NoError(t, err)
				assert.NotNil(t, inspector)
				assert.IsType(t, tt.wantType, factory)
			}
		})
	}
}

func TestGetConverterFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		family   string
		wantType any
		wantErr  string
	}{
		{
			name:     "EVM chain success",
			family:   chainselectors.FamilyEVM,
			wantType: &evmConverterFactory{},
		},
		{
			name:     "Solana chain success",
			family:   chainselectors.FamilySolana,
			wantType: &solanaConverterFactory{},
		},
		{
			name:     "Aptos chain success",
			family:   chainselectors.FamilyAptos,
			wantType: &aptosConverterFactory{},
		},
		{
			name:    "unsupported chain family",
			family:  "unsupported",
			wantErr: "chain family not supported: unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory, err := GetConverterFactory(tt.family)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, factory)
			} else {
				require.NoError(t, err)
				require.NotNil(t, factory)

				// Verify we can create a converter
				converter, err := factory.Make()
				require.NoError(t, err)
				assert.NotNil(t, converter)
				assert.IsType(t, tt.wantType, factory)
			}
		})
	}
}

func TestGetExecutorFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chain    fchain.BlockChain
		encoder  mcmssdk.Encoder
		wantType any
		wantErr  string
	}{
		{
			name:     "EVM chain with correct encoder",
			chain:    stubEVMChain(),
			encoder:  &mcmsevmsdk.Encoder{},
			wantType: &evmExecutorFactory{},
		},
		{
			name:    "EVM chain with wrong encoder",
			chain:   stubEVMChain(),
			encoder: &mcmssolanasdk.Encoder{},
			wantErr: fmt.Sprintf("encoder not found for chain selector %d", stubEVMChain().Selector),
		},
		{
			name:     "Solana chain with correct encoder",
			chain:    stubSolanaChain(),
			encoder:  &mcmssolanasdk.Encoder{},
			wantType: &solanaExecutorFactory{},
		},
		{
			name:    "Solana chain with wrong encoder",
			chain:   stubSolanaChain(),
			encoder: &mcmsevmsdk.Encoder{},
			wantErr: fmt.Sprintf("encoder not found for chain selector %d", stubSolanaChain().Selector),
		},
		{
			name:     "Aptos chain with correct encoder",
			chain:    stubAptosChain(),
			encoder:  &mcmsaptossdk.Encoder{},
			wantType: &aptosExecutorFactory{},
		},
		{
			name:    "Aptos chain with wrong encoder",
			chain:   stubAptosChain(),
			encoder: &mcmsevmsdk.Encoder{},
			wantErr: fmt.Sprintf("encoder not found for chain selector %d", stubAptosChain().Selector),
		},
		{
			name:    "unsupported chain family",
			chain:   testutils.NewStubChain(999999),
			encoder: &mcmsevmsdk.Encoder{},
			wantErr: "chain family not supported: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory, err := GetExecutorFactory(tt.chain, tt.encoder)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, factory)
			} else {
				require.NoError(t, err)
				require.NotNil(t, factory)

				// Verify we can create an executor
				executor, err := factory.Make()
				require.NoError(t, err)
				assert.NotNil(t, executor)
				assert.IsType(t, tt.wantType, factory)
			}
		})
	}
}

func TestGetTimelockExecutorFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chain    fchain.BlockChain
		wantType any
		wantErr  string
	}{
		{
			name:     "EVM chain success",
			chain:    stubEVMChain(),
			wantType: &evmTimelockExecutorFactory{},
		},
		{
			name:     "Solana chain success",
			chain:    stubSolanaChain(),
			wantType: &solanaTimelockExecutorFactory{},
		},
		{
			name:     "Aptos chain success",
			chain:    stubAptosChain(),
			wantType: &aptosTimelockExecutorFactory{},
		},
		{
			name:    "unsupported chain family",
			chain:   testutils.NewStubChain(999999),
			wantErr: "chain family not supported: test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			factory, err := GetTimelockExecutorFactory(tt.chain)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, factory)
			} else {
				require.NoError(t, err)
				require.NotNil(t, factory)

				// Verify we can create a timelock executor
				executor, err := factory.Make()
				require.NoError(t, err)
				assert.NotNil(t, executor)
			}
		})
	}
}
