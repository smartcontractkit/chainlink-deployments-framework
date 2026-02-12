package mcms

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/smartcontractkit/mcms/sdk/evm"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

const noRevertData = "(no revert data)"

type errorSelector [4]byte

var emptySelector = errorSelector{}

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

// matchErrorSelector tries to resolve a 4-byte selector to an error name.
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
		return pretty, ok
	}
	// 3) fallback on selector
	if len(data) >= 4 {
		return "custom error 0x" + hex.EncodeToString(data[:4]), true
	}

	return noRevertData, true
}

// DecodedExecutionError contains the decoded revert reasons from an ExecutionError.
type DecodedExecutionError struct {
	RevertReason            string
	RevertReasonDecoded     bool
	UnderlyingReason        string
	UnderlyingReasonDecoded bool
}

// tryDecodeExecutionError decodes an evm.ExecutionError into human-readable strings.
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

// ReadExecutionErrorFromFile reads and parses an execution error from a JSON file.
func ReadExecutionErrorFromFile(data []byte) (*evm.ExecutionError, error) {
	var jsonData map[string]any
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	execErrData, ok := jsonData["execution_error"]
	if !ok {
		return nil, errors.New("no execution error to decode, json file must contain an 'execution_error' key to get revert reasons decoded")
	}

	execErrBytes, err := json.Marshal(execErrData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling execution_error: %w", err)
	}

	var execErr evm.ExecutionError
	if err := json.Unmarshal(execErrBytes, &execErr); err != nil {
		return nil, fmt.Errorf("error unmarshaling execution_error: %w", err)
	}

	return &execErr, nil
}
