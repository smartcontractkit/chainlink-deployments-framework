package renderer

import (
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"
)

func TestCloneTimelockProposal_nil(t *testing.T) {
	t.Parallel()

	cloned, err := CloneTimelockProposal(nil)
	require.NoError(t, err)
	require.Nil(t, cloned)
}

func TestCloneTimelockProposal_isolatesMutation(t *testing.T) {
	t.Parallel()

	original := testTimelockProposalForClone("original description")
	chainSelector := mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)

	cloned, err := CloneTimelockProposal(original)
	require.NoError(t, err)
	require.NotSame(t, original, cloned)

	cloned.Description = "mutated description"
	cloned.Operations[0].Transactions[0].Data[0] = 0xff
	cloned.TimelockAddresses[chainSelector] = "0x3333333333333333333333333333333333333333"

	require.Equal(t, "original description", original.Description)
	require.Equal(t, byte(0x01), original.Operations[0].Transactions[0].Data[0])
	require.Equal(t, "0x2222222222222222222222222222222222222222", original.TimelockAddresses[chainSelector])
}

func testTimelockProposalForClone(description string) *mcms.TimelockProposal {
	chainSelector := mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)

	return &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmstypes.KindTimelockProposal,
			ValidUntil:  uint32(time.Now().Add(time.Hour).Unix()), //nolint:gosec // test fixture
			Description: description,
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				chainSelector: {MCMAddress: "0x1111111111111111111111111111111111111111"},
			},
		},
		Action: mcmstypes.TimelockActionSchedule,
		Delay:  mcmstypes.NewDuration(time.Hour),
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			chainSelector: "0x2222222222222222222222222222222222222222",
		},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: chainSelector,
				Transactions: []mcmstypes.Transaction{
					{To: "0x1111111111111111111111111111111111111111", Data: []byte{0x01, 0x02}},
				},
			},
		},
	}
}
