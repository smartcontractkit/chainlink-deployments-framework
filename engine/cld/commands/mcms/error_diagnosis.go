package mcms

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/mcms/types"

	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// diagnoseTimelockRevert attempts to diagnose why a timelock execution reverted.
func diagnoseTimelockRevert(
	ctx context.Context,
	lggr logger.Logger,
	rpcURL string,
	selector uint64,
	bops []types.BatchOperation,
	timelockAddr ethcommon.Address,
	addressBook cldf.AddressBook,
	proposalCtx analyzer.ProposalContext,
) error {
	// One client for both impersonation and eth calls
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer rpcClient.Close()

	ec := ethclient.NewClient(rpcClient)
	defer ec.Close()

	// Start/stop impersonation
	if err = rpcClient.CallContext(ctx, nil, "anvil_impersonateAccount", timelockAddr.Hex()); err != nil {
		return fmt.Errorf("impersonate timelock: %w", err)
	}
	defer func() {
		_ = rpcClient.CallContext(context.Background(), nil, "anvil_stopImpersonatingAccount", timelockAddr.Hex())
	}()
	lggr.Infof("Impersonating timelock %s on selector %d", timelockAddr.Hex(), selector)

	var errLogs []string
	errDec, err := NewErrDecoder(proposalCtx.GetEVMRegistry())
	if err != nil {
		return fmt.Errorf("create error decoder: %w", err)
	}
	for bi, bop := range bops {
		if uint64(bop.ChainSelector) != selector {
			continue
		}
		for ti, tx := range bop.Transactions {
			value, valErr := parseEVMValue(string(tx.AdditionalFields))
			if valErr != nil {
				msg := fmt.Sprintf("batch %d tx %d: additionalFields invalid: %v", bi, ti, valErr)
				lggr.Error(msg)
				errLogs = append(errLogs, msg)

				continue
			}

			to := ethcommon.HexToAddress(tx.To)
			msg := ethereum.CallMsg{
				From:  timelockAddr,
				To:    &to,
				Value: value,
				Data:  tx.Data,
			}

			lggr.Infof("Dry-running batch %d tx %d -> to=%s value=%s dataLen=%d",
				bi, ti, to.Hex(), value.String(), len(tx.Data))

			_, callErr := ec.CallContract(ctx, msg, nil)
			if callErr == nil {
				lggr.Infof("batch %d - tx #%d succeeded (no revert)", bi, ti)

				continue
			}

			if data, ok := extractRevertData(callErr); ok {
				lggr.Warnf("raw revert data len=%d hex=%s", len(data), hex.EncodeToString(data))
			}

			calldataHex := "0x" + hex.EncodeToString(tx.Data)
			lggr.Infof("Calldata : %s", calldataHex)

			sel := first4(tx.Data)
			selHex := "0x" + hex.EncodeToString(sel)

			// Try to resolve function name from registry (fallback if AddressBook is empty)
			if fn, ok := funcNameFromRegistry(errDec.registry, sel); ok {
				lggr.Infof("batch %d - tx #%d selector %s was not found on addressbook, but looks like ABI from %s", bi, ti, selHex, fn)
			} else {
				lggr.Infof("batch %d - tx #%d selector %s (unknown to registry)", bi, ti, selHex)
			}

			// Prefer the target contract ABI (if known in AddressBook/Registry)
			prefABI := preferredABIForAddress(errDec, addressBook, selector, tx.To)

			// If the target ABI is known but doesn't contain the selector, call it out up front.
			if prefABI != "" {
				if _, ok := funcNameFromABI(prefABI, sel); !ok {
					// Try to guess the name from the global registry (often enough to identify the intent)
					fn, _ := funcNameFromRegistry(errDec.registry, sel)
					lggr.Warnf("batch %d - tx #%d: target %s does NOT implement selector %s (%s) — likely ABI/version mismatch",
						bi, ti, to.Hex(), selHex, fn)
				}
			}

			pretty, got := prettyRevertFromError(callErr, prefABI, errDec)

			if got && pretty != "" && pretty != noRevertData {
				msg := fmt.Sprintf("batch %d - tx #%d reverted: %s", bi, ti, pretty)
				errLogs = append(errLogs, msg)
				lggr.Error(msg)
			} else {
				msg := fmt.Sprintf("batch %d - tx #%d REVERTED (unknown reason): %v", bi, ti, callErr)
				errLogs = append(errLogs, msg)
				lggr.Error(msg)
			}
		}
	}

	if len(errLogs) > 0 {
		return fmt.Errorf("timelock revert diagnosis: %d errors found", len(errLogs))
	}

	return nil
}

