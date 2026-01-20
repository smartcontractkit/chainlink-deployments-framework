package canton

import (
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = chaincommon.ChainMetadata

// Chain represents a Canton network instance used CLDF.
// It contains chain metadata and augments it with Canton-specific information.
// In particular, it tracks all known Canton participants and their connection
// details so that callers can discover and interact with the network's APIs.
type Chain struct {
	ChainMetadata

	// List of participants in the Canton network
	// The number of participants depends on the config the provider has been initialized with
	Participants []Participant
}

// Participant represents a single Canton participant node in the network.
// A participant hosts parties and their local ledger state, and mediates all
// interactions with the Canton ledger for those parties via its exposed APIs
// (ledger, admin, validator, etc.). It is identified by a human-readable
// name, provides a set of API endpoints, and uses a JWT provider to issue
// authentication tokens for secure access to those endpoints.
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
