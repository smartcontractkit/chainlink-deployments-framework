package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestSortAddressRefs_order(t *testing.T) {
	t.Parallel()

	v10 := semver.MustParse("10.0.0")
	v2 := semver.MustParse("2.0.0")
	v1 := semver.MustParse("1.0.0")

	refs := []AddressRef{
		{Address: "0x2", ChainSelector: 1, Type: "Router", Version: v10, Qualifier: "b"},
		{Address: "0x1", ChainSelector: 1, Type: "Router", Version: v2, Qualifier: "a"},
		{Address: "0x3", ChainSelector: 1, Type: "OnRamp", Version: v1},
		{Address: "0x4", ChainSelector: 2, Type: "Router", Version: v1},
		{Address: "0x5", ChainSelector: 1, Type: "Router", Version: v2, Qualifier: "z"},
		{Address: "0x6", ChainSelector: 1, Type: "Router", Version: nil, Qualifier: "nil"},
	}

	SortAddressRefs(refs)

	expected := []AddressRef{
		{Address: "0x3", ChainSelector: 1, Type: "OnRamp", Version: v1},
		{Address: "0x6", ChainSelector: 1, Type: "Router", Version: nil, Qualifier: "nil"},
		{Address: "0x1", ChainSelector: 1, Type: "Router", Version: v2, Qualifier: "a"},
		{Address: "0x5", ChainSelector: 1, Type: "Router", Version: v2, Qualifier: "z"},
		{Address: "0x2", ChainSelector: 1, Type: "Router", Version: v10, Qualifier: "b"},
		{Address: "0x4", ChainSelector: 2, Type: "Router", Version: v1},
	}

	require.Equal(t, expected, refs)
}

func TestSortAddressRefs_semverNotLexicographic(t *testing.T) {
	t.Parallel()

	v10 := semver.MustParse("10.0.0")
	v2 := semver.MustParse("2.0.0")

	refs := []AddressRef{
		{Address: "0x1", ChainSelector: 1, Type: "Router", Version: v10},
		{Address: "0x2", ChainSelector: 1, Type: "Router", Version: v2},
	}

	SortAddressRefs(refs)

	require.Equal(t, v2.String(), refs[0].Version.String())
	require.Equal(t, v10.String(), refs[1].Version.String())
}

func TestSortAddressRefs_stableForEqualKeys(t *testing.T) {
	t.Parallel()

	v1 := semver.MustParse("1.0.0")
	first := AddressRef{
		Address:       "0xA",
		ChainSelector: 1,
		Type:          "Router",
		Version:       v1,
		Qualifier:     "q",
		Labels:        NewLabelSet("first"),
	}
	second := AddressRef{
		Address:       "0xA",
		ChainSelector: 1,
		Type:          "Router",
		Version:       v1,
		Qualifier:     "q",
		Labels:        NewLabelSet("second"),
	}

	refs := []AddressRef{first, second}
	SortAddressRefs(refs)

	require.Equal(t, "first", refs[0].Labels.List()[0])
	require.Equal(t, "second", refs[1].Labels.List()[0])
}

func TestSortAddressRefs_deterministic(t *testing.T) {
	t.Parallel()

	refs := []AddressRef{
		{Address: "0xB", ChainSelector: 2, Type: "Router", Version: semver.MustParse("1.2.0")},
		{Address: "0xA", ChainSelector: 1, Type: "Router", Version: semver.MustParse("1.2.0")},
	}

	SortAddressRefs(refs)

	refs2 := []AddressRef{
		{Address: "0xB", ChainSelector: 2, Type: "Router", Version: semver.MustParse("1.2.0")},
		{Address: "0xA", ChainSelector: 1, Type: "Router", Version: semver.MustParse("1.2.0")},
	}
	SortAddressRefs(refs2)

	require.Equal(t, refs, refs2)
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	v1 := semver.MustParse("1.0.0")
	v2 := semver.MustParse("2.0.0")

	require.Equal(t, 0, compareVersions(nil, nil))
	require.Equal(t, -1, compareVersions(nil, v1))
	require.Equal(t, 1, compareVersions(v1, nil))
	require.Equal(t, -1, compareVersions(v1, v2))
	require.Equal(t, 1, compareVersions(v2, v1))
}
