package datastore

import (
	"sort"

	"github.com/Masterminds/semver/v3"
)

// SortAddressRefs sorts refs in place for deterministic JSON serialization.
// Order: chainSelector, type, version, address, qualifier.
func SortAddressRefs(refs []AddressRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].ChainSelector != refs[j].ChainSelector {
			return refs[i].ChainSelector < refs[j].ChainSelector
		}
		if refs[i].Type != refs[j].Type {
			return refs[i].Type < refs[j].Type
		}
		if cmp := compareVersions(refs[i].Version, refs[j].Version); cmp != 0 {
			return cmp < 0
		}
		if refs[i].Address != refs[j].Address {
			return refs[i].Address < refs[j].Address
		}

		return refs[i].Qualifier < refs[j].Qualifier
	})
}

func compareVersions(a, b *semver.Version) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	default:
		return a.Compare(b)
	}
}
