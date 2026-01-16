package provider

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
)

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  RPCChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: RPCChainProviderConfig{
				Participants: []blockchain.CantonParticipantEndpoints{
					{},
				},
			},
		},
		{
			name:    "empty participants",
			config:  RPCChainProviderConfig{},
			wantErr: "no participants specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("validate() error = %v, wantErr nil", err)
			}
		})
	}
}

func Test_RPCChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   RPCChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainsel.CANTON_LOCALNET.Selector,
			giveConfig: RPCChainProviderConfig{
				Participants: []blockchain.CantonParticipantEndpoints{
					{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewRPCChainProvider(tt.giveSelector, tt.giveConfig)
			chain, err := provider.Initialize(nil)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Initialize() error = %v, wantErr nil", err)
				return
			}
			if chain == nil {
				t.Errorf("Initialize() returned nil chain, want non-nil")
			}
		})
	}
}
