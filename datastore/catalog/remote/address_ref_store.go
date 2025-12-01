package remote

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

type catalogAddressRefStoreConfig struct {
	Domain      string
	Environment string
	Client      *CatalogClient
}

type catalogAddressRefStore struct {
	domain      string
	environment string
	client      *CatalogClient
}

// Ensure catalogAddressRefStore implements the V2 interface
var _ datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] = &catalogAddressRefStore{}

func newCatalogAddressRefStore(cfg catalogAddressRefStoreConfig) *catalogAddressRefStore {
	return &catalogAddressRefStore{
		domain:      cfg.Domain,
		environment: cfg.Environment,
		client:      cfg.Client,
	}
}

func (s *catalogAddressRefStore) Get(_ context.Context, key datastore.AddressRefKey, options ...datastore.GetOption) (datastore.AddressRef, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	return s.get(ignoreTransactions, key)
}

func (s *catalogAddressRefStore) get(
	ignoreTransaction bool,
	key datastore.AddressRefKey,
) (datastore.AddressRef, error) {
	// Create the find request with the key converted to a filter
	filter := s.keyToFilter(key)
	findRequest := &pb.AddressReferenceFindRequest{
		KeyFilter:         filter,
		IgnoreTransaction: ignoreTransaction,
	}

	// Create the request
	request := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_AddressReferenceFindRequest{
			AddressReferenceFindRequest: findRequest,
		},
	}

	// Create a bidirectional stream with the initial request for HMAC
	stream, err := s.client.DataAccess(request)
	if err != nil {
		return datastore.AddressRef{}, fmt.Errorf("failed to create data access stream: %w", err)
	}

	if sendErr := stream.Send(request); sendErr != nil {
		return datastore.AddressRef{}, fmt.Errorf("failed to send find request: %w", sendErr)
	}

	// Receive the response
	response, err := stream.Recv()
	if err != nil {
		return datastore.AddressRef{}, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if statusErr := parseResponseStatus(response.Status); statusErr != nil {
		st, sterr := parseStatusError(statusErr)
		if sterr != nil {
			return datastore.AddressRef{}, sterr
		}

		if st.Code() == codes.NotFound {
			return datastore.AddressRef{}, fmt.Errorf("%w: %s", datastore.ErrAddressRefNotFound, statusErr.Error())
		}

		return datastore.AddressRef{}, fmt.Errorf("get address ref failed: %w", statusErr)
	}

	// Extract the address find response
	findResponse := response.GetAddressReferenceFindResponse()
	if findResponse == nil {
		return datastore.AddressRef{}, errors.New("unexpected response type")
	}

	// Convert the response to datastore format
	if len(findResponse.References) == 0 {
		return datastore.AddressRef{}, datastore.ErrAddressRefNotFound
	}

	// Get the first matching reference
	protoRef := findResponse.References[0]
	addressRef, err := s.protoToAddressRef(protoRef)
	if err != nil {
		return datastore.AddressRef{}, fmt.Errorf("failed to convert proto to address ref: %w", err)
	}

	return addressRef, nil
}

