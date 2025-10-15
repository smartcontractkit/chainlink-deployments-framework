package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

type catalogContractMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      *CatalogClient
}

type catalogContractMetadataStore struct {
	domain      string
	environment string
	client      *CatalogClient
	// versionCache tracks the current version of each record for optimistic concurrency control
	mu           sync.RWMutex
	versionCache map[string]int32
}

var _ datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] = &catalogContractMetadataStore{}

func newCatalogContractMetadataStore(cfg catalogContractMetadataStoreConfig) *catalogContractMetadataStore {
	return &catalogContractMetadataStore{
		domain:       cfg.Domain,
		environment:  cfg.Environment,
		client:       cfg.Client,
		versionCache: make(map[string]int32),
	}
}

// getVersion retrieves the cached version for a record, defaulting to 0 for new records
func (s *catalogContractMetadataStore) getVersion(key datastore.ContractMetadataKey) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cacheKey := key.String()
	if version, exists := s.versionCache[cacheKey]; exists {
		return version
	}

	return 0 // Default version for new records
}

// setVersion updates the cached version for a record
func (s *catalogContractMetadataStore) setVersion(key datastore.ContractMetadataKey, version int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheKey := key.String()
	s.versionCache[cacheKey] = version
}

// keyToFilter converts a ContractMetadataKey to a ContractMetadataKeyFilter for gRPC requests
func (s *catalogContractMetadataStore) keyToFilter(key datastore.ContractMetadataKey) *pb.ContractMetadataKeyFilter {
	return &pb.ContractMetadataKeyFilter{
		Domain:        wrapperspb.String(s.domain),
		Environment:   wrapperspb.String(s.environment),
		ChainSelector: wrapperspb.UInt64(key.ChainSelector()),
		Address:       wrapperspb.String(key.Address()),
	}
}

// protoToContractMetadata converts a protobuf ContractMetadata to a datastore ContractMetadata
func (s *catalogContractMetadataStore) protoToContractMetadata(protoRecord *pb.ContractMetadata) (datastore.ContractMetadata, error) {
	var metadata any
	if protoRecord.Metadata != "" {
		if err := json.Unmarshal([]byte(protoRecord.Metadata), &metadata); err != nil {
			return datastore.ContractMetadata{}, fmt.Errorf("failed to unmarshal metadata JSON: %w", err)
		}
	}

	return datastore.ContractMetadata{
		Address:       protoRecord.Address,
		ChainSelector: protoRecord.ChainSelector,
		Metadata:      metadata,
	}, nil
}

// contractMetadataToProto converts a datastore ContractMetadata to a protobuf ContractMetadata
func (s *catalogContractMetadataStore) contractMetadataToProto(record datastore.ContractMetadata, version int32) *pb.ContractMetadata {
	var metadataJSON string
	if record.Metadata != nil {
		if metadataBytes, err := json.Marshal(record.Metadata); err == nil {
			metadataJSON = string(metadataBytes)
		}
	}

	return &pb.ContractMetadata{
		Domain:        s.domain,
		Environment:   s.environment,
		ChainSelector: record.ChainSelector,
		Address:       record.Address,
		Metadata:      metadataJSON,
		RowVersion:    version,
	}
}
func (s *catalogContractMetadataStore) Get(
	_ context.Context,
	key datastore.ContractMetadataKey,
	options ...datastore.GetOption,
) (datastore.ContractMetadata, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	return s.get(ignoreTransactions, key)
}

