package rpcclient

import (
	"testing"

	sollib "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_reachedCommitment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     solrpc.ConfirmationStatusType
		commitment solrpc.CommitmentType
		want       bool
	}{
		{"", solrpc.CommitmentConfirmed, false},
		{solrpc.ConfirmationStatusProcessed, solrpc.CommitmentConfirmed, false},
		{solrpc.ConfirmationStatusConfirmed, solrpc.CommitmentConfirmed, true},
		{solrpc.ConfirmationStatusConfirmed, solrpc.CommitmentFinalized, false},
		{solrpc.ConfirmationStatusFinalized, solrpc.CommitmentFinalized, true},
		{solrpc.ConfirmationStatusFinalized, solrpc.CommitmentConfirmed, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status)+"/"+string(tt.commitment), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, reachedCommitment(tt.status, tt.commitment))
		})
	}
}

func Test_Client_SendAndConfirmTx_RejectsProcessedCommitment(t *testing.T) {
	t.Parallel()

	// No RPC is reachable at this URL, so the call must fail before sending anything.
	client := New(solrpc.New("http://127.0.0.1:1"), sollib.NewWallet().PrivateKey)

	_, err := client.SendAndConfirmTx(t.Context(), nil, WithCommitment(solrpc.CommitmentProcessed))
	require.ErrorContains(t, err, `unsupported commitment "processed"`)
}
