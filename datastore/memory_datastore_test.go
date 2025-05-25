package datastore

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestMemoryDataStore_Merge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		setup               func() (*MemoryDataStore, *MemoryDataStore)
		expectAddresses     []AddressRef
		expectContractMetas []ContractMetadata
		expectEnvMeta       *EnvMetadata
	}{
		{
			name: "Merge single address",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()
				err := dataStore2.Addresses().Upsert(AddressRef{
					Address:   "0x123",
					Type:      "type1",
					Version:   semver.MustParse("1.0.0"),
					Qualifier: "qualifier1",
				})
				require.NoError(t, err)
				return dataStore1, dataStore2
			},
			expectAddresses: []AddressRef{{
				Address:   "0x123",
				Type:      "type1",
				Version:   semver.MustParse("1.0.0"),
				Qualifier: "qualifier1",
			}},
			expectContractMetas: nil,
			expectEnvMeta:       nil,
		},
		{
			name: "Match existing address with labels",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()
				err := dataStore1.Addresses().Upsert(AddressRef{
					Address:   "0x123",
					Type:      "type1",
					Version:   semver.MustParse("1.0.0"),
					Qualifier: "qualifier1",
					Labels:    NewLabelSet("label1"),
				})
				require.NoError(t, err)
				err = dataStore2.Addresses().Upsert(AddressRef{
					Address:   "0x123",
					Type:      "type1",
					Version:   semver.MustParse("1.0.0"),
					Qualifier: "qualifier1",
					Labels:    NewLabelSet("label2"),
				})
				require.NoError(t, err)
				return dataStore1, dataStore2
			},
			expectAddresses: []AddressRef{{
				Address:   "0x123",
				Type:      "type1",
				Version:   semver.MustParse("1.0.0"),
				Qualifier: "qualifier1",
				Labels:    NewLabelSet("label2"),
			}},
			expectContractMetas: nil,
			expectEnvMeta:       nil,
		},
		{
			name: "Merge contract metadata and env metadata",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()
				contractMeta := ContractMetadata{
					Address:       "0x456",
					ChainSelector: 99,
					Metadata:      TestMetadata{Data: "meta", Version: 1, Tags: []string{"a", "b"}, Extra: map[string]string{"k": "v"}, Nested: NestedMeta{Flag: true, Detail: "bar"}},
				}
				err := dataStore2.ContractMetadata().Upsert(contractMeta)
				require.NoError(t, err)
				envMeta := EnvMetadata{Metadata: TestMetadata{Data: "env", Version: 2, Tags: []string{"x", "y"}, Extra: map[string]string{"foo": "bar"}, Nested: NestedMeta{Flag: false, Detail: "baz"}}}
				err = dataStore2.EnvMetadata().Set(envMeta)
				require.NoError(t, err)
				return dataStore1, dataStore2
			},
			expectAddresses: nil,
			expectContractMetas: []ContractMetadata{{
				Address:       "0x456",
				ChainSelector: 99,
				Metadata:      TestMetadata{Data: "meta", Version: 1, Tags: []string{"a", "b"}, Extra: map[string]string{"k": "v"}, Nested: NestedMeta{Flag: true, Detail: "bar"}},
			}},
			expectEnvMeta: &EnvMetadata{Metadata: TestMetadata{Data: "env", Version: 2, Tags: []string{"x", "y"}, Extra: map[string]string{"foo": "bar"}, Nested: NestedMeta{Flag: false, Detail: "baz"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dataStore1, dataStore2 := tt.setup()
			err := dataStore1.Merge(dataStore2.Seal())
			require.NoError(t, err)

			// Check addresses
			addressRefs, err := dataStore1.Addresses().Fetch()
			require.NoError(t, err)
			if tt.expectAddresses != nil {
				require.Len(t, addressRefs, len(tt.expectAddresses))
				for i, exp := range tt.expectAddresses {
					require.Equal(t, exp.Address, addressRefs[i].Address)
					require.Equal(t, exp.Type, addressRefs[i].Type)
					require.Equal(t, exp.Version.String(), addressRefs[i].Version.String())
					require.Equal(t, exp.Qualifier, addressRefs[i].Qualifier)
					require.Equal(t, exp.Labels.String(), addressRefs[i].Labels.String())
				}
			} else {
				require.Len(t, addressRefs, 0)
			}

			// Check contract metadata
			contractMeta, err := dataStore1.ContractMetadata().Fetch()
			require.NoError(t, err)
			if tt.expectContractMetas != nil {
				require.Len(t, contractMeta, len(tt.expectContractMetas))
				for i, exp := range tt.expectContractMetas {
					require.Equal(t, exp.Address, contractMeta[i].Address)
					require.Equal(t, exp.ChainSelector, contractMeta[i].ChainSelector)
					require.Equal(t, exp.Metadata, contractMeta[i].Metadata)
				}
			} else {
				require.Len(t, contractMeta, 0)
			}

			// Check env metadata
			if tt.expectEnvMeta != nil {
				envMeta, err := dataStore1.EnvMetadata().Get()
				require.NoError(t, err)
				require.Equal(t, tt.expectEnvMeta.Metadata, envMeta.Metadata)
			} else {
				_, err := dataStore1.EnvMetadata().Get()
				require.Error(t, err)
			}
		})
	}
}

