package upf

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go-stellar-sdk/xdr"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-stellar/bindings"
	"github.com/smartcontractkit/chainlink-stellar/bindings/scval"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	mcmsanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

const (
	stellarProposerMCM = "CB66PWVWBA765OSEMEPL3766WXEFPVGVMVS5UHNPEP266Y6QZGFJQG4B"
	stellarTimelock    = "CAZPM2APSKKYZEGGS3E3V6MONKF2ZQZSZVCFSZOSDYPQK3XQ2XUO2LQW"

	stellarAdditionalFields = `{"family":"stellar","encodingVersion":1}`
)

// stellarTx builds a Stellar MCMS transaction carrying a Soroban invoke payload.
func stellarTx(t *testing.T, function string, args ...xdr.ScVal) mcmstypes.Transaction {
	t.Helper()

	data, err := bindings.EncodeSorobanInvokePayload(function, args)
	require.NoError(t, err)

	return mcmstypes.Transaction{
		To:               stellarProposerMCM,
		Data:             data,
		AdditionalFields: json.RawMessage(stellarAdditionalFields),
	}
}

// stellarBadTx builds a Stellar MCMS transaction whose payload is not a valid invoke payload.
func stellarBadTx() mcmstypes.Transaction {
	return mcmstypes.Transaction{
		To:               stellarProposerMCM,
		Data:             []byte{0x01, 0x02, 0x03},
		AdditionalFields: json.RawMessage(stellarAdditionalFields),
	}
}

func stellarProposalCtx() mcmsanalyzer.ProposalContext {
	return &mcmsanalyzer.DefaultProposalContext{AddressesByChain: deployment.AddressesByChain{}}
}

// TestBatchOperationsToUpfDecodedCalls_Stellar proves the UPF batch-decode path produces decoded
// Stellar calls (previously the default branch emitted "stellar transaction decoding is not
// supported" for every transaction in the batch).
func TestBatchOperationsToUpfDecodedCalls_Stellar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		txs       []mcmstypes.Transaction
		wantCalls []string
	}{
		{
			name:      "single no-arg call",
			txs:       []mcmstypes.Transaction{stellarTx(t, "accept_ownership")},
			wantCalls: []string{"accept_ownership"},
		},
		{
			name: "every transaction in the batch is decoded",
			txs: []mcmstypes.Transaction{
				stellarTx(t, "accept_ownership"),
				stellarTx(t, "transfer_ownership", scval.AddressToScVal(stellarTimelock)),
			},
			wantCalls: []string{"accept_ownership", "transfer_ownership"},
		},
		{
			name:      "a malformed payload degrades that call only",
			txs:       []mcmstypes.Transaction{stellarBadTx(), stellarTx(t, "accept_ownership")},
			wantCalls: []string{"failed to decode Stellar transaction", "accept_ownership"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			batches := []mcmstypes.BatchOperation{{
				ChainSelector: mcmstypes.ChainSelector(chainsel.STELLAR_TESTNET.Selector),
				Transactions:  tt.txs,
			}}

			decoded, err := batchOperationsToUpfDecodedCalls(
				t.Context(), stellarProposalCtx(), deployment.Environment{}, batches,
			)
			require.NoError(t, err)
			require.Len(t, decoded, 1)
			require.Len(t, decoded[0], len(tt.wantCalls))

			for i, want := range tt.wantCalls {
				require.NotNil(t, decoded[0][i].Data)
				require.Contains(t, decoded[0][i].Data.FunctionName, want)
				require.Equal(t, stellarProposerMCM, decoded[0][i].To)
			}
		})
	}
}

// TestAnalyzeTransaction_Stellar covers the single-operation UPF path, which previously returned
// "unsupported chain family: stellar".
func TestAnalyzeTransaction_Stellar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tx         mcmstypes.Transaction
		wantMethod string
	}{
		{
			name:       "no-arg call",
			tx:         stellarTx(t, "accept_ownership"),
			wantMethod: "accept_ownership",
		},
		{
			name:       "call with arguments",
			tx:         stellarTx(t, "transfer_ownership", scval.AddressToScVal(stellarTimelock)),
			wantMethod: "transfer_ownership",
		},
		{
			name:       "malformed payload is reported in the method",
			tx:         stellarBadTx(),
			wantMethod: "failed to decode Stellar transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op := mcmstypes.Operation{
				ChainSelector: mcmstypes.ChainSelector(chainsel.STELLAR_TESTNET.Selector),
				Transaction:   tt.tx,
			}

			decoded, abi, err := analyzeTransaction(
				t.Context(), stellarProposalCtx(), deployment.Environment{}, op,
			)
			require.NoError(t, err)
			require.Empty(t, abi)
			require.NotNil(t, decoded)
			require.Contains(t, decoded.Method, tt.wantMethod)
			require.Equal(t, stellarProposerMCM, decoded.Address)
		})
	}
}

// TestEncodeTransactionData_Stellar pins the data encoding for Stellar: the non-EVM default is
// base64, matching the `data` field in the proposal JSON.
func TestEncodeTransactionData_Stellar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tx   mcmstypes.Transaction
		want string
	}{
		{
			name: "no-arg call",
			tx:   stellarTx(t, "accept_ownership"),
			want: "AAAAEAAAAAEAAAABAAAADwAAABBhY2NlcHRfb3duZXJzaGlw",
		},
		{
			name: "call with an address argument",
			tx:   stellarTx(t, "transfer_ownership", scval.AddressToScVal(stellarTimelock)),
			want: "AAAAEAAAAAEAAAACAAAADwAAABJ0cmFuc2Zlcl9vd25lcnNoaXAAAAAAABIAAAABMvZoD5KVjJDGlsm6+Y5qi6zDMs1EWWXSHh8FbvDV6O0=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op := mcmstypes.Operation{
				ChainSelector: mcmstypes.ChainSelector(chainsel.STELLAR_TESTNET.Selector),
				Transaction:   tt.tx,
			}

			encoded, err := encodeTransactionData(op)
			require.NoError(t, err)
			require.Equal(t, tt.want, encoded)
		})
	}
}
