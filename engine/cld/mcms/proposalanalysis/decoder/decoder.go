package decoder

import (
	"context"
	"encoding/json"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type DecodedTimelockProposal interface {
	BatchOperations() DecodedBatchOperations
}

type DecodedBatchOperations []DecodedBatchOperation

type DecodedBatchOperation interface {
	ChainSelector() uint64
	Calls() DecodedCalls
}

type DecodedCalls []DecodedCall

type DecodedCall interface {
	To() string   // legacy analyzer uses "Address"
	Name() string // legacy analyzer uses "Method"
	Inputs() DecodedParameters
	Outputs() DecodedParameters
	Data() []byte
	AdditionalFields() json.RawMessage
	ContractType() string
	ContractVersion() string
}

type DecodedParameters []DecodedParameter

type DecodedParameter interface {
	Name() string
	Type() string
	Value() any
}

// ProposalDecoder decodes MCMS proposals into structured DecodedTimelockProposal
type ProposalDecoder interface {
	Decode(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (DecodedTimelockProposal, error)
}
