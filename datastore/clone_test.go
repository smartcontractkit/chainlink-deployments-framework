package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

type testStruct struct {
	A uint64
	B string
	C []int
}

type chanStruct struct {
	A  uint64
	Ch chan int
}

func TestClone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    any
		wantErr string
	}{
		{
			name: "simple struct",
			give: testStruct{A: chainsel.APTOS_MAINNET.Selector, B: "foo", C: []int{1, 2, 3}},
		},
		{
			name:    "struct with channel",
			give:    chanStruct{A: chainsel.APTOS_MAINNET.Selector, Ch: make(chan int)},
			wantErr: "json: unsupported type: chan int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clone, err := clone(tt.give)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err, "clone should not return an error for %s", tt.name)
				typed, err := As[testStruct](clone)
				require.NoError(t, err, "As should not return an error for %s", tt.name)
				require.Equal(t, tt.give, typed)
			}
		})
	}
}
