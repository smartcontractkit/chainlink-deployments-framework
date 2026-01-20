package canton

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

func TestChain_ChainInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   uint64
		wantName   string
		wantString string
		wantFamily string
	}{
		{
			name:       "returns correct info",
			selector:   chainsel.CANTON_TESTNET.Selector,
			wantName:   chainsel.CANTON_TESTNET.Name,
			wantString: "canton-testnet (9268731218649498074)",
			wantFamily: chainsel.FamilyCanton,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			c := Chain{
				ChainMetadata: chaincommon.ChainMetadata{Selector: test.selector},
			}

			assert.Equal(t, test.selector, c.ChainSelector())
			assert.Equal(t, test.wantString, c.String())
			assert.Equal(t, test.wantName, c.Name())
			assert.Equal(t, test.wantFamily, c.Family())
		})
	}
}
