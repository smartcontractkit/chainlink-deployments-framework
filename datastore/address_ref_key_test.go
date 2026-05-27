package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressRefKey_Equals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key1     AddressRefKey
		key2     AddressRefKey
		expected bool
	}{
		{
			name:     "Identical keys",
			key1:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			key2:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			expected: true,
		},
		{
			name:     "Different chainSelector",
			key1:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			key2:     NewAddressRefKey(2, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			expected: false,
		},
		{
			name:     "Different contractType",
			key1:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			key2:     NewAddressRefKey(1, ContractType("typeB"), semver.MustParse("1.0.0"), "qualifier1"),
			expected: false,
		},
		{
			name:     "Different version",
			key1:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			key2:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("2.0.0"), "qualifier1"),
			expected: false,
		},
		{
			name:     "Different qualifier",
			key1:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier1"),
			key2:     NewAddressRefKey(1, ContractType("typeA"), semver.MustParse("1.0.0"), "qualifier2"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.key1.Equals(tt.key2))
		})
	}
}

func TestNewAddressRefKey(t *testing.T) {
	t.Parallel()

	version := semver.MustParse("1.0.0")
	key := NewAddressRefKey(1, ContractType("typeA"), version, "qualifier1")

	assert.Equal(t, uint64(1), key.ChainSelector(), "ChainSelector should match")
	assert.Equal(t, ContractType("typeA"), key.Type(), "ContractType should match")
	assert.Equal(t, version, key.Version(), "Version should match")
	assert.Equal(t, "qualifier1", key.Qualifier(), "Qualifier should match")
}

func TestAddressRefKey_String(t *testing.T) {
	t.Parallel()

	key := NewAddressRefKey(42, ContractType("MyType"), semver.MustParse("1.2.3"), "qual")
	expected := "42_MyType_1.2.3_qual"
	assert.Equal(t, expected, key.String())
}

func TestNewAddressRefKeyFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		want    AddressRefKey
		wantErr string
	}{
		{
			name: "success: valid string",
			give: "42_MyContract_1.2.3_primary",
			want: NewAddressRefKey(42, ContractType("MyContract"), semver.MustParse("1.2.3"), "primary"),
		},
		{
			name:    "failure: too few parts",
			give:    "42_MyContract_1.2.3",
			wantErr: "invalid address ref key",
		},
		{
			name:    "failure: too many parts",
			give:    "42_MyContract_1.2.3_primary_extra",
			wantErr: "invalid address ref key",
		},
		{
			name:    "failure: invalid chain selector",
			give:    "notanumber_MyContract_1.2.3_primary",
			wantErr: "failed to parse chain selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewAddressRefKeyFromString(tt.give)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.ChainSelector(), got.ChainSelector())
			assert.Equal(t, tt.want.Type(), got.Type())
			assert.True(t, tt.want.Version().Equal(got.Version()))
			assert.Equal(t, tt.want.Qualifier(), got.Qualifier())
		})
	}
}
