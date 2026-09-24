package statediff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

// DecodeJSON decodes exactly one JSON value without rounding numbers through
// float64. Callers storing arbitrary state values must retain json.Number all
// the way through artifact loading, merging and rendering.
func DecodeJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return fmt.Errorf("trailing JSON: %w", err)
		}

		return errors.New("unexpected trailing JSON value")
	}

	return nil
}

// EqualJSONValues compares decoded JSON values, treating equivalent numeric
// spellings (1, 1.0, 1e0) as equal without converting them to floating point.
func EqualJSONValues(a, b any) bool {
	var ok bool
	if a, ok = jsonShape(a); !ok {
		return false
	}
	if b, ok = jsonShape(b); !ok {
		return false
	}
	if an, ok := a.(json.Number); ok {
		bn, bothNumbers := b.(json.Number)
		return bothNumbers && numberKey(an) == numberKey(bn)
	}
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for key, value := range av {
			other, present := bv[key]
			if !present || !EqualJSONValues(value, other) {
				return false
			}
		}

		return true
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i, value := range av {
			if !EqualJSONValues(value, bv[i]) {
				return false
			}
		}

		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

// jsonShape also accepts typed artifact structs and values constructed in
// memory, while keeping already-decoded documents allocation-free here.
func jsonShape(value any) (any, bool) {
	switch value.(type) {
	case nil, string, bool, json.Number:
		return value, true
	case map[string]any, []any:
		if reflect.ValueOf(value).IsNil() {
			return nil, true
		}

		return value, true
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, false
		}
		var decoded any
		if err := DecodeJSON(raw, &decoded); err != nil {
			return nil, false
		}

		return decoded, true
	}
}

// numberKey normalizes a valid JSON number to coefficient and decimal exponent.
// The exponent stays symbolic so even 1e1000000000 requires bounded memory.
func numberKey(n json.Number) string {
	coefficient, exponent := numberParts(n)
	return coefficient + "e" + exponent.String()
}

func numberParts(n json.Number) (string, *big.Int) {
	s := string(n)
	exponent := new(big.Int)
	if pos := strings.IndexAny(s, "eE"); pos >= 0 {
		exponent.SetString(s[pos+1:], 10)
		s = s[:pos]
	}
	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	if point := strings.IndexByte(s, '.'); point >= 0 {
		exponent.Sub(exponent, big.NewInt(int64(len(s)-point-1)))
		s = s[:point] + s[point+1:]
	}
	s = strings.TrimLeft(s, "0")
	if s == "" {
		return "0", new(big.Int)
	}
	trimmed := strings.TrimRight(s, "0")
	exponent.Add(exponent, big.NewInt(int64(len(s)-len(trimmed))))
	if negative {
		trimmed = "-" + trimmed
	}

	return trimmed, exponent
}

// selectorNumber accepts only an exact uint64 identity. Decimal and
// exponent spellings are allowed when they denote an integer in that range.
func selectorNumber(n json.Number) (string, bool) {
	coefficient, exponent := numberParts(n)
	if strings.HasPrefix(coefficient, "-") || !exponent.IsInt64() {
		return "", false
	}
	scale := exponent.Int64()
	if scale < 0 || scale > 19 || int64(len(coefficient))+scale > 20 {
		return "", false
	}
	identity := coefficient + strings.Repeat("0", int(scale))
	value, err := strconv.ParseUint(identity, 10, 64)

	return strconv.FormatUint(value, 10), err == nil
}

// normalizeAddress preserves case-sensitive non-hex identities (e.g. Solana).
func normalizeAddress(address string) string {
	address = strings.TrimSpace(address)
	if strings.HasPrefix(address, "0x") || strings.HasPrefix(address, "0X") {
		return strings.ToLower(address)
	}

	return address
}