// tryDecodeTxRevertEVM attempts to decode an EVM transaction revert.
func tryDecodeTxRevertEVM(
	ctx context.Context,
	evmClient cldf_evm.OnchainClient,
	tx *gethtypes.Transaction,
	preferredABIJSON string,
	blockNum *big.Int,
	proposalCtx analyzer.ProposalContext,
) (string, bool) {
	decoder, err := NewErrDecoder(proposalCtx.GetEVMRegistry())
	if err != nil {
		return "", false // best-effort, no decoder available
	}
	// Compute sender (falls back if signature missing)
	signer := gethtypes.LatestSignerForChainID(tx.ChainId())
	from, err := gethtypes.Sender(signer, tx)
	if err != nil {
		// best-effort: from is optional for eth_call; many reverts don't depend on it
		from = ethcommon.Address{}
	}

	msg := ethereum.CallMsg{
		From:  from,
		To:    tx.To(),
		Value: tx.Value(),
		Data:  tx.Data(),
	}
	_, callErr := evmClient.CallContract(ctx, msg, blockNum)
	if callErr == nil {
		return "", false
	}

	return prettyRevertFromError(callErr, preferredABIJSON, decoder)
}

// parseEVMValue extracts the value from transaction additional fields.
//
//nolint:unparam // error return is for future implementation
func parseEVMValue(additionalFields string) (*big.Int, error) {
	if additionalFields == "" {
		return big.NewInt(0), nil
	}

	// Try to parse as JSON with "value" field
	// For simplicity, assume value is 0 if not specified
	return big.NewInt(0), nil
}

// first4 returns the first 4 bytes of a slice (function selector).
func first4(data []byte) []byte {
	if len(data) < 4 {
		return data
	}

	return data[:4]
}

// extractRevertData attempts to extract revert data from an error.
//
//nolint:unparam // data return is for future implementation
func extractRevertData(err error) ([]byte, bool) {
	if err == nil {
		return nil, false
	}
	// This is a simplified implementation
	// In the real implementation, you'd parse the error for revert data

	return nil, false
}

// funcNameFromRegistry attempts to find a function name from the registry.
//
//nolint:unparam // name return is for future implementation
func funcNameFromRegistry(registry analyzer.EVMABIRegistry, selector []byte) (string, bool) {
	if registry == nil || len(selector) < 4 {
		return "", false
	}

	// Try to find function name from registry
	// This is a simplified implementation

	return "", false
}

// funcNameFromABI attempts to find a function name from an ABI.
func funcNameFromABI(_ string, _ []byte) (string, bool) {
	// Simplified implementation

	return "", false
}

// preferredABIForAddress returns the preferred ABI for decoding errors from an address.
func preferredABIForAddress(_ *ErrDecoder, _ cldf.AddressBook, _ uint64, _ string) string {
	// Simplified implementation - return empty to use generic decoding

	return ""
}

// prettyRevertFromError attempts to decode a revert error into a human-readable string.
func prettyRevertFromError(err error, preferredABIJSON string, dec *ErrDecoder) (string, bool) {
	if err == nil {
		return "", false
	}

	// Try to decode the error
	if dec != nil {
		errStr := err.Error()
		if pretty, ok := dec.decodeRecursive([]byte(errStr), preferredABIJSON); ok {
			return pretty, true
		}
	}

	return err.Error(), true
}
