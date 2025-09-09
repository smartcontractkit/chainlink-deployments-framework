package mcmsv2

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// ErrSig describes a resolved custom error.
type ErrSig struct {
	TypeVer string
	Name    string
	Inputs  abi.Arguments
	id      [4]byte
}

// ErrDecoder indexes custom-error selectors across many ABIs.
type ErrDecoder struct {
	bySelector map[[4]byte][]ErrSig
	registry   analyzer.EVMABIRegistry
}

// NewErrDecoder builds an index from EVM ABI registry.
func NewErrDecoder(registry analyzer.EVMABIRegistry) (*ErrDecoder, error) {
	idx := make(map[[4]byte][]ErrSig)
	for tv, jsonABI := range registry.GetAllABIs() {
		a, err := abi.JSON(strings.NewReader(jsonABI))
		if err != nil {
			return nil, fmt.Errorf("parse ABI for %s: %w", tv, err)
		}
		for name, e := range a.Errors {
			var key [4]byte
			copy(key[:], e.ID[:4]) // selector is first 4 bytes of the keccak(sig)
			idx[key] = append(idx[key], ErrSig{
				TypeVer: tv,
				Name:    name,
				Inputs:  e.Inputs,
				id:      key,
			})
		}
	}

	return &ErrDecoder{bySelector: idx, registry: registry}, nil
}

// decodeRecursive tries preferred ABI first (with recursive unwrap),
// then the global ABI registry (also with recursive unwrap).
func (d *ErrDecoder) decodeRecursive(revertData []byte, preferredABIJSON string) (string, bool) {
	if len(revertData) < 4 {
		return "", false
	}

	sel := revertData[:4]
	payload := revertData[4:]

	// --- A) Preferred ABI recursive-aware ---
	if preferredABIJSON != "" {
		if a, err := abi.JSON(strings.NewReader(preferredABIJSON)); err == nil {
			// Find error by selector
			for name, e := range a.Errors {
				if bytes.Equal(e.ID[:4], sel) {
					vs, err := e.Inputs.Unpack(payload)
					if err != nil {
						break // malformed, fall through to registry
					}
					// Unwrap if single bytes arg
					if len(vs) == 1 {
						if inner, ok := vs[0].([]byte); ok && len(inner) >= 4 {
							// 1) Standard Error(string)?
							if reason, derr := abi.UnpackRevert(inner); derr == nil {
								return fmt.Sprintf("%s(...) -> Error(%s)", name, reason), true
							}
							// 2) Another custom error? Recurse with no preferred ABI.
							if pretty, ok := d.decodeRecursive(inner, ""); ok {
								return fmt.Sprintf("%s(...) -> %s", name, pretty), true
							}
						}
					}
					// Not a single-bytes wrapper: print args as-is
					args := make([]string, len(vs))
					for i, v := range vs {
						args[i] = fmt.Sprintf("%v", v)
					}

					return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", ")), true
				}
			}
		}
	}

	// --- B) Registry lookup
	var key [4]byte
	copy(key[:], sel)
	cands, ok := d.bySelector[key]
	if !ok {
		return "", false
	}

	for _, c := range cands {
		vs, err := c.Inputs.Unpack(payload)
		if err != nil {
			continue
		}
		if len(vs) == 1 {
			if inner, ok := vs[0].([]byte); ok && len(inner) >= 4 {
				if reason, derr := abi.UnpackRevert(inner); derr == nil {
					return fmt.Sprintf("%s(...) -> Error(%s)", c.Name, reason), true
				}
				if pretty, ok := d.decodeRecursive(inner, ""); ok {
					return fmt.Sprintf("%s(...) -> %s", c.Name, pretty), true
				}
			}
		}
		args := make([]string, len(vs))
		for i, v := range vs {
			args[i] = fmt.Sprintf("%v", v)
		}

		return fmt.Sprintf("%s(%s) @%s", c.Name, strings.Join(args, ", "), c.TypeVer), true
	}

	return "", false
}

// decodeWithABI decodes the revert data using a specific ABI JSON.
func decodeWithABI(abiJSON string, revertData []byte) (string, bool) {
	if len(revertData) < 4 {
		return "", false
	}
	a, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", false
	}
	sel := revertData[:4]
	for name, e := range a.Errors {
		if bytes.Equal(e.ID[:4], sel) {
			vs, err := e.Inputs.Unpack(revertData[4:])
			if err != nil {
				return "", false
			}
			args := make([]string, len(vs))
			for i, v := range vs {
				args[i] = fmt.Sprintf("%v", v)
			}

			return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", ")), true
		}
	}

	return "", false
}

