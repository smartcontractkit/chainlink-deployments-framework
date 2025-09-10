package onchain

import (
	"errors"
	"fmt"
)

// ErrMaxSelectorsReached is returned when the user requests more selectors than are available
// from the predefined test selectors.
var ErrMaxSelectorsReached = errors.New("max selectors reached")

// errMaxSelectors wraps ErrMaxSelectorsReached with a message indicating the maximum number of
// selectors that are available.
func errMaxSelectors(maxCount int) error {
	return fmt.Errorf("%w: a maximum of %d selectors are available", ErrMaxSelectorsReached, maxCount)
}
