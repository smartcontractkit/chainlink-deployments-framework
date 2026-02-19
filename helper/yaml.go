package helper

import (
	"errors"
	"math/big"
	"regexp"
	"strconv"
)

type KeyMatchFunc func(key string) bool

// CoerceBigIntStrings walks a yaml.Unmarshal result (maps/slices/scalars) and
// converts *string* values that look like integers AND overflow int64 into *big.Int.
// It mutates map[string]any and []any in-place.
func CoerceBigIntStrings(v any) any {
	return coerceBigIntStrings(v, "", nil)
}

// CoerceBigIntStringsForKeys is the safer variant: only converts numeric strings
// under specific leaf keys (e.g. "maxfeejuelspermsg").
func CoerceBigIntStringsForKeys(v any, matchFunc KeyMatchFunc) any {
	return coerceBigIntStrings(v, "", matchFunc)
}

func coerceBigIntStrings(v any, currentKey string, matchFunc KeyMatchFunc) any {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			key := currentKey + "." + k
			x[k] = coerceBigIntStrings(vv, key, matchFunc)
		}

		return x

	case []map[string]any:
		for i := range x {
			// elements in a list don't have their own key, so we use the parent key with a ".[]" suffix
			// to check if we should coerce big ints in this list.
			key := currentKey + ".[]"
			x[i] = coerceBigIntStrings(x[i], key, matchFunc).(map[string]any)
		}

		return x

	case []any:
		for i := range x {
			// elements in a list don't have their own key, so we use the parent key with a ".[]" suffix
			// to check if we should coerce big ints in this list.
			key := currentKey + ".[]"
			x[i] = coerceBigIntStrings(x[i], key, matchFunc)
		}

		return x

	case string:
		if matchFunc != nil {
			if !matchFunc(currentKey) {
				return x
			}
		}
		if bi, ok := stringToBigIntIfOverflowInt64(x); ok {
			return bi // IMPORTANT: *big.Int (pointer), not big.Int (value)
		}
		return x

	default:
		return v
	}
}

func stringToBigIntIfOverflowInt64(s string) (*big.Int, bool) {
	// If it fits int64, keep it as a string (likely user meant a string, or YAML had quotes).
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return nil, false
	} else {
		var ne *strconv.NumError
		if errors.As(err, &ne) && ne.Err != strconv.ErrRange {
			// not a range overflow; should be safe to treat as string
			return nil, false
		}
	}

	// If it fits uint64, keep it as a string (likely user meant a string, or YAML had quotes).
	if _, err := strconv.ParseUint(s, 10, 64); err == nil {
		return nil, false
	} else {
		var ne *strconv.NumError
		if errors.As(err, &ne) && ne.Err != strconv.ErrRange {
			// not a range overflow; should be safe to treat as string
			return nil, false
		}
	}

	z, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, false
	}
	return z, true
}

// FIXUP: we coerce big int strings to avoid yaml parsing issues where large numbers
// get parsed as string instead of big number compatible type like json.Number or *big.Int.
func DefaultMatchKeysToFix(key string) bool {
	patterns := []string{
		// Matching examples:
		//	.[].0006_deploy_evm_chain_contracts.payload.contractparamsperchain.7759470850252068959.feequoterparams.maxfeejuelspermsg
		`^.*\.contractparamsperchain\.\d+\.feequoterparams\.maxfeejuelspermsg$`,
		// Matching examples:
		// 	.[].0009_deploy_ton_ccip_contracts.payload.chains.13879075125137744094.maxfeejuelspermsg
		`^.*\.chains\.\d+\.maxfeejuelspermsg$`,
		// Matching examples:
		// 	.[].0016_configure_lanes.payload.lanes.[].chaina.tokenprices.EQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAd99
		// 	.[].0016_configure_lanes.payload.lanes.[].chainb.tokenprices.0xb1D4538B4571d411F07960EF2838Ce337FE1E80E
		`^.*\.lanes\.\[\]\.(?:chaina|chainb)\.tokenprices\.[a-zA-Z0-9]+$`,
		`^.*\.lanes\.\[\]\.(?:chaina|chainb)\.feequoterdestchainconfig\.gasmultiplierweipereth$`,
	}

	matched := false
	for _, p := range patterns {
		var err error
		matched, err = regexp.MatchString(p, key)
		if err != nil {
			break
		}
		if matched {
			break
		}
	}

	return matched
}
