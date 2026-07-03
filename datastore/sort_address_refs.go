package datastore

import "sort"

// SortAddressRefs sorts refs in place for stable JSON serialization.
// Order: chainSelector, type, version, address, qualifier.
func SortAddressRefs(refs []AddressRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].ChainSelector != refs[j].ChainSelector {
			return refs[i].ChainSelector < refs[j].ChainSelector
		}
		if refs[i].Type != refs[j].Type {
			return refs[i].Type < refs[j].Type
		}
		vi, vj := "", ""
		if refs[i].Version != nil {
			vi = refs[i].Version.String()
		}
		if refs[j].Version != nil {
			vj = refs[j].Version.String()
		}
		if vi != vj {
			return vi < vj
		}
		if refs[i].Address != refs[j].Address {
			return refs[i].Address < refs[j].Address
		}

		return refs[i].Qualifier < refs[j].Qualifier
	})
}
