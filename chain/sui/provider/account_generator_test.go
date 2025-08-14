package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			wantErr:        "hex private key must be exactly 64 characters",
		},
		{
			name:           "empty private key",
			givePrivateKey: "",
			wantErr:        "hex private key must be exactly 64 characters",
		},
		{
			name:           "invalid hex private key",
			givePrivateKey: "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
			wantErr:        "encoding/hex: invalid byte",
		},
		{
			name:           "wrong length private key",
			givePrivateKey: "E4FD0E90D32CB98DC6AD64516A421E8C",
			wantErr:        "hex private key must be exactly 64 characters",
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

				// Verify the account generates the expected address
				actualAddr, err := account.GetAddress()
				require.NoError(t, err)

				assert.Equal(t, tt.wantAddr, actualAddr)
			}
		})
	}
}
