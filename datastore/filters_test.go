package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func TestAddressRefByAddress(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x123456",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x123456",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}

		recordThree = AddressRef{
			Address:       "0x456789",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveAddress    string
		expectedResult []AddressRef
	}{
		{
			name: "success: returns 2 records with given address",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveAddress:    "0x123456",
			expectedResult: []AddressRef{recordOne, recordTwo},
		},
		{
			name: "success: returns 1 record with given address",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveAddress:    "0x456789",
			expectedResult: []AddressRef{recordThree},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := AddressRefByAddress(tt.giveAddress)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestAddressRefByChainSelector(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveChain      uint64
		expectedResult []AddressRef
	}{
		{
			name: "success: returns record with given chain",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveChain:      2,
			expectedResult: []AddressRef{recordTwo},
		},
		{
			name: "success: returns no record with given chain",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveChain:      5,
			expectedResult: []AddressRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := AddressRefByChainSelector(tt.giveChain)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestAddressRefByType(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveType       ContractType
		expectedResult []AddressRef
	}{
		{
			name: "success: returns record with given type",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveType: "typeX",
			expectedResult: []AddressRef{
				recordTwo,
			},
		},
		{
			name: "success: returns no record with given type",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveType:       "typeL",
			expectedResult: []AddressRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := AddressRefByType(tt.giveType)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestAddressRefByVersion(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveVersion    *semver.Version
		expectedResult []AddressRef
	}{
		{
			name: "success: returns record with given version",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveVersion: semver.MustParse("0.5.0"),
			expectedResult: []AddressRef{
				recordOne,
				recordTwo,
			},
		},
		{
			name: "success: returns no record with given version",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveVersion:    semver.MustParse("0.6.0"),
			expectedResult: []AddressRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := AddressRefByVersion(tt.giveVersion)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestAddressRefByQualifier(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual2",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveQualifier  string
		expectedResult []AddressRef
	}{
		{
			name: "success: returns record with given qualifier",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveQualifier: "qual1",
			expectedResult: []AddressRef{
				recordOne,
			},
		},
		{
			name: "success: returns no record with given qualifier",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveQualifier:  "qual32",
			expectedResult: []AddressRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := AddressRefByQualifier(tt.giveQualifier)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestContractMetadataByChainSelector(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "Record1", ChainSelector: 0},
		}
		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "Record2", ChainSelector: 0},
		}
		recordThree = ContractMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "Record3", ChainSelector: 0},
		}
	)

	tests := []struct {
		name           string
		givenState     []ContractMetadata
		giveChain      uint64
		expectedResult []ContractMetadata
	}{
		{
			name: "success: returns records with given chain",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveChain: 1,
			expectedResult: []ContractMetadata{
				recordOne,
				recordThree,
			},
		},
		{
			name: "success: returns no records with given chain",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveChain:      3,
			expectedResult: []ContractMetadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := ContractMetadataByChainSelector(tt.giveChain)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestChainMetadataByChainSelector(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "Record1", ChainSelector: 0},
		}
		recordTwo = ChainMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "Record2", ChainSelector: 0},
		}
		recordThree = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "Record3", ChainSelector: 0},
		}
	)

	tests := []struct {
		name           string
		givenState     []ChainMetadata
		giveChain      uint64
		expectedResult []ChainMetadata
	}{
		{
			name: "success: returns records with given chain",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveChain: 1,
			expectedResult: []ChainMetadata{
				recordOne,
				recordThree,
			},
		},
		{
			name: "success: returns no records with given chain",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveChain:      3,
			expectedResult: []ChainMetadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := ChainMetadataByChainSelector(tt.giveChain)
			filteredRecords := filter(tt.givenState)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}
