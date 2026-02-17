package canton

import (
	"golang.org/x/oauth2"
	"google.golang.org/grpc"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	adminv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2/admin"
	participantv30 "github.com/digital-asset/dazl-client/v8/go/api/com/digitalasset/canton/admin/participant/v30"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = chaincommon.ChainMetadata

// Chain represents a Canton network instance initialized by CLDF.
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
// name, provides a set of API endpoints, and uses a TokenSource to issue
// authentication tokens for secure access to those endpoints.
type Participant struct {
	// A human-readable name for the participant
	Name string
	// The endpoints to interact with the participant's APIs
	Endpoints ParticipantEndpoints
	// The set of service clients to interact with the participant's Ledger API.
	// All clients are ready-to-use and are already configured with the correct authentication.
	LedgerServices LedgerServiceClients
	// (Optional) The set of service clients to interact with the participant's Admin API.
	// Will only be populated if the participant has been configured with an Admin API URL
	AdminServices *AdminServiceClients
	// An OAuth2 token source to obtain access tokens for authentication with the participant's APIs
	TokenSource oauth2.TokenSource
	// The UserID that will be used to interact with this participant.
	// The TokenSource will return access tokens containing this UserID as a subject claim.
	UserID string
	// The PartyID that will be used to interact with this participant.
	PartyID string
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
	// This also serves the Scan Proxy API, which provides access to the Global Scan and Token Standard APIs:
	// https://docs.sync.global/app_dev/validator_api/index.html#validator-api-scan-proxy
	ValidatorAPIURL string
}

// LedgerAdminServiceClients provides all available Ledger API admin gRPC service clients.
type LedgerAdminServiceClients struct {
	CommandInspection      adminv2.CommandInspectionServiceClient
	IdentityProviderConfig adminv2.IdentityProviderConfigServiceClient
	PackageManagement      adminv2.PackageManagementServiceClient
	ParticipantPruning     adminv2.ParticipantPruningServiceClient
	PartyManagement        adminv2.PartyManagementServiceClient
	UserManagement         adminv2.UserManagementServiceClient
}

// LedgerServiceClients provides all available Ledger API gRPC service clients.
type LedgerServiceClients struct {
	CommandCompletion apiv2.CommandCompletionServiceClient
	Command           apiv2.CommandServiceClient
	CommandSubmission apiv2.CommandSubmissionServiceClient
	EventQuery        apiv2.EventQueryServiceClient
	PackageService    apiv2.PackageServiceClient
	State             apiv2.StateServiceClient
	Update            apiv2.UpdateServiceClient
	Version           apiv2.VersionServiceClient
	// Ledger API admin clients
	// These endpoints can only be accessed if the user the participant has been configured with
	// has admin rights. Access with caution and only if you're certain that admin rights are available.
	Admin LedgerAdminServiceClients
}

// CreateLedgerServiceClients creates all LedgerServiceClients given a gRPC client connection.
func CreateLedgerServiceClients(conn grpc.ClientConnInterface) LedgerServiceClients {
	return LedgerServiceClients{
		Admin: LedgerAdminServiceClients{
			CommandInspection:      adminv2.NewCommandInspectionServiceClient(conn),
			IdentityProviderConfig: adminv2.NewIdentityProviderConfigServiceClient(conn),
			PackageManagement:      adminv2.NewPackageManagementServiceClient(conn),
			ParticipantPruning:     adminv2.NewParticipantPruningServiceClient(conn),
			PartyManagement:        adminv2.NewPartyManagementServiceClient(conn),
			UserManagement:         adminv2.NewUserManagementServiceClient(conn),
		},
		CommandCompletion: apiv2.NewCommandCompletionServiceClient(conn),
		Command:           apiv2.NewCommandServiceClient(conn),
		CommandSubmission: apiv2.NewCommandSubmissionServiceClient(conn),
		EventQuery:        apiv2.NewEventQueryServiceClient(conn),
		PackageService:    apiv2.NewPackageServiceClient(conn),
		State:             apiv2.NewStateServiceClient(conn),
		Update:            apiv2.NewUpdateServiceClient(conn),
		Version:           apiv2.NewVersionServiceClient(conn),
	}
}

// AdminServiceClients provides all available Admin API service clients.
// These services can only be accessed if the user the participant has been configured with has
// admin rights.
type AdminServiceClients struct {
	Package                  participantv30.PackageServiceClient
	ParticipantInspection    participantv30.ParticipantInspectionServiceClient
	ParticipantRepair        participantv30.ParticipantRepairServiceClient
	ParticipantStatus        participantv30.ParticipantStatusServiceClient
	PartyManagement          participantv30.PartyManagementServiceClient
	Ping                     participantv30.PingServiceClient
	Pruning                  participantv30.PruningServiceClient
	ResourceManagement       participantv30.ResourceManagementServiceClient
	SynchronizerConnectivity participantv30.SynchronizerConnectivityServiceClient
	TrafficControl           participantv30.TrafficControlServiceClient
}

// CreateAdminServiceClients creates all AdminServiceClients given a gRPC client connection.
func CreateAdminServiceClients(conn grpc.ClientConnInterface) AdminServiceClients {
	return AdminServiceClients{
		Package:                  participantv30.NewPackageServiceClient(conn),
		ParticipantInspection:    participantv30.NewParticipantInspectionServiceClient(conn),
		ParticipantRepair:        participantv30.NewParticipantRepairServiceClient(conn),
		ParticipantStatus:        participantv30.NewParticipantStatusServiceClient(conn),
		PartyManagement:          participantv30.NewPartyManagementServiceClient(conn),
		Ping:                     participantv30.NewPingServiceClient(conn),
		Pruning:                  participantv30.NewPruningServiceClient(conn),
		ResourceManagement:       participantv30.NewResourceManagementServiceClient(conn),
		SynchronizerConnectivity: participantv30.NewSynchronizerConnectivityServiceClient(conn),
		TrafficControl:           participantv30.NewTrafficControlServiceClient(conn),
	}
}
