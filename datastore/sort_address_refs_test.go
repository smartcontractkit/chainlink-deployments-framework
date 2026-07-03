package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestSortAddressRefs_deterministic(t *testing.T) {
	t.Parallel()

	refs := []AddressRef{
		{Address: "0xB", ChainSelector: 2, Type: "Router", Version: semver.MustParse("1.2.0")},
		{Address: "0xA", ChainSelector: 1, Type: "Router", Version: semver.MustParse("1.2.0")},
	}

	SortAddressRefs(refs)

	if refs[0].ChainSelector != 1 || refs[1].ChainSelector != 2 {
		t.Fatalf("unexpected order: %+v", refs)
	}

	refs2 := []AddressRef{
		{Address: "0xB", ChainSelector: 2, Type: "Router", Version: semver.MustParse("1.2.0")},
		{Address: "0xA", ChainSelector: 1, Type: "Router", Version: semver.MustParse("1.2.0")},
	}
	SortAddressRefs(refs2)

	for i := range refs {
		if refs[i].ChainSelector != refs2[i].ChainSelector ||
			refs[i].Type != refs2[i].Type ||
			refs[i].Address != refs2[i].Address {
			t.Fatalf("sort not deterministic at %d", i)
		}
	}
}
