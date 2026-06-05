package datastore_test

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

var errFormat = errors.New("format failed")

func TestFindUniqueRef(t *testing.T) {
	t.Parallel()
	runUniqueRefTests(t, func(store datastore.AddressRefStore, ref datastore.AddressRef) (datastore.AddressRef, error) {
		return datastore.FindUniqueRef(store, ref)
	})
}

func TestFindAndFormatRef(t *testing.T) {
	t.Parallel()
	runUniqueRefTests(t, func(store datastore.AddressRefStore, ref datastore.AddressRef) (datastore.AddressRef, error) {
		return datastore.FindAndFormatRef(store, ref, func(r datastore.AddressRef) (datastore.AddressRef, error) {
			return r, nil
		})
	})
}

func runUniqueRefTests(t *testing.T, find func(datastore.AddressRefStore, datastore.AddressRef) (datastore.AddressRef, error)) {
	t.Helper()

	tests := []struct {
		desc        string
		makeStore   func(t *testing.T) datastore.AddressRefStore
		expectedErr error
		errContains []string
		ref         datastore.AddressRef
		want        datastore.AddressRef
	}{
		{
			desc: "find one ref",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t, datastore.AddressRef{
					ChainSelector: 4340886533089894000,
					Address:       "0x01",
					Type:          datastore.ContractType("TestContract"),
					Version:       semver.MustParse("1.0.0"),
					Qualifier:     "For testing",
				})
			},
			ref: datastore.AddressRef{
				ChainSelector: 4340886533089894000,
				Address:       "0x01",
				Type:          datastore.ContractType("TestContract"),
				Version:       semver.MustParse("1.0.0"),
				Qualifier:     "For testing",
			},
			want: datastore.AddressRef{
				ChainSelector: 4340886533089894000,
				Address:       "0x01",
				Type:          datastore.ContractType("TestContract"),
				Version:       semver.MustParse("1.0.0"),
				Qualifier:     "For testing",
			},
		},
		{
			desc: "ambiguous match",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t,
					datastore.AddressRef{
						ChainSelector: 4340886533089894000,
						Address:       "0x01",
						Type:          datastore.ContractType("TestContract"),
						Version:       semver.MustParse("1.0.0"),
						Qualifier:     "For testing",
					},
					datastore.AddressRef{
						ChainSelector: 4340886533089894000,
						Address:       "0x02",
						Type:          datastore.ContractType("TestContract"),
						Version:       semver.MustParse("1.0.0"),
						Qualifier:     "For production",
					},
				)
			},
			ref: datastore.AddressRef{
				ChainSelector: 4340886533089894000,
				Type:          datastore.ContractType("TestContract"),
				Version:       semver.MustParse("1.0.0"),
			},
			expectedErr: datastore.ErrAddressRefQueryAmbiguous,
			errContains: []string{"found 2"},
		},
		{
			desc: "no match",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return datastore.NewMemoryDataStore().Seal().Addresses()
			},
			ref: datastore.AddressRef{
				ChainSelector: 4340886533089894000,
				Type:          datastore.ContractType("TestContract"),
				Version:       semver.MustParse("1.0.0"),
				Qualifier:     "For testing",
			},
			expectedErr: datastore.ErrAddressRefQueryNoMatch,
			errContains: []string{"found 0"},
		},
		{
			desc: "empty query criteria",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t, datastore.AddressRef{
					ChainSelector: 1,
					Address:       "0xonly",
					Type:          datastore.ContractType("Only"),
					Version:       semver.MustParse("1.0.0"),
				})
			},
			ref:         datastore.AddressRef{},
			expectedErr: datastore.ErrAddressRefQueryEmpty,
		},
		{
			desc: "cross-chain ambiguous without chain selector",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t,
					datastore.AddressRef{
						ChainSelector: 111,
						Address:       "0x01",
						Type:          datastore.ContractType("Router"),
						Version:       semver.MustParse("1.6.0"),
					},
					datastore.AddressRef{
						ChainSelector: 222,
						Address:       "0x02",
						Type:          datastore.ContractType("Router"),
						Version:       semver.MustParse("1.6.0"),
					},
				)
			},
			ref: datastore.AddressRef{
				Type:    datastore.ContractType("Router"),
				Version: semver.MustParse("1.6.0"),
			},
			expectedErr: datastore.ErrAddressRefQueryAmbiguous,
			errContains: []string{"found 2"},
		},
		{
			desc: "cross-chain unique with chain selector",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t,
					datastore.AddressRef{
						ChainSelector: 111,
						Address:       "0x01",
						Type:          datastore.ContractType("Router"),
						Version:       semver.MustParse("1.6.0"),
					},
					datastore.AddressRef{
						ChainSelector: 222,
						Address:       "0x02",
						Type:          datastore.ContractType("Router"),
						Version:       semver.MustParse("1.6.0"),
					},
				)
			},
			ref: datastore.AddressRef{
				ChainSelector: 222,
				Type:          datastore.ContractType("Router"),
				Version:       semver.MustParse("1.6.0"),
			},
			want: datastore.AddressRef{
				ChainSelector: 222,
				Address:       "0x02",
				Type:          datastore.ContractType("Router"),
				Version:       semver.MustParse("1.6.0"),
			},
		},
		{
			desc: "same-chain ambiguous without version",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t,
					datastore.AddressRef{
						ChainSelector: 111,
						Address:       "0x01",
						Type:          datastore.ContractType("Router"),
						Version:       semver.MustParse("1.5.0"),
					},
					datastore.AddressRef{
						ChainSelector: 111,
						Address:       "0x02",
						Type:          datastore.ContractType("Router"),
						Version:       semver.MustParse("1.6.0"),
					},
				)
			},
			ref: datastore.AddressRef{
				ChainSelector: 111,
				Type:          datastore.ContractType("Router"),
			},
			expectedErr: datastore.ErrAddressRefQueryAmbiguous,
			errContains: []string{"found 2"},
		},
		{
			desc: "query labels ignored",
			makeStore: func(t *testing.T) datastore.AddressRefStore {
				t.Helper()
				return storeWithRefs(t, datastore.AddressRef{
					ChainSelector: 1,
					Address:       "0xlabeled",
					Type:          datastore.ContractType("Labeled"),
					Version:       semver.MustParse("1.0.0"),
					Labels:        datastore.NewLabelSet("stored"),
				})
			},
			ref: datastore.AddressRef{
				ChainSelector: 1,
				Type:          datastore.ContractType("Labeled"),
				Version:       semver.MustParse("1.0.0"),
				Labels:        datastore.NewLabelSet("query-only"),
			},
			want: datastore.AddressRef{
				ChainSelector: 1,
				Address:       "0xlabeled",
				Type:          datastore.ContractType("Labeled"),
				Version:       semver.MustParse("1.0.0"),
				Labels:        datastore.NewLabelSet("stored"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			store := test.makeStore(t)
			got, err := find(store, test.ref)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, test.expectedErr)
				for _, msg := range test.errContains {
					require.ErrorContains(t, err, msg)
				}

				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want.ChainSelector, got.ChainSelector)
			require.Equal(t, test.want.Address, got.Address)
			require.Equal(t, test.want.Type, got.Type)
			require.Equal(t, test.want.Qualifier, got.Qualifier)
			require.True(t, test.want.Version.Equal(got.Version))
			require.Equal(t, test.want.Labels, got.Labels)
		})
	}
}

