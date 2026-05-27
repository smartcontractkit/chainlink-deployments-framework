package evm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// ResolveRoleField converts a role string from the config into a [32]byte value.
//
// Accepted formats:
//   - "DEFAULT_ADMIN_ROLE" → all-zero bytes32 (OpenZeppelin convention)
//   - any other role name  → keccak256(utf8 bytes), matching the Solidity
//     pattern `bytes32 constant SOME_ROLE = keccak256("SOME_ROLE")`
func ResolveRoleField(s string) ([32]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return [32]byte{}, errors.New("empty role")
	}
	if isRawRoleHash(s) {
		return [32]byte{}, errors.New("role must be a human-readable role name, not a raw bytes32 hash")
	}
	if s == "DEFAULT_ADMIN_ROLE" {
		return [32]byte{}, nil
	}
	outHash := crypto.Keccak256([]byte(s))
	var out [32]byte
	copy(out[:], outHash)

	return out, nil
}

func isRawRoleHash(s string) bool {
	h := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	return strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") || isHexString(h, 64)
}

func isHexString(s string, length int) bool {
	if len(s) != length {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}

	return true
}

// FormatRoleGoLiteral renders a [32]byte role value as a Go byte-array literal,
// e.g. [32]byte{0x12, 0x34, …}.
func FormatRoleGoLiteral(r [32]byte) string {
	if r == [32]byte{} {
		return "[32]byte{}"
	}
	var b strings.Builder
	b.WriteString("[32]byte{")
	for i, v := range r {
		if i > 0 {
			b.WriteString(", ")
		}
		_, _ = fmt.Fprintf(&b, "0x%02x", v)
	}
	b.WriteString("}")

	return b.String()
}