func TestMemoryDataStore_JSONSerialization(t *testing.T) {
	t.Parallel()

	testMeta := TestMetadata{
		Data:    "foo",
		Version: 42,
		Tags:    []string{"a", "b"},
		Extra:   map[string]string{"k": "v"},
		Nested:  NestedMeta{Flag: true, Detail: "bar"},
	}

	tests := []struct {
		name          string
		addressRef    AddressRef
		contractMeta  ContractMetadata
		envMeta       EnvMetadata
		expectMeta    CustomMetadata
		expectEnvMeta CustomMetadata
	}{
		{
			name: "success with TestMetadata",
			addressRef: AddressRef{
				Address:   "0xdef",
				Type:      ContractType("testType2"),
				Version:   semver.MustParse("2.3.4"),
				Qualifier: "qual2",
			},
			contractMeta: ContractMetadata{
				Address:       "0xdef",
				ChainSelector: 2,
				Metadata:      testMeta,
			},
			envMeta:       EnvMetadata{Metadata: testMeta},
			expectMeta:    testMeta,
			expectEnvMeta: testMeta,
		},
		{
			name: "success with nil metadata",
			addressRef: AddressRef{
				Address:   "0xabc",
				Type:      ContractType("testType"),
				Version:   semver.MustParse("1.2.3"),
				Qualifier: "qual",
			},
			contractMeta: ContractMetadata{
				Address:       "0xabc",
				ChainSelector: 1,
				Metadata:      nil,
			},
			envMeta:       EnvMetadata{Metadata: nil},
			expectMeta:    nil,
			expectEnvMeta: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ds := NewMemoryDataStore()
			err := ds.Addresses().Add(tc.addressRef)
			require.NoError(t, err)
			err = ds.ContractMetadata().Add(tc.contractMeta)
			require.NoError(t, err)
			err = ds.EnvMetadata().Set(tc.envMeta)
			require.NoError(t, err)

			jsonBytes, err := json.Marshal(ds)
			require.NoError(t, err)

			var ds2 MemoryDataStore
			err = json.Unmarshal(jsonBytes, &ds2)
			require.NoError(t, err)

			addressRefs, err := ds2.Addresses().Fetch()
			require.NoError(t, err)
			require.Len(t, addressRefs, 1)
			require.Equal(t, tc.addressRef.Address, addressRefs[0].Address)
			require.Equal(t, tc.addressRef.Type, addressRefs[0].Type)
			require.Equal(t, tc.addressRef.Version.String(), addressRefs[0].Version.String())

			contractMetas, err := ds2.ContractMetadata().Fetch()
			require.NoError(t, err)
			require.Len(t, contractMetas, 1)
			require.Equal(t, tc.contractMeta.Address, contractMetas[0].Address)
			require.Equal(t, tc.contractMeta.ChainSelector, contractMetas[0].ChainSelector)
			if tc.expectMeta == nil {
				// Should be json.RawMessage("null") or equivalent
				b, err := json.Marshal(contractMetas[0].Metadata)
				require.NoError(t, err)
				require.Equal(t, []byte("null"), b)
			} else {
				cm, err := As[TestMetadata](contractMetas[0].Metadata)
				require.NoError(t, err)
				require.Equal(t, tc.expectMeta, cm)
			}

			envMeta, err := ds2.EnvMetadata().Get()
			require.NoError(t, err)
			if tc.expectEnvMeta == nil {
				b, err := json.Marshal(envMeta.Metadata)
				require.NoError(t, err)
				require.Equal(t, []byte("null"), b)
			} else {
				cm, err := As[TestMetadata](envMeta.Metadata)
				require.NoError(t, err)
				require.Equal(t, tc.expectEnvMeta, cm)
			}
		})
	}
}
