package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

type catalogChainMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      *CatalogClient
}

type catalogChainMetadataStore struct {
	domain      string
	environment string
	client      *CatalogClient
	// versionCache tracks the current version of each record for optimistic concurrency control
	mu           sync.RWMutex
	versionCache map[string]int32
}

var _ datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] = &catalogChainMetadataStore{}

func newCatalogChainMetadataStore(cfg catalogChainMetadataStoreConfig) *catalogChainMetadataStore {
	return &catalogChainMetadataStore{
		domain:       cfg.Domain,
		environment:  cfg.Environment,
		client:       cfg.Client,
		versionCache: make(map[string]int32),
	}
}

// getVersion retrieves the cached version for a record, defaulting to 0 for new records
func (s *catalogChainMetadataStore) getVersion(key datastore.ChainMetadataKey) int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cacheKey := key.String()
	if version, exists := s.versionCache[cacheKey]; exists {
		return version
	}

	return 0 // Default version for new records
}

// setVersion updates the cached version for a record
func (s *catalogChainMetadataStore) setVersion(key datastore.ChainMetadataKey, version int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheKey := key.String()
	s.versionCache[cacheKey] = version
}

// keyToFilter converts a ChainMetadataKey to a ChainMetadataKeyFilter for gRPC requests
func (s *catalogChainMetadataStore) keyToFilter(key datastore.ChainMetadataKey) *pb.ChainMetadataKeyFilter {
	return &pb.ChainMetadataKeyFilter{
		Domain:        wrapperspb.String(s.domain),
		Environment:   wrapperspb.String(s.environment),
		ChainSelector: wrapperspb.UInt64(key.ChainSelector()),
	}
}

// protoToChainMetadata converts a protobuf ChainMetadata to a datastore ChainMetadata
func (s *catalogChainMetadataStore) protoToChainMetadata(protoRecord *pb.ChainMetadata) (datastore.ChainMetadata, error) {
	var metadata any
	if protoRecord.Metadata != "" {
		if err := json.Unmarshal([]byte(protoRecord.Metadata), &metadata); err != nil {
			return datastore.ChainMetadata{}, fmt.Errorf("failed to unmarshal metadata JSON: %w", err)
		}
	}

	return datastore.ChainMetadata{
		ChainSelector: protoRecord.ChainSelector,
		Metadata:      metadata,
	}, nil
}

// chainMetadataToProto converts a datastore ChainMetadata to a protobuf ChainMetadata
func (s *catalogChainMetadataStore) chainMetadataToProto(record datastore.ChainMetadata, version int32) *pb.ChainMetadata {
	var metadataJSON string
	if record.Metadata != nil {
		if metadataBytes, err := json.Marshal(record.Metadata); err == nil {
			metadataJSON = string(metadataBytes)
		}
	}

	return &pb.ChainMetadata{
		Domain:        s.domain,
		Environment:   s.environment,
		ChainSelector: record.ChainSelector,
		Metadata:      metadataJSON,
		RowVersion:    version,
	}
}
func (s *catalogChainMetadataStore) Get(
	_ context.Context,
	key datastore.ChainMetadataKey,
	options ...datastore.GetOption,
) (datastore.ChainMetadata, error) {
	ignoreTransactions := false
	for _, option := range options {
		switch option {
		case datastore.IgnoreTransactionsGetOption:
			ignoreTransactions = true
		}
	}

	return s.get(ignoreTransactions, key)
}