func TestFindUniqueRef_errorShowsActiveQueryOnly(t *testing.T) {
	t.Parallel()

	store := storeWithRefs(t,
		datastore.AddressRef{
			ChainSelector: 111,
			Type:          datastore.ContractType("Router"),
			Version:       semver.MustParse("1.6.0"),
		},
		datastore.AddressRef{
			ChainSelector: 222,
			Type:          datastore.ContractType("Router"),
			Version:       semver.MustParse("1.6.0"),
		},
	)
	ref := datastore.AddressRef{
		Type:    datastore.ContractType("Router"),
		Version: semver.MustParse("1.6.0"),
	}

	_, err := datastore.FindUniqueRef(store, ref)
	require.ErrorIs(t, err, datastore.ErrAddressRefQueryAmbiguous)
	require.ErrorContains(t, err, "{Type: Router, Version: 1.6.0}")
	require.NotContains(t, err.Error(), "ChainSelector: 0")
}

func TestFindUniqueRef_returnsClone(t *testing.T) {
	t.Parallel()

	store := storeWithRefs(t, datastore.AddressRef{
		ChainSelector: 1,
		Address:       "0xlabeled",
		Type:          datastore.ContractType("Labeled"),
		Version:       semver.MustParse("1.0.0"),
		Labels:        datastore.NewLabelSet("stored"),
	})
	ref := datastore.AddressRef{
		ChainSelector: 1,
		Type:          datastore.ContractType("Labeled"),
		Version:       semver.MustParse("1.0.0"),
	}

	got, err := datastore.FindUniqueRef(store, ref)
	require.NoError(t, err)
	got.Labels.Add("mutated")
	got.Version = semver.MustParse("9.9.9")

	again, err := datastore.FindUniqueRef(store, ref)
	require.NoError(t, err)
	require.False(t, again.Labels.Contains("mutated"))
	require.True(t, again.Labels.Contains("stored"))
	require.Equal(t, "1.0.0", again.Version.String())
}

