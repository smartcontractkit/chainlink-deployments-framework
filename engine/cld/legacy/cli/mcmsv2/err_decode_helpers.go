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

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/mcms/sdk/evm"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

const noRevertData = "(no revert data)"

type errorSelector [4]byte

var emptySelector = errorSelector{}

type traceConfig struct {
	DisableStorage bool `json:"disableStorage,omitempty"`
	DisableMemory  bool `json:"disableMemory,omitempty"`
	DisableStack   bool `json:"disableStack,omitempty"`
}

// toMap converts traceConfig to a map for RPC call.
func (c traceConfig) toMap() map[string]any {
	m := map[string]any{}
	if c.DisableStorage {
		m["disableStorage"] = true
	}
	if c.DisableMemory {
		m["disableMemory"] = true
	}
	if c.DisableStack {
		m["disableStack"] = true
	}

	return m
}

// ErrSig describes a resolved custom error.
type ErrSig struct {
	TypeVer string
	Name    string
	Inputs  abi.Arguments
	id      errorSelector
}

// ErrDecoder indexes custom-error selectors across many ABIs.
type ErrDecoder struct {
	bySelector map[errorSelector][]ErrSig
	registry   analyzer.EVMABIRegistry
}

// NewErrDecoder builds an index from EVM ABI registry.
func NewErrDecoder(registry analyzer.EVMABIRegistry) (*ErrDecoder, error) {
	idx := make(map[errorSelector][]ErrSig)
	for tv, jsonABI := range registry.GetAllABIs() {
		a, err := abi.JSON(strings.NewReader(jsonABI))
		if err != nil {
			return nil, fmt.Errorf("parse ABI for %s: %w", tv, err)
		}
		for name, e := range a.Errors {
			var key errorSelector
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

// funcNameFromABI returns "Name(type1,type2,...)" if selector exists in abiJSON.
func funcNameFromABI(abiJSON string, sel4 []byte) (string, bool) {
	if abiJSON == "" || len(sel4) < 4 {
		return "", false
	}
	a, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", false
	}
	for name, m := range a.Methods {
		if len(m.ID) >= 4 && bytes.Equal(m.ID[:4], sel4[:4]) {
			// build canonical signature "name(type1,type2,...)"
			argTypes := make([]string, len(m.Inputs))
			for i, in := range m.Inputs {
				argTypes[i] = in.Type.String()
			}

			return fmt.Sprintf("%s(%s)", name, strings.Join(argTypes, ",")), true
		}
	}

	return "", false
}

func first4(data []byte) []byte {
	if len(data) < 4 {
		return data
	}

	return data[:4]
}

// funcNameFromRegistry searches the whole registry and returns "Type@Version: name(sig)" if found.
func funcNameFromRegistry(reg analyzer.EVMABIRegistry, sel4 []byte) (string, bool) {
	if len(sel4) < 4 {
		return "", false
	}
	for tv, jsonABI := range reg.GetAllABIs() {
		if sig, ok := funcNameFromABI(jsonABI, sel4); ok {
			return fmt.Sprintf("%s: %s", tv, sig), true
		}
	}

	return "", false
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
	var key errorSelector
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

// prettyFromBytes tries Error(string), then custom errors (pref ABI -> registry)
func prettyFromBytes(data []byte, preferredABIJSON string, dec *ErrDecoder) (string, bool) {
	if len(data) == 0 {
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
	// 3) fallback on selector
	if len(data) >= 4 {
		return "custom error 0x" + hex.EncodeToString(data[:4]), true
	}

	return noRevertData, true
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

			// If the target ABI is known but doesn’t contain the selector, call it out up front.
			if prefABI != "" {
				if _, ok := funcNameFromABI(prefABI, sel); !ok {
					// Try to guess the name from the global registry (often enough to identify the intent)
					fn, _ := funcNameFromRegistry(errDec.registry, sel)
					lggr.Warnf("batch %d - tx #%d: target %s does NOT implement selector %s (%s) — likely ABI/version mismatch",
						bi, ti, to.Hex(), selHex, fn)
				}
			}

			pretty, got := prettyRevertFromError(callErr, prefABI, errDec)

			// (A) We decoded a *useful* reason → log it, no trace.
			if got && pretty != "" && pretty != noRevertData {
				m := fmt.Sprintf("batch %d - tx #%d reverted: %s", bi, ti, pretty)
				lggr.Warn(m)
				errLogs = append(errLogs, m)

				continue
			}

			// (B) We either got nothing or just "(no revert data)" → try trace now.
			if traceBytes, traceTxt, terr := debugTraceCall(ctx, rpcClient, msg); terr == nil {
				if len(traceBytes) > 0 {
					if p2, ok2 := prettyFromBytes(traceBytes, prefABI, errDec); ok2 && p2 != "" && p2 != noRevertData {
						m := fmt.Sprintf("batch %d - tx #%d reverted (trace): %s", bi, ti, p2)
						lggr.Error(m)
						errLogs = append(errLogs, m)

						continue
					}
					m := fmt.Sprintf("batch %d - tx #%d reverted (trace bytes, %d): 0x%s",
						bi, ti, len(traceBytes), hex.EncodeToString(traceBytes))
					lggr.Warn(m)
					errLogs = append(errLogs, m)

					continue
				}
				if traceTxt != "" {
					m := fmt.Sprintf("batch %d - tx #%d reverted (trace text): %s", bi, ti, traceTxt)
					lggr.Warn(m)
					errLogs = append(errLogs, m)

					continue
				}
			}

			// (C) Still nothing helpful → raw error.
			m := fmt.Sprintf("batch %d - tx #%d reverted (raw): %v", bi, ti, callErr)
			lggr.Error(m)
			errLogs = append(errLogs, m)
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

// maxScanDepth limits the recursive scanning depth for revert data to prevent infinite loops
// and stack overflows. The value 16 was chosen as a safe upper bound based on expected error
// nesting complexity in EVM transaction data. Adjust if deeper nesting is encountered in practice.
const maxScanDepth = 16

func scanRevertData(v interface{}) ([]byte, bool) {
	return scanRevertDataDepth(v, maxScanDepth)
}

func scanRevertDataDepth(v interface{}, depth int) ([]byte, bool) {
	if depth <= 0 || v == nil {
		return nil, false
	}

	decodeHex := func(s string) ([]byte, bool) {
		if strings.HasPrefix(s, "0x") {
			if b, e := hexutil.Decode(s); e == nil {
				return b, true
			}
		}

		return nil, false
	}

	switch t := v.(type) {
	case string:
		return decodeHex(t)
	case []byte:
		return t, true
	case hexutil.Bytes:
		return t, true
	case map[string]interface{}:
		for _, key := range []string{"data", "return", "returnValue"} {
			if s, ok := t[key].(string); ok {
				if b, ok := decodeHex(s); ok {
					return b, true
				}
			}
		}
		if orig, ok := t["originalError"].(map[string]interface{}); ok {
			if s, ok := orig["data"].(string); ok {
				if b, ok := decodeHex(s); ok {
					return b, true
				}
			}
			// continue into originalError
			if b, ok := scanRevertDataDepth(orig, depth-1); ok {
				return b, true
			}
		}
		for _, vv := range t {
			if b, ok := scanRevertDataDepth(vv, depth-1); ok {
				return b, true
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

// extractRevertData recursively unwraps the error chain to find revert data.
func extractRevertData(err error) ([]byte, bool) {
	if err == nil {
		return nil, false
	}
	type dataErr interface{ ErrorData() interface{} }

	for e := err; e != nil; e = errors.Unwrap(e) {
		var de dataErr
		if errors.As(e, &de) {
			if b, ok := scanRevertData(de.ErrorData()); ok {
				return b, true
			}
		}
	}

	return nil, false
}

// debugTraceCall recovers revert bytes or textual reason via debug_traceCall.
func debugTraceCall(ctx context.Context, rpcClient *rpc.Client, msg ethereum.CallMsg) (revertBytes []byte, reason string, _ error) {
	type callArg struct {
		From  string `json:"from,omitempty"`
		To    string `json:"to,omitempty"`
		Gas   string `json:"gas,omitempty"`
		Value string `json:"value,omitempty"`
		Data  string `json:"data,omitempty"`
	}
	arg := callArg{
		From: msg.From.Hex(),
		Data: "0x" + hex.EncodeToString(msg.Data),
	}
	if msg.To != nil {
		arg.To = msg.To.Hex()
	}
	if msg.Gas > 0 {
		arg.Gas = hexutil.EncodeUint64(msg.Gas)
	}
	if msg.Value != nil {
		arg.Value = hexutil.EncodeBig(msg.Value)
	}

	cfg := traceConfig{DisableStorage: true, DisableMemory: true, DisableStack: false}

	var res map[string]any
	if err := rpcClient.CallContext(ctx, &res, "debug_traceCall", arg, "latest", cfg.toMap()); err != nil {
		return nil, "", err
	}
	if s, ok := res["returnValue"].(string); ok && strings.HasPrefix(s, "0x") {
		if b, err := hexutil.Decode(s); err == nil {
			return b, "", nil
		}
	}
	if s, ok := res["return"].(string); ok && strings.HasPrefix(s, "0x") {
		if b, err := hexutil.Decode(s); err == nil {
			return b, "", nil
		}
	}
	if s, ok := res["error"].(string); ok && s != "" {
		return nil, s, nil
	}

	m, err := json.Marshal(res)
	if err != nil {
		return nil, "", err
	}
	if !bytes.Equal(m, []byte("{}")) {
		return nil, string(m), nil
	}

	return nil, "", nil
}

// prettyRevertFromError tries to extract and decode revert data from an error.
func prettyRevertFromError(err error, preferredABIJSON string, dec *ErrDecoder) (string, bool) {
	if err == nil {
		return "", false
	}

	if data, ok := extractRevertData(err); ok {
		// 1) standard Error(string)
		if reason, derr := abi.UnpackRevert(data); derr == nil {
			return reason, true
		}
		// 2) custom errors (preferred ABI -> registry, recursive unwrap)
		if pretty, ok := dec.decodeRecursive(data, preferredABIJSON); ok {
			return pretty, true
		}
		// 3) fallback on selector
		if len(data) >= 4 {
			return "custom error 0x" + hex.EncodeToString(data[:4]), true
		}
		// data present but <4 bytes
		return noRevertData, true
	}

	// 4) textual fallback in message
	if s := err.Error(); strings.Contains(s, "execution reverted:") {
		parts := strings.SplitN(s, "execution reverted:", 2)
		if len(parts) == 2 {
			txt := strings.TrimSpace(parts[1])
			if txt != "" {
				return txt, true
			}
		}
	}

	return "", false
}

// DecodedExecutionError contains the decoded revert reasons from an ExecutionError.
type DecodedExecutionError struct {
	RevertReason            string
	RevertReasonDecoded     bool
	UnderlyingReason        string
	UnderlyingReasonDecoded bool
}

// tryDecodeExecutionError decodes an evm.ExecutionError into human-readable strings.
// It first checks for RevertReasonDecoded and UnderlyingReasonDecoded fields.
// If those are not available, it extracts RevertReasonRaw and UnderlyingReasonRaw from the struct
// and decodes them using the provided ErrDecoder to match error selectors against the ABI registry.
func tryDecodeExecutionError(execError *evm.ExecutionError, dec *ErrDecoder) DecodedExecutionError {
	if execError == nil {
		return DecodedExecutionError{}
	}

	revertReason, revertDecoded := decodeRevertReasonWithStatus(execError, dec)
	underlyingReason, underlyingDecoded := decodeUnderlyingReasonWithStatus(execError, dec)

	return DecodedExecutionError{
		RevertReason:            revertReason,
		RevertReasonDecoded:     revertDecoded,
		UnderlyingReason:        underlyingReason,
		UnderlyingReasonDecoded: underlyingDecoded,
	}
}

// decodeRevertReasonWithStatus decodes the revert reason and returns both the reason and decoded status.
func decodeRevertReasonWithStatus(execError *evm.ExecutionError, dec *ErrDecoder) (string, bool) {
	if execError.RevertReasonDecoded != "" {
		return execError.RevertReasonDecoded, true
	}

	if execError.RevertReasonRaw == nil {
		return "", false
	}

	hasData := len(execError.RevertReasonRaw.Data) > 0
	hasSelector := execError.RevertReasonRaw.Selector != emptySelector

	if hasData {
		if reason, decoded := tryDecodeFromData(execError.RevertReasonRaw, dec); decoded {
			return reason, true
		}
	}

	if hasSelector && !hasData {
		reason := decodeSelectorOnly(execError.RevertReasonRaw.Selector, dec)
		return reason, reason != ""
	}

	return "", false
}

// tryDecodeFromData attempts to decode revert data from the CustomErrorData.
func tryDecodeFromData(raw *evm.CustomErrorData, dec *ErrDecoder) (string, bool) {
	if len(raw.Data) >= 4 {
		if reason, decoded := decodeRevertDataFromBytes(raw.Data, dec, ""); decoded {
			return reason, true
		}
	}

	if raw.Selector != emptySelector {
		if combined := raw.Combined(); len(combined) > 4 {
			return decodeRevertDataFromBytes(combined, dec, "")
		}
	}

	return "", false
}

// decodeSelectorOnly decodes an error when only the selector is available.
func decodeSelectorOnly(selector errorSelector, dec *ErrDecoder) string {
	if dec == nil {
		return formatSelectorHex(selector)
	}

	if matched, ok := dec.matchErrorSelector(selector); ok {
		return matched
	}

	return formatSelectorHex(selector)
}

// formatSelectorHex formats a selector as a hex string.
func formatSelectorHex(selector errorSelector) string {
	return "custom error 0x" + hex.EncodeToString(selector[:])
}

// decodeUnderlyingReasonWithStatus decodes the underlying reason and returns both the reason and decoded status.
func decodeUnderlyingReasonWithStatus(execError *evm.ExecutionError, dec *ErrDecoder) (string, bool) {
	if execError.UnderlyingReasonDecoded != "" {
		return execError.UnderlyingReasonDecoded, true
	}

	if execError.UnderlyingReasonRaw == "" {
		return "", false
	}

	reason, decoded := decodeRevertData(execError.UnderlyingReasonRaw, dec, "")

	return reason, decoded
}

// decodeRevertData decodes a hex string containing revert data into a human-readable error message.
func decodeRevertData(hexStr string, dec *ErrDecoder, preferredABIJSON string) (string, bool) {
	if hexStr == "" {
		return "", false
	}

	data, err := hexutil.Decode(hexStr)
	if err != nil || len(data) == 0 {
		return "", false
	}

	return decodeRevertDataFromBytes(data, dec, preferredABIJSON)
}

// matchErrorSelector tries to resolve a 4-byte selector to an error name.
// Returns "ErrorName(...) @Type@Version" if found in registry, or empty string if not found.
func (d *ErrDecoder) matchErrorSelector(sel4 errorSelector) (string, bool) {
	if d == nil || d.bySelector == nil {
		return "", false
	}

	cands, ok := d.bySelector[sel4]
	if !ok || len(cands) == 0 {
		return "", false
	}

	// If multiple ABIs define the same selector, pick the first.
	c := cands[0]

	return fmt.Sprintf("%s(...) @%s", c.Name, c.TypeVer), true
}

// decodeRevertDataFromBytes decodes revert data bytes into a human-readable error message.
func decodeRevertDataFromBytes(data []byte, dec *ErrDecoder, preferredABIJSON string) (string, bool) {
	if len(data) == 0 {
		return "", false
	}

	if dec == nil {
		if len(data) >= 4 {
			return "custom error 0x" + hex.EncodeToString(data[:4]), true
		}

		return "", false
	}

	return prettyFromBytes(data, preferredABIJSON, dec)
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
	}
	_, callErr := evmClient.CallContract(ctx, msg, blockNum)
	if callErr == nil {
		return "", false
	}

	return prettyRevertFromError(callErr, preferredABIJSON, decoder)
}
