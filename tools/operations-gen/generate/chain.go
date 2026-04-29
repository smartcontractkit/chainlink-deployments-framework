package generate

import (
	"sort"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/core"
	"github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen/internal/families/evm"
)

// chainFamilies is the single registration point for all supported chain families.
var chainFamilies = map[string]core.ChainFamilyHandler{
	"evm": evm.Handler{},
}

// supportedFamilies returns a sorted, comma-separated list of supported chain families.
func supportedFamilies() string {
	families := make([]string, 0, len(chainFamilies))
	for k := range chainFamilies {
		families = append(families, k)
	}
	sort.Strings(families)

	return strings.Join(families, ", ")
}
