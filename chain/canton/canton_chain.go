package canton

import (
	"context"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
)

type Chain struct {
	Selector uint64

	Participants []Participant
}

func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

func (c Chain) String() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.String()
}

func (c Chain) Name() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Name()
}

func (c Chain) Family() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Family()
}

type Participant struct {
	Name      string
	Endpoints blockchain.CantonParticipantEndpoints
	JWT       func(ctx context.Context) (string, error)
}
