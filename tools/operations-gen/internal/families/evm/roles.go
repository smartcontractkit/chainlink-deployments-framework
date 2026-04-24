package evm

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// resolveRoleField converts a role string from the config into a [32]byte value.
//
// Accepted formats:
//   - "DEFAULT_ADMIN_ROLE"        → all-zero bytes32 (OpenZeppelin convention)
//   - 64 hex chars (optional 0x)  → raw bytes32 value
//   - any other string            → keccak256(utf8 bytes), matching the Solidity
//     pattern `bytes32 constant SOME_ROLE = keccak256("SOME_ROLE")`
func resolveRoleField(s string) ([32]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return [32]byte{}, errors.New("empty role")
	}
	h := strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if len(h) == 64 && isAllHex64(h) {
		return parseRoleHex(s)
	}
	if s == "DEFAULT_ADMIN_ROLE" {
		return [32]byte{}, nil
	}
	outHash := crypto.Keccak256([]byte(s))
	var out [32]byte
	copy(out[:], outHash)

	return out, nil
}

// isAllHex64 returns true if h is exactly 64 valid hex characters.
func isAllHex64(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, r := range h {
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

// parseRoleHex decodes a 64-hex-character (optional 0x prefix) string into a [32]byte.
func parseRoleHex(s string) ([32]byte, error) {
	var zero [32]byte
	h := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X"))
	if len(h) != 64 {
		return zero, fmt.Errorf(
			"role hex must be 64 hex characters (32 bytes), got %d chars", len(h),
		)
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return zero, fmt.Errorf("invalid role hex: %w", err)
	}
	if len(b) != 32 {
		return zero, errors.New("invalid role length")
	}
	var out [32]byte
	copy(out[:], b)

	return out, nil
}

// formatRoleGoLiteral renders a [32]byte role value as a Go byte-array literal,
// e.g. [32]byte{0x12, 0x34, …}.
func formatRoleGoLiteral(r [32]byte) string {
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
