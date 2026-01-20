package canton

import (
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type Chain struct {
	chaincommon.ChainMetadata

	// List of participants in the Canton network
	// The number of participants depends on the config the provider has been initialized with
	Participants []Participant
}

type Participant struct {
	// A human-readable name for the participant
	Name string
	// The endpoints to interact with the participant's APIs
	Endpoints ParticipantEndpoints
	// A JWT provider instance to generate JWTs for authentication with the participant's APIs
	JWTProvider JWTProvider
}

// ParticipantEndpoints holds all available API endpoints for a Canton participant
type ParticipantEndpoints struct {
	// (HTTP) The URL to access the participant's JSON Ledger API
	// https://docs.digitalasset.com/build/3.5/reference/json-api/json-api.html
	JSONLedgerAPIURL string
	// (gRPC) The URL to access the participant's gRPC Ledger API
	// https://docs.digitalasset.com/build/3.5/reference/lapi-proto-docs.html
	GRPCLedgerAPIURL string
	// (gRPC) The URL to access the participant's Admin API
	// https://docs.digitalasset.com/operate/3.5/howtos/configure/apis/admin_api.html
	AdminAPIURL string
	// (HTTP) The URL to access the participant's Validator API
	// https://docs.sync.global/app_dev/validator_api/index.html
	ValidatorAPIURL string
}
