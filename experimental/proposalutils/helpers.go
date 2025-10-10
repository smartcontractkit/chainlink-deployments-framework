package proposalutils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MatchesProposalPath is a simple filter for proposal JSON files in the expected dir.
func MatchesProposalPath(domain, environment, p string) bool {
	p = filepath.ToSlash(p)
	prefix := fmt.Sprintf("domains/%s/%s/proposals/", domain, environment)

	return strings.HasPrefix(p, prefix) && strings.HasSuffix(p, ".json")
}
