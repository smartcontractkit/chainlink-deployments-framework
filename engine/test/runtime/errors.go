package runtime

import "fmt"

func errProposalNotFound(id string) error {
	return fmt.Errorf("proposal not found: %s", id)
}
