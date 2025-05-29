package provider

import (
	"testing"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AccountGenCtfDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		giveAccountAddr string
		givePrivateKey  string
		wantErr         string
	}{
		{
			name: "valid default account and private key",
		},
		{
			name:            "invalid account address",
			giveAccountAddr: "invalid_address",
			givePrivateKey:  blockchain.DefaultAptosPrivateKey,
			wantErr:         "failed to parse account address",
		},
		{
			name:            "invalid private key",
			giveAccountAddr: blockchain.DefaultAptosAccount,
			givePrivateKey:  "invalid_private_key",
			wantErr:         "failed to decode private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gen *accountGenCTFDefault
			if tt.giveAccountAddr == "" && tt.givePrivateKey == "" {
				gen = AccountGenCTFDefault() // Use the constructor with default values if we don't need to override
			} else {
				gen = &accountGenCTFDefault{
					accountStr:    tt.giveAccountAddr,
					privateKeyStr: tt.givePrivateKey,
				}
			}

			got, err := gen.Generate()
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, blockchain.DefaultAptosAccount, got.Address.String())
			}
		})
	}
}

func Test_AccountGenNewSingleSender(t *testing.T) {
	t.Parallel()

	gen := AccountGenNewSingleSender()
	got, err := gen.Generate()
	require.NoError(t, err)
	assert.NotNil(t, got)
}

func Test_AccountGenPrivateKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		givePrivateKey string
		wantAddr       string
		wantErr        string
	}{
		{
			name:           "valid private key",
			givePrivateKey: testPrivateKey,
			wantAddr:       testAccountAddr,
		},
		{
			name:           "invalid private key",
			givePrivateKey: "invalid_private_key",
			wantErr:        "failed to parse private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := AccountGenPrivateKey(tt.givePrivateKey)
			account, err := gen.Generate()
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, account)
				assert.Equal(t, tt.wantAddr, account.Address.String())
			}
		})
	}
}
