package mcmsutils

import (
	"errors"
)

// ErrFamilyNotSupported is the error returned when a chain family is not implemented.
var ErrFamilyNotSupported = errors.New("chain family not supported")