func (s *catalogContractMetadataStore) get(ignoreTransaction bool, key datastore.ContractMetadataKey) (datastore.ContractMetadata, error) {
	stream, err := s.client.DataAccess()
	if err != nil {
		return datastore.ContractMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Send find request
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ContractMetadataFindRequest{
			ContractMetadataFindRequest: &pb.ContractMetadataFindRequest{
				KeyFilter:         s.keyToFilter(key),
				IgnoreTransaction: ignoreTransaction,
			},
		},
	}

	if sendErr := stream.Send(findReq); sendErr != nil {
		return datastore.ContractMetadata{}, fmt.Errorf("failed to send find request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return datastore.ContractMetadata{}, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if statusErr := parseResponseStatus(resp.Status); statusErr != nil {
		st, sterr := parseStatusError(statusErr)
		if sterr != nil {
			return datastore.ContractMetadata{}, sterr
		}

		if st.Code() == codes.NotFound {
			return datastore.ContractMetadata{}, fmt.Errorf("%w: %s", datastore.ErrContractMetadataNotFound, statusErr.Error())
		}

		return datastore.ContractMetadata{}, fmt.Errorf("get contract metadata failed: %w", statusErr)
	}

	findResp := resp.GetContractMetadataFindResponse()
	if findResp == nil {
		return datastore.ContractMetadata{}, errors.New("unexpected response type")
	}

	if len(findResp.References) == 0 {
		return datastore.ContractMetadata{}, datastore.ErrContractMetadataNotFound
	}

	protoRecord := findResp.References[0]
	record, err := s.protoToContractMetadata(protoRecord)
	if err != nil {
		return datastore.ContractMetadata{}, err
	}

	// Cache the version for future operations
	s.setVersion(record.Key(), protoRecord.RowVersion)

	return record, nil
}

// Fetch returns a copy of all ContractMetadata in the catalog.
func (s *catalogContractMetadataStore) Fetch(_ context.Context) ([]datastore.ContractMetadata, error) {
	stream, err := s.client.DataAccess()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Send find request with domain and environment filter only (fetch all)
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ContractMetadataFindRequest{
			ContractMetadataFindRequest: &pb.ContractMetadataFindRequest{
				KeyFilter: &pb.ContractMetadataKeyFilter{
					Domain:      wrapperspb.String(s.domain),
					Environment: wrapperspb.String(s.environment),
				},
			},
		},
	}

	if sendErr := stream.Send(findReq); sendErr != nil {
		return nil, fmt.Errorf("failed to send find request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if statusErr := parseResponseStatus(resp.Status); statusErr != nil {
		return nil, fmt.Errorf("fetch contract metadata failed: %w", statusErr)
	}

	findResp := resp.GetContractMetadataFindResponse()
	if findResp == nil {
		return nil, errors.New("unexpected response type")
	}

	records := make([]datastore.ContractMetadata, 0, len(findResp.References))
	for _, protoRecord := range findResp.References {
		record, convErr := s.protoToContractMetadata(protoRecord)
		if convErr != nil {
			return nil, fmt.Errorf("failed to convert proto to contract metadata: %w", err)
		}

		// Cache the version for future operations
		s.setVersion(record.Key(), protoRecord.RowVersion)

		records = append(records, record)
	}

	return records, nil
}

// Filter returns a copy of all ContractMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *catalogContractMetadataStore) Filter(ctx context.Context, filters ...datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]) ([]datastore.ContractMetadata, error) {
	records, err := s.Fetch(ctx)
	if err != nil {
		return []datastore.ContractMetadata{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *catalogContractMetadataStore) Add(_ context.Context, record datastore.ContractMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_INSERT)
}

func (s *catalogContractMetadataStore) Upsert(ctx context.Context, key datastore.ContractMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record for merging
	currentRecord, err := s.Get(ctx, key)
	if err != nil {
		// If record doesn't exist, just insert the new record directly
		if errors.Is(err, datastore.ErrContractMetadataNotFound) {
			record := datastore.ContractMetadata{
				Address:       key.Address(),
				ChainSelector: key.ChainSelector(),
				Metadata:      metadata,
			}

			return s.editRecord(record, pb.EditSemantics_SEMANTICS_INSERT)
		}

		return fmt.Errorf("failed to get current record for upsert: %w", err)
	}

	// Record exists, apply the updater to merge with existing metadata
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ContractMetadata{
		Address:       key.Address(),
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPSERT)
}

func (s *catalogContractMetadataStore) Update(ctx context.Context, key datastore.ContractMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record - it must exist for update
	currentRecord, err := s.Get(ctx, key)
	if err != nil {
		if errors.Is(err, datastore.ErrContractMetadataNotFound) {
			return datastore.ErrContractMetadataNotFound
		}

		return fmt.Errorf("failed to get current record for update: %w", err)
	}

	// Apply the updater (either default or custom)
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ContractMetadata{
		Address:       key.Address(),
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPDATE)
}

func (s *catalogContractMetadataStore) Delete(_ context.Context, _ datastore.ContractMetadataKey) error {
	return errors.New("delete operation not supported for catalog contract metadata store")
}

// editRecord is a helper method that handles Add, Upsert, and Update operations
func (s *catalogContractMetadataStore) editRecord(record datastore.ContractMetadata, semantics pb.EditSemantics) error {
	stream, err := s.client.DataAccess()
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Get the current version for this record
	key := record.Key()
	version := s.getVersion(key)

	// Send edit request
	editReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ContractMetadataEditRequest{
			ContractMetadataEditRequest: &pb.ContractMetadataEditRequest{
				Record:    s.contractMetadataToProto(record, version),
				Semantics: semantics,
			},
		},
	}

	if sendErr := stream.Send(editReq); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive response
	resp, recvErr := stream.Recv()
	if recvErr != nil {
		if errors.Is(recvErr, io.EOF) {
			return errors.New("unexpected end of stream")
		}

		return fmt.Errorf("failed to receive response: %w", recvErr)
	}

	// Check for errors in the edit response
	if statusErr := parseResponseStatus(resp.Status); statusErr != nil {
		st, err := parseStatusError(statusErr)
		if err != nil {
			return err
		}

		switch st.Code() { //nolint:exhaustive // We don't need to handle all codes here
		case codes.NotFound:
			return fmt.Errorf("%w: %s", datastore.ErrContractMetadataNotFound, statusErr.Error())
		case codes.Aborted:
			return fmt.Errorf("%w: %s", datastore.ErrContractMetadataStale, statusErr.Error())
		default:
			return fmt.Errorf("edit request failed: %w", statusErr)
		}
	}

	editResp := resp.GetContractMetadataEditResponse()
	if editResp == nil {
		return errors.New("unexpected response type")
	}

	// Update the version cache - increment the version after successful edit
	newVersion := s.getVersion(key) + 1
	s.setVersion(key, newVersion)

	return nil
}