// diagnoseTimelockRevert impersonates the Timelock in Anvil and dry-runs each
// BatchOperation tx directly on its target. It aggregates all revert reasons
// and returns them in a single error (while also logging them).
func diagnoseTimelockRevert(
	ctx context.Context,
	lggr logger.Logger,
	rpcURL string,
	selector uint64,
	bops []types.BatchOperation,
	timelockAddr ethcommon.Address,
	addressBook cldf.AddressBook, // resolve type/version of target for ABI
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
	var errDec *ErrDecoder
	errDec, err = NewErrDecoder(proposalCtx.GetEVMRegistry())
	if err != nil {
		return fmt.Errorf("create error decoder: %w", err)
	}
	for bi, bop := range bops {
		if uint64(bop.ChainSelector) != selector {
			continue
		}
		for ti, tx := range bop.Transactions {
			value, valErr := parseEVMValue(tx.AdditionalFields)
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

			// Prefer the target contract ABI (if known in AddressBook/Registry)
			prefABI := preferredABIForAddress(errDec, addressBook, selector, tx.To)

			if pretty, ok := prettyRevertFromError(callErr, prefABI, errDec); ok {
				msg := fmt.Sprintf("batch %d - tx #%d reverted: %s", bi, ti, pretty)
				lggr.Error(msg)
				errLogs = append(errLogs, msg)
			} else {
				// Could not extract EVM-style revert data, print raw error
				msg := fmt.Sprintf("batch %d - tx #%d reverted (raw): %v", bi, ti, callErr)
				lggr.Error(msg)
				errLogs = append(errLogs, msg)
			}
		}
	}

	lggr.Info("Diagnosis finished.")
	if len(errLogs) > 0 {
		return fmt.Errorf("timelock diagnosis found issues:\n%s", strings.Join(errLogs, "\n"))
	}

	return nil
}

// parseEVMValue parses the additional fields of an EVM transaction to extract the value.
func parseEVMValue(additional json.RawMessage) (*big.Int, error) {
	fields := evm.AdditionalFields{Value: big.NewInt(0)}
	if len(additional) != 0 {
		if err := json.Unmarshal(additional, &fields); err != nil {
			return nil, fmt.Errorf("unmarshal additionalFields: %w", err)
		}
	}
	if err := fields.Validate(); err != nil {
		return nil, err
	}
	if fields.Value == nil {
		return big.NewInt(0), nil
	}

	return fields.Value, nil
}

// extractRevertData attempts to extract revert data from an error.
func extractRevertData(err error) ([]byte, bool) {
	if err == nil {
		return nil, false
	}

	// go-ethereum exposes error data via an ErrorData() method on some errors (e.g. rpc errors)
	type dataErr interface{ ErrorData() interface{} }

	for e := err; e != nil; e = errors.Unwrap(e) {
		var de dataErr
		if errors.As(e, &de) {
			switch v := de.ErrorData().(type) {
			case string:
				if strings.HasPrefix(v, "0x") {
					bytes, err := hexutil.Decode(v)
					if err == nil {
						return bytes, true
					}
				}
			case []byte:
				return v, true
			case hexutil.Bytes:
				return v, true
			case map[string]interface{}:
				if data, ok := v["data"].(string); ok {
					if strings.HasPrefix(data, "0x") {
						bytes, err := hexutil.Decode(data)
						if err == nil {
							return bytes, true
						}
					}
				}
			}
		}
	}

	return nil, false
}

// preferredABIForAddress looks up the target address in the address book and returns its ABI JSON if known.
// TODO: figure out how to support this when AddressBook gets deprecated using Catalogue
func preferredABIForAddress(errDec *ErrDecoder, ab cldf.AddressBook, selector uint64, contractAddress string) string {
	if ab == nil {
		return ""
	}
	contracts, _ := ab.AddressesForChain(selector)
	if c, ok := contracts[strings.ToLower(contractAddress)]; ok {
		tv := cldf.NewTypeAndVersion(c.Type, c.Version)
		if _, abiJSON, err := errDec.registry.GetABIByType(tv); err == nil {
			return abiJSON
		}
	}

	return ""
}

// prettyRevertFromError extracts revert data from an error and pretty-prints it.
// It tries Error(string), then custom errors (with recursion), else selector fallback.
func prettyRevertFromError(err error, preferredABIJSON string, dec *ErrDecoder) (string, bool) {
	data, ok := extractRevertData(err)
	if !ok {
		return "", false
	}

	// 1) standard Error(string)
	if reason, derr := abi.UnpackRevert(data); derr == nil {
		return reason, true
	}

	// 2) custom errors (preferred ABI -> registry, recursive unwrap)
	if pretty, ok := dec.decodeRecursive(data, preferredABIJSON); ok {
		return pretty, true
	}

	// 3) fallback
	if len(data) >= 4 {
		return "custom error 0x" + hex.EncodeToString(data[:4]), true
	}

	return "(no revert data)", true
}

type callContractClient interface {
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

// tryDecodeTxRevertEVM simulates the given tx and decodes any revert payload.
// If it finds EVM revert data, returns a pretty message and true.
func tryDecodeTxRevertEVM(
	ctx context.Context,
	evmClient callContractClient,
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
		// Gas/GasPrice not needed for reverting calls; omit to avoid "intrinsic gas too low"
	}
	_, callErr := evmClient.CallContract(ctx, msg, blockNum)
	if callErr == nil {
		return "", false
	}

	return prettyRevertFromError(callErr, preferredABIJSON, decoder)
}