// Fetch returns a copy of all AddressRef in the catalog.
func (s *catalogAddressRefStore) Fetch(_ context.Context) ([]datastore.AddressRef, error) {
	// Create the find request with an empty filter to get all records
	// We only filter by domain and environment to get all records for this store's scope
	filter := &pb.AddressReferenceKeyFilter{
		Domain:      wrapperspb.String(s.domain),
		Environment: wrapperspb.String(s.environment),
		// Leave other fields nil to fetch all records within the domain/environment
	}

	findRequest := &pb.AddressReferenceFindRequest{
		KeyFilter: filter,
	}

	// Create the request
	request := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_AddressReferenceFindRequest{
			AddressReferenceFindRequest: findRequest,
		},
	}

	// Create a bidirectional stream with the initial request for HMAC
	stream, err := s.client.DataAccess(request)
	if err != nil {
		return nil, fmt.Errorf("failed to create data access stream: %w", err)
	}

	if sendErr := stream.Send(request); sendErr != nil {
		return nil, fmt.Errorf("failed to send find request: %w", sendErr)
	}

	// Receive the response
	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if statusErr := parseResponseStatus(response.Status); statusErr != nil {
		st, sterr := parseStatusError(statusErr)
		if sterr != nil {
			return nil, sterr
		}

		if st.Code() == codes.NotFound {
			return nil, datastore.ErrAddressRefNotFound
		}

		return nil, fmt.Errorf("fetch address refs failed: %w", statusErr)
	}

	// Extract the address find response
	findResponse := response.GetAddressReferenceFindResponse()
	if findResponse == nil {
		return nil, errors.New("unexpected response type")
	}

	// Convert all protobuf references to datastore format
	addressRefs := make([]datastore.AddressRef, 0, len(findResponse.References))
	for _, protoRef := range findResponse.References {
		addressRef, convErr := s.protoToAddressRef(protoRef)
		if convErr != nil {
			return nil, fmt.Errorf("failed to convert proto to address ref: %w", convErr)
		}
		addressRefs = append(addressRefs, addressRef)
	}

	return addressRefs, nil
}

// Filter returns a copy of all AddressRef in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *catalogAddressRefStore) Filter(
	ctx context.Context,
	filters ...datastore.FilterFunc[datastore.AddressRefKey, datastore.AddressRef],
) ([]datastore.AddressRef, error) {
	// First, fetch all records from the catalog
	records, err := s.Fetch(ctx)
	if err != nil {
		// In case of error, return empty slice
		// In a more robust implementation, you might want to log this error
		return []datastore.AddressRef{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	// Apply each filter in sequence
	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *catalogAddressRefStore) Add(_ context.Context, record datastore.AddressRef) error {
	// Convert the datastore record to protobuf
	protoRef := s.addressRefToProto(record)

	// Create the edit request with INSERT semantics
	editRequest := &pb.AddressReferenceEditRequest{
		Record:    protoRef,
		Semantics: pb.EditSemantics_SEMANTICS_INSERT,
	}

	// Create the request
	editReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_AddressReferenceEditRequest{
			AddressReferenceEditRequest: editRequest,
		},
	}

	// Create a bidirectional stream with the initial request for HMAC
	stream, err := s.client.DataAccess(editReq)
	if err != nil {
		return fmt.Errorf("failed to create data access stream: %w", err)
	}

	if sendErr := stream.Send(editReq); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive the edit response
	editResponse, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive edit response: %w", err)
	}

	// Check for errors in the edit response
	if err := parseResponseStatus(editResponse.Status); err != nil {
		return fmt.Errorf("add address ref failed: %w", err)
	}

	// Extract the edit response to validate it
	editResp := editResponse.GetAddressReferenceEditResponse()
	if editResp == nil {
		return errors.New("unexpected edit response type")
	}

	return nil
}

func (s *catalogAddressRefStore) Upsert(_ context.Context, record datastore.AddressRef) error {
	// Convert the datastore record to protobuf
	protoRef := s.addressRefToProto(record)

	// Create the edit request with UPSERT semantics
	editRequest := &pb.AddressReferenceEditRequest{
		Record:    protoRef,
		Semantics: pb.EditSemantics_SEMANTICS_UPSERT,
	}

	// Create the request
	request := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_AddressReferenceEditRequest{
			AddressReferenceEditRequest: editRequest,
		},
	}

	// Create a bidirectional stream with the initial request for HMAC
	stream, err := s.client.DataAccess(request)
	if err != nil {
		return fmt.Errorf("failed to create data access stream: %w", err)
	}

	if sendErr := stream.Send(request); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive the response
	response, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if err := parseResponseStatus(response.Status); err != nil {
		return fmt.Errorf("upsert address ref failed: %w", err)
	}

	// Extract the edit response to validate it
	editResponse := response.GetAddressReferenceEditResponse()
	if editResponse == nil {
		return errors.New("unexpected response type")
	}

	return nil
}

