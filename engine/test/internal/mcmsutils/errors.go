package mcmsutils

import (
	"errors"
	"fmt"
)

// errFamilyNotSupported is the error returned when a chain family is not implemented.
var ErrFamilyNotSupported = errors.New("chain family not supported")

// errFamilyNotSupported returns a wrapped error with additional information about the chain
// family that is not supported.
func errFamilyNotSupported(family string) error {
	return fmt.Errorf("%w: %s", ErrFamilyNotSupported, family)
}

func errEncoderNotFound(selector uint64) error {
	return fmt.Errorf("encoder not found for chain selector %d", selector)
}
