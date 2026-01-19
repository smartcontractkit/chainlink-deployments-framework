package canton

import (
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
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
	Name        string
	Endpoints   ParticipantEndpoints
	JWTProvider JWTProvider
}

type ParticipantEndpoints struct {
	JSONLedgerAPIURL string // https://docs.digitalasset.com/build/3.5/reference/json-api/json-api.html
	GRPCLedgerAPIURL string // https://docs.digitalasset.com/build/3.5/reference/lapi-proto-docs.html
	AdminAPIURL      string // https://docs.digitalasset.com/operate/3.5/howtos/configure/apis/admin_api.html
	ValidatorAPIURL  string // https://docs.sync.global/app_dev/validator_api/index.html
}