func (s *catalogChainMetadataStore) get(ignoreTransaction bool, key datastore.ChainMetadataKey) (datastore.ChainMetadata, error) {
	stream, err := s.client.DataAccess()
	if err != nil {
		return datastore.ChainMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Send find request
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ChainMetadataFindRequest{
			ChainMetadataFindRequest: &pb.ChainMetadataFindRequest{
				KeyFilter:         s.keyToFilter(key),
				IgnoreTransaction: ignoreTransaction,
			},
		},
	}

	if sendErr := stream.Send(findReq); sendErr != nil {
		return datastore.ChainMetadata{}, fmt.Errorf("failed to send find request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return datastore.ChainMetadata{}, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if statusErr := checkResponseStatus(resp.Status); statusErr != nil {
		st, _ := status.FromError(statusErr)

		switch st.Code() {
		case codes.NotFound:
			return datastore.ChainMetadata{}, fmt.Errorf("%w: %w", datastore.ErrChainMetadataNotFound, statusErr)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded,
			codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition,
			codes.Aborted, codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable,
			codes.DataLoss, codes.Unauthenticated:
			return datastore.ChainMetadata{}, fmt.Errorf("get chain metadata failed: %w", statusErr)
		default:
			return datastore.ChainMetadata{}, fmt.Errorf("get chain metadata failed: %w", statusErr)
		}
	}

	findResp := resp.GetChainMetadataFindResponse()
	if findResp == nil {
		return datastore.ChainMetadata{}, errors.New("unexpected response type")
	}

	if len(findResp.References) == 0 {
		return datastore.ChainMetadata{}, datastore.ErrChainMetadataNotFound
	}

	protoRecord := findResp.References[0]
	record, err := s.protoToChainMetadata(protoRecord)
	if err != nil {
		return datastore.ChainMetadata{}, err
	}

	// Cache the version for future operations
	s.setVersion(record.Key(), protoRecord.RowVersion)

	return record, nil
}

// Fetch returns a copy of all ChainMetadata in the catalog.
func (s *catalogChainMetadataStore) Fetch(_ context.Context) ([]datastore.ChainMetadata, error) {
	stream, err := s.client.DataAccess()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Send find request with domain and environment filter only (fetch all)
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ChainMetadataFindRequest{
			ChainMetadataFindRequest: &pb.ChainMetadataFindRequest{
				KeyFilter: &pb.ChainMetadataKeyFilter{
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
	if err := checkResponseStatus(resp.Status); err != nil {
		return nil, fmt.Errorf("fetch chain metadata failed: %w", err)
	}

	findResp := resp.GetChainMetadataFindResponse()
	if findResp == nil {
		return nil, errors.New("unexpected response type")
	}

	records := make([]datastore.ChainMetadata, 0, len(findResp.References))
	for _, protoRecord := range findResp.References {
		record, err := s.protoToChainMetadata(protoRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proto to chain metadata: %w", err)
		}

		// Cache the version for future operations
		s.setVersion(record.Key(), protoRecord.RowVersion)

		records = append(records, record)
	}

	return records, nil
}

// Filter returns a copy of all ChainMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *catalogChainMetadataStore) Filter(ctx context.Context, filters ...datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]) ([]datastore.ChainMetadata, error) {
	records, err := s.Fetch(ctx)
	if err != nil {
		return []datastore.ChainMetadata{}, fmt.Errorf("failed to fetch records: %w", err)
	}

	for _, filter := range filters {
		records = filter(records)
	}

	return records, nil
}

func (s *catalogChainMetadataStore) Add(_ context.Context, record datastore.ChainMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_INSERT)
}

func (s *catalogChainMetadataStore) Upsert(ctx context.Context, key datastore.ChainMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
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
		if errors.Is(err, datastore.ErrChainMetadataNotFound) {
			record := datastore.ChainMetadata{
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
	record := datastore.ChainMetadata{
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPSERT)
}

func (s *catalogChainMetadataStore) Update(ctx context.Context, key datastore.ChainMetadataKey, metadata any, opts ...datastore.UpdateOption) error {
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
		if errors.Is(err, datastore.ErrChainMetadataNotFound) {
			return datastore.ErrChainMetadataNotFound
		}

		return fmt.Errorf("failed to get current record for update: %w", err)
	}

	// Apply the updater (either default or custom)
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.ChainMetadata{
		ChainSelector: key.ChainSelector(),
		Metadata:      finalMetadata,
	}

	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPDATE)
}

func (s *catalogChainMetadataStore) Delete(_ context.Context, _ datastore.ChainMetadataKey) error {
	return errors.New("delete operation not supported for catalog chain metadata store")
}

// editRecord is a helper method that handles Add, Upsert, and Update operations
func (s *catalogChainMetadataStore) editRecord(record datastore.ChainMetadata, semantics pb.EditSemantics) error {
	stream, err := s.client.DataAccess()
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Get the current version for this record
	key := record.Key()
	version := s.getVersion(key)

	// Send edit request
	editReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ChainMetadataEditRequest{
			ChainMetadataEditRequest: &pb.ChainMetadataEditRequest{
				Record:    s.chainMetadataToProto(record, version),
				Semantics: semantics,
			},
		},
	}

	if sendErr := stream.Send(editReq); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the edit response
	if statusErr := checkResponseStatus(resp.Status); statusErr != nil {
		st, _ := status.FromError(statusErr)

		// Check for specific error conditions
		switch st.Code() {
		case codes.NotFound:
			return fmt.Errorf("%w: %w", datastore.ErrChainMetadataNotFound, statusErr)
		case codes.Aborted:
			return fmt.Errorf("%w: %w", datastore.ErrChainMetadataStale, statusErr)
		case codes.OK, codes.Canceled, codes.Unknown, codes.InvalidArgument, codes.DeadlineExceeded,
			codes.AlreadyExists, codes.PermissionDenied, codes.ResourceExhausted, codes.FailedPrecondition,
			codes.OutOfRange, codes.Unimplemented, codes.Internal, codes.Unavailable,
			codes.DataLoss, codes.Unauthenticated:
			return fmt.Errorf("edit request failed: %w", statusErr)
		default:
			return fmt.Errorf("edit request failed: %w", statusErr)
		}
	}
	editResp := resp.GetChainMetadataEditResponse()
	if editResp == nil {
		return errors.New("unexpected response type")
	}

	// Update the version cache - increment the version after successful edit
	newVersion := s.getVersion(key) + 1
	s.setVersion(key, newVersion)

	return nil
}