func TestFindAndFormatRef_nilFormat(t *testing.T) {
	t.Parallel()

	store := storeWithRefs(t, datastore.AddressRef{
		ChainSelector: 1,
		Type:          datastore.ContractType("TestContract"),
		Version:       semver.MustParse("1.0.0"),
	})
	ref := datastore.AddressRef{
		ChainSelector: 1,
		Type:          datastore.ContractType("TestContract"),
		Version:       semver.MustParse("1.0.0"),
	}

	_, err := datastore.FindAndFormatRef[string](store, ref, nil)
	require.ErrorIs(t, err, datastore.ErrAddressRefFormatFailed)
	require.ErrorContains(t, err, "format function is required")
}

func TestFindAndFormatRef_formatError(t *testing.T) {
	t.Parallel()

	store := storeWithRefs(t, datastore.AddressRef{
		ChainSelector: 1,
		Address:       "0x01",
		Type:          datastore.ContractType("TestContract"),
		Version:       semver.MustParse("1.0.0"),
	})
	ref := datastore.AddressRef{
		ChainSelector: 1,
		Type:          datastore.ContractType("TestContract"),
		Version:       semver.MustParse("1.0.0"),
	}

	_, err := datastore.FindAndFormatRef(store, ref, func(datastore.AddressRef) (string, error) {
		return "", errFormat
	})
	require.ErrorIs(t, err, datastore.ErrAddressRefFormatFailed)
	require.ErrorIs(t, err, errFormat)
	require.ErrorContains(t, err, "ref {ChainSelector: 1")
}

func TestFindAndFormatRef_typedFormat(t *testing.T) {
	t.Parallel()

	store := storeWithRefs(t, datastore.AddressRef{
		ChainSelector: 1,
		Address:       "0xabc",
		Type:          datastore.ContractType("TestContract"),
		Version:       semver.MustParse("2.0.0"),
	})
	ref := datastore.AddressRef{
		ChainSelector: 1,
		Type:          datastore.ContractType("TestContract"),
		Version:       semver.MustParse("2.0.0"),
	}

	got, err := datastore.FindAndFormatRef(store, ref, func(r datastore.AddressRef) (string, error) {
		return r.Address, nil
	})
	require.NoError(t, err)
	require.Equal(t, "0xabc", got)
}

func storeWithRefs(t *testing.T, refs ...datastore.AddressRef) datastore.AddressRefStore {
	t.Helper()
	ds := datastore.NewMemoryDataStore()
	for _, ref := range refs {
		require.NoError(t, ds.Addresses().Add(ref))
	}

	return ds.Seal().Addresses()
}