func (s *catalogAddressRefStore) Update(ctx context.Context, record datastore.AddressRef) error {
	// First check if the record exists
	key := datastore.NewAddressRefKey(record.ChainSelector, record.Type, record.Version, record.Qualifier)
	_, err := s.Get(ctx, key)
	if errors.Is(err, datastore.ErrAddressRefNotFound) {
		// Record doesn't exist, return error
		return datastore.ErrAddressRefNotFound
	}
	if err != nil {
		// Some other error occurred during Get
		return fmt.Errorf("failed to check if record exists: %w", err)
	}

	// Record exists, proceed with updating it
	// Convert the datastore record to protobuf
	protoRef := s.addressRefToProto(record)

	// Create the edit request with UPDATE semantics
	editRequest := &pb.AddressReferenceEditRequest{
		Record:    protoRef,
		Semantics: pb.EditSemantics_SEMANTICS_UPDATE,
	}

	// Create the request
	editReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_AddressReferenceEditRequest{
			AddressReferenceEditRequest: editRequest,
		},
	}

	// Create a bidirectional stream with the initial request for HMAC
	stream, streamErr := s.client.DataAccess(editReq)
	if streamErr != nil {
		return fmt.Errorf("failed to create data access stream: %w", streamErr)
	}

	if sendErr := stream.Send(editReq); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive the edit response
	editResponse, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive edit response: %w", err)
	}

	// Check for errors in the edit response
	if err := parseResponseStatus(editResponse.Status); err != nil {
		return fmt.Errorf("update address ref failed: %w", err)
	}

	// Extract the edit response to validate it
	editResp := editResponse.GetAddressReferenceEditResponse()
	if editResp == nil {
		return errors.New("unexpected edit response type")
	}

	return nil
}

func (s *catalogAddressRefStore) Delete(_ context.Context, _ datastore.AddressRefKey) error {
	// The catalog API does not support delete operations
	// This is intentional as catalogs are typically immutable reference stores
	return errors.New("delete operation not supported by catalog API")
}

// keyToFilter converts a datastore.AddressRefKey to a protobuf AddressReferenceKeyFilter
func (s *catalogAddressRefStore) keyToFilter(key datastore.AddressRefKey) *pb.AddressReferenceKeyFilter {
	return &pb.AddressReferenceKeyFilter{
		Domain:        wrapperspb.String(s.domain),
		Environment:   wrapperspb.String(s.environment),
		ChainSelector: wrapperspb.UInt64(key.ChainSelector()),
		ContractType:  wrapperspb.String(string(key.Type())),
		Version:       wrapperspb.String(key.Version().String()),
		Qualifier:     wrapperspb.String(key.Qualifier()),
	}
}

// protoToAddressRef converts a protobuf AddressReference to a datastore.AddressRef
func (s *catalogAddressRefStore) protoToAddressRef(protoRef *pb.AddressReference) (datastore.AddressRef, error) {
	// Parse the version
	version, err := semver.NewVersion(protoRef.Version)
	if err != nil {
		return datastore.AddressRef{}, fmt.Errorf("failed to parse version %s: %w", protoRef.Version, err)
	}

	// Convert label set
	labelSet := datastore.NewLabelSet(protoRef.LabelSet...)

	return datastore.AddressRef{
		Address:       protoRef.Address,
		ChainSelector: protoRef.ChainSelector,
		Type:          datastore.ContractType(protoRef.ContractType),
		Version:       version,
		Qualifier:     protoRef.Qualifier,
		Labels:        labelSet,
	}, nil
}

// addressRefToProto converts a datastore.AddressRef to a protobuf AddressReference
func (s *catalogAddressRefStore) addressRefToProto(addressRef datastore.AddressRef) *pb.AddressReference {
	return &pb.AddressReference{
		Domain:        s.domain,
		Environment:   s.environment,
		ChainSelector: addressRef.ChainSelector,
		ContractType:  string(addressRef.Type),
		Version:       addressRef.Version.String(),
		Qualifier:     addressRef.Qualifier,
		Address:       addressRef.Address,
		LabelSet:      addressRef.Labels.List(),
	}
}
