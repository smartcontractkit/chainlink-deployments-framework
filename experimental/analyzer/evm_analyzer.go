package analyzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/types"
)

func AnalyzeEVMTransactions(ctx ProposalContext, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
	chainFamily, err := chainsel.GetSelectorFamily(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain family for selector %v: %w", chainSelector, err)
	}
	if chainFamily != chainsel.FamilyEVM {
		return nil, fmt.Errorf("unsupported chain family (%v)", chainFamily)
	}

	decoder := NewTxCallDecoder(nil)

	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		decodedTxs[i], _, _, err = AnalyzeEVMTransaction(ctx, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze transaction %d: %w", i, err)
		}
	}

	return decodedTxs, nil
}

func AnalyzeEVMTransaction(
	ctx ProposalContext, decoder *EVMTxCallDecoder, chainSelector uint64, mcmsTx types.Transaction,
) (*DecodedCall, *abi.ABI, string, error) {
	// Check if this is a native token transfer
	if isNativeTokenTransfer(mcmsTx) {
		return createNativeTransferCall(mcmsTx), nil, "", nil
	}

	evmRegistry := ctx.GetEVMRegistry()
	if evmRegistry == nil {
		return nil, nil, "", errors.New("EVM registry is not available")
	}
	abi, abiStr, err := evmRegistry.GetABIByAddress(chainSelector, mcmsTx.To)
	if err != nil {
		return nil, nil, "", err
	}

	analyzeResult, err := decoder.Decode(mcmsTx.To, abi, mcmsTx.Data)
	if err != nil {
		return nil, nil, "", fmt.Errorf("error analyzing operation: %w", err)
	}

	return analyzeResult, abi, abiStr, nil
}

// isNativeTokenTransfer checks if a transaction is a native token transfer
func isNativeTokenTransfer(mcmsTx types.Transaction) bool {
	// Native transfers have empty data and non-zero value
	value := getTransactionValue(mcmsTx)
	return len(mcmsTx.Data) == 0 && value.Cmp(big.NewInt(0)) > 0
}

// getTransactionValue extracts the value from AdditionalFields
func getTransactionValue(mcmsTx types.Transaction) *big.Int {
	// Try to unmarshal as a number first (most common case)
	var additionalFields struct{ Value json.Number }
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFields); err == nil {
		value, ok := new(big.Int).SetString(string(additionalFields.Value), 10)
		if ok {
			return value
		}
	}

	// Fallback: try to unmarshal as a string
	var additionalFieldsStr struct{ Value string }
	if err := json.Unmarshal(mcmsTx.AdditionalFields, &additionalFieldsStr); err == nil {
		value, ok := new(big.Int).SetString(additionalFieldsStr.Value, 10)
		if ok {
			return value
		}
	}

	// If both fail, return 0
	return big.NewInt(0)
}

// createNativeTransferCall creates a DecodedCall for native token transfers
func createNativeTransferCall(mcmsTx types.Transaction) *DecodedCall {
	value := getTransactionValue(mcmsTx)

	// Convert wei to ETH using big.Rat for precise decimal representation
	eth := new(big.Rat).SetFrac(value, big.NewInt(1e18))

	return &DecodedCall{
		Address: mcmsTx.To,
		Method:  "native_transfer",
		Inputs: []NamedDescriptor{
			{
				Name:  "recipient",
				Value: AddressDescriptor{Value: mcmsTx.To},
			},
			{
				Name:  "amount_wei",
				Value: SimpleDescriptor{Value: value.String()},
			},
			{
				Name:  "amount_eth",
				Value: SimpleDescriptor{Value: eth.FloatString(18)},
			},
		},
		Outputs: []NamedDescriptor{},
	}
}
