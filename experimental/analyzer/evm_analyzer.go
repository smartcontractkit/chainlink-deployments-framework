package analyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
)

// EIP1967TargetContractStorageSlot is the storage slot for EIP-1967 proxy implementation address
// keccak256("eip1967.proxy.implementation") - 1
var EIP1967TargetContractStorageSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

func AnalyzeEVMTransactions(ctx context.Context, proposalCtx ProposalContext, env deployment.Environment, chainSelector uint64, txs []types.Transaction) ([]*DecodedCall, error) {
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
		decodedTxs[i], _, _, err = AnalyzeEVMTransaction(ctx, proposalCtx, env, decoder, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze transaction %d: %w", i, err)
		}
	}

	return decodedTxs, nil
}

func AnalyzeEVMTransaction(
	ctx context.Context, proposalCtx ProposalContext, env deployment.Environment, decoder *EVMTxCallDecoder, chainSelector uint64, mcmsTx types.Transaction,
) (*DecodedCall, *abi.ABI, string, error) {
	// Check if this is a native token transfer
	if isNativeTokenTransfer(mcmsTx) {
		return createNativeTransferCall(mcmsTx), nil, "", nil
	}

	evmRegistry := proposalCtx.GetEVMRegistry()
	if evmRegistry == nil {
		return nil, nil, "", errors.New("EVM registry is not available. Ensure you have provided one in the ProposalContextProvider via the WithEVMABIMappings() option")
	}
	abi, abiStr, err := evmRegistry.GetABIByAddress(chainSelector, mcmsTx.To)
	if err != nil {
		return nil, nil, "", err
	}

	analyzeResult, err := decoder.Decode(mcmsTx.To, abi, mcmsTx.Data)
	if err != nil {
		// Check if this is a "method not found" error - could be a proxy with wrong ABI
		if isMethodNotFoundError(err) {
			// Try EIP-1967 proxy fallback: query implementation slot and retry with implementation ABI
			fallbackResult, fallbackABI, fallbackABIStr, fallbackErr := tryEIP1967ProxyFallback(
				ctx, proposalCtx, env, chainSelector, mcmsTx.To, mcmsTx.Data, decoder,
			)
			if fallbackErr == nil {
				// Successfully decoded with implementation ABI
				return fallbackResult, fallbackABI, fallbackABIStr, nil
			}
			// Fallback failed, return original error
		}

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
		Inputs: []NamedField{
			{
				Name:  "recipient",
				Value: AddressField{Value: mcmsTx.To},
			},
			{
				Name:  "amount_wei",
				Value: SimpleField{Value: value.String()},
			},
			{
				Name:  "amount_eth",
				Value: SimpleField{Value: eth.FloatString(18)},
			},
		},
		Outputs: []NamedField{},
	}
}

// isMethodNotFoundError checks if an error indicates a method not found in ABI.
// This typically happens when trying to decode a transaction with the wrong ABI,
// such as using a proxy ABI when the transaction is actually calling the implementation.
func isMethodNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	return strings.Contains(errStr, "no method with id") ||
		strings.Contains(errStr, "method not found") ||
		strings.Contains(errStr, "invalid method id")
}

// queryEIP1967ImplementationSlot queries the EIP-1967 implementation storage slot
// and returns the implementation address if found.
func queryEIP1967ImplementationSlot(ctx context.Context, evmChain evm.Chain, proxyAddress string) (common.Address, error) {
	storageValue, err := evmChain.Client.StorageAt(ctx, common.HexToAddress(proxyAddress), EIP1967TargetContractStorageSlot, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read EIP-1967 storage slot: %w", err)
	}

	// Extract address from storage (last 20 bytes, right-padded)
	implAddress := common.BytesToAddress(storageValue)

	return implAddress, nil
}

// tryEIP1967ProxyFallback attempts to decode using implementation ABI if address is EIP-1967 proxy.
// This function orchestrates the full fallback flow:
// 1. Gets EVM chain from environment
// 2. Queries EIP-1967 implementation slot
// 3. Looks up implementation TypeAndVersion from address book
// 4. Gets implementation ABI
// 5. Retries decode with implementation ABI
func tryEIP1967ProxyFallback(
	ctx context.Context,
	proposalCtx ProposalContext,
	env deployment.Environment,
	chainSelector uint64,
	proxyAddress string,
	txData []byte,
	decoder *EVMTxCallDecoder,
) (*DecodedCall, *abi.ABI, string, error) {
	// Lazily get EVM chain from environment (only when fallback is needed)
	evmChains := env.BlockChains.EVMChains()
	evmChain, exists := evmChains[chainSelector]
	if !exists {
		return nil, nil, "", fmt.Errorf("EVM chain not available for selector %d", chainSelector)
	}

	// Query EIP-1967 implementation slot
	implAddress, err := queryEIP1967ImplementationSlot(ctx, evmChain, proxyAddress)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to query EIP-1967 implementation: %w", err)
	}

	// Check if implementation address is zero (not an EIP-1967 proxy)
	if implAddress == (common.Address{}) {
		return nil, nil, "", errors.New("EIP-1967 slot contains zero address (not a proxy)")
	}

	// Look up implementation TypeAndVersion from address book (checking both ExistingAddresses and DataStore)
	implAddressStr := implAddress.Hex()
	addressesByChain, err := getAllAddressesByChain(env)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get addresses: %w", err)
	}

	addressesForChain, ok := addressesByChain[chainSelector]
	if !ok {
		return nil, nil, "", fmt.Errorf("no addresses found for chain selector %d", chainSelector)
	}

	implTypeAndVersion, ok := addressesForChain[implAddressStr]
	if !ok {
		return nil, nil, "", fmt.Errorf("implementation address %s not found in address book or datastore for chain selector %d", implAddressStr, chainSelector)
	}

	// Get implementation ABI using existing registry method
	evmRegistry := proposalCtx.GetEVMRegistry()
	if evmRegistry == nil {
		return nil, nil, "", errors.New("EVM registry is not available")
	}

	implABI, implABIStr, err := evmRegistry.GetABIByType(implTypeAndVersion)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get ABI for implementation %v: %w", implTypeAndVersion, err)
	}

	// Retry decode with implementation ABI
	decodedResult, err := decoder.Decode(proxyAddress, implABI, txData)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to decode with implementation ABI: %w", err)
	}

	return decodedResult, implABI, implABIStr, nil
}

// getAllAddressesByChain retrieves addresses from both ExistingAddresses and DataStore,
// merging them into a single map.
func getAllAddressesByChain(env deployment.Environment) (deployment.AddressesByChain, error) {
	// Start with addresses from ExistingAddresses
	addressesByChain, err := env.ExistingAddresses.Addresses() //nolint:staticcheck
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses from ExistingAddresses: %w", err)
	}

	// Fetch addresses from DataStore
	dataStoreAddresses, err := env.DataStore.Addresses().Fetch()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch addresses from DataStore: %w", err)
	}

	// Merge DataStore addresses into the map
	for _, address := range dataStoreAddresses {
		chainAddresses, exists := addressesByChain[address.ChainSelector]
		if !exists {
			chainAddresses = map[string]deployment.TypeAndVersion{}
		}
		chainAddresses[address.Address] = deployment.TypeAndVersion{
			Type:    deployment.ContractType(address.Type),
			Version: pointer.DerefOrEmpty(address.Version),
			Labels:  deployment.NewLabelSet(address.Labels.List()...),
		}
		addressesByChain[address.ChainSelector] = chainAddresses
	}

	return addressesByChain, nil
}
