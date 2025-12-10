package provider

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_zkSyncSigner_Getters tests the getters of the zkSyncSigner.
func Test_zkSyncSigner_Getters(t *testing.T) {
	t.Parallel()

	s := newZkSyncSigner(testAddr1, testChainIDBig, nil)

	assert.Equal(t, testAddr1, s.Address())
	assert.Equal(t, testChainIDBig, s.ChainID())
	assert.Nil(t, s.PrivateKey())
}

func Test_zkSuncSigner_NotImplementedMethods(t *testing.T) {
	t.Parallel()

	s := newZkSyncSigner(testAddr1, testChainIDBig, nil)

	_, err := s.SignMessage(context.Background(), []byte{})
	require.Error(t, err)
}

func Test_zkSyncSigner_SignTransaction(t *testing.T) {
	t.Parallel()

	s := newZkSyncSigner(testAddr1, testChainIDBig, nil)

	_, err := s.SignMessage(context.Background(), []byte{})
	require.Error(t, err)

	_, err = s.SignTransaction(context.Background(), nil)
	require.Error(t, err)
}

func Test_zkSyncSigner_SignTypedData(t *testing.T) {
	t.Parallel()

	var (
		addr    = testAddr1
		chainID = testChainIDBig

		// Valid typed data
		data = &apitypes.TypedData{
			Types: apitypes.Types{
				"EIP712Domain": []apitypes.Type{
					{Name: "name", Type: "string"},
				},
				"Mail": []apitypes.Type{
					{Name: "contents", Type: "string"},
				},
			},
			PrimaryType: "Mail",
			Domain: apitypes.TypedDataDomain{
				Name: "zkSync",
			},
			Message: map[string]any{
				"contents": "hello",
			},
		}
	)

	// fakeSignHashFunc is a mock function that simulates signing a hash.
	fakeSignHashFunc := func(v int) func(hash []byte) ([]byte, error) {
		return func(hash []byte) ([]byte, error) {
			sig := make([]byte, 65)
			copy(sig, hash)
			sig[64] = byte(v)

			return sig, nil
		}
	}

	tests := []struct {
		name         string
		signHashFunc func(hash []byte) ([]byte, error)
		typedData    *apitypes.TypedData
		wantErr      string
		wantSigLen   int
		wantV        byte
	}{
		{
			name:         "valid signature",
			signHashFunc: fakeSignHashFunc(27),
			typedData:    data,
			wantSigLen:   65,
			wantV:        27,
		},
		{
			name:         "valid signature with v < 27",
			signHashFunc: fakeSignHashFunc(1),
			typedData:    data,
			wantSigLen:   65,
			wantV:        28,
		},
		{
			name: "fails to hash struct",
			typedData: &apitypes.TypedData{
				Types: apitypes.Types{
					"EIP712Domain": []apitypes.Type{
						{Name: "name", Type: "string"},
					},
					"Mail": []apitypes.Type{
						{Name: "contents", Type: "string"},
					},
				},
				Domain: apitypes.TypedDataDomain{
					Name: "zkSync",
				},
				Message: map[string]any{
					"contents": "hello",
				},
			},
			wantErr: "failed to get hash of typed message",
		},
		{
			name: "signHash returns error",
			signHashFunc: func(hash []byte) ([]byte, error) {
				return nil, assert.AnError
			},
			typedData: data,
			wantErr:   "failed to sign hash of typed data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newZkSyncSigner(addr, chainID, tt.signHashFunc)
			sig, err := s.SignTypedData(t.Context(), tt.typedData)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Len(t, sig, tt.wantSigLen)
				assert.Equal(t, tt.wantV, sig[64])
			}
		})
	}
}
