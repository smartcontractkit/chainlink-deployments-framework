package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/protos"
)

type CatalogChainMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      pb.DeploymentsDatastoreClient
}

type CatalogChainMetadataStore struct {
	domain      string
	environment string
	client      pb.DeploymentsDatastoreClient
	// versionCache tracks the current version of each record for optimistic concurrency control
	versionCache map[string]int32
}

var _ datastore.ChainMetadataStore = &CatalogChainMetadataStore{}
var _ datastore.MutableChainMetadataStore = &CatalogChainMetadataStore{}

func NewCatalogChainMetadataStore(cfg CatalogChainMetadataStoreConfig) *CatalogChainMetadataStore {
	return &CatalogChainMetadataStore{
		domain:       cfg.Domain,
		environment:  cfg.Environment,
		client:       cfg.Client,
		versionCache: make(map[string]int32),
	}
}

// getVersion retrieves the cached version for a record, defaulting to 0 for new records
func (s *CatalogChainMetadataStore) getVersion(key datastore.ChainMetadataKey) int32 {
	cacheKey := key.String()
	if version, exists := s.versionCache[cacheKey]; exists {
		return version
	}
	return 0 // Default version for new records
}

// setVersion updates the cached version for a record
func (s *CatalogChainMetadataStore) setVersion(key datastore.ChainMetadataKey, version int32) {
	cacheKey := key.String()
	s.versionCache[cacheKey] = version
}

// keyToFilter converts a ChainMetadataKey to a ChainMetadataKeyFilter for gRPC requests
func (s *CatalogChainMetadataStore) keyToFilter(key datastore.ChainMetadataKey) *pb.ChainMetadataKeyFilter {
	return &pb.ChainMetadataKeyFilter{
		Domain:        wrapperspb.String(s.domain),
		Environment:   wrapperspb.String(s.environment),
		ChainSelector: wrapperspb.UInt64(key.ChainSelector()),
	}
}

// protoToChainMetadata converts a protobuf ChainMetadata to a datastore ChainMetadata
func (s *CatalogChainMetadataStore) protoToChainMetadata(protoRecord *pb.ChainMetadata) (datastore.ChainMetadata, error) {
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
func (s *CatalogChainMetadataStore) chainMetadataToProto(record datastore.ChainMetadata, version int32) *pb.ChainMetadata {
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

func (s *CatalogChainMetadataStore) Get(key datastore.ChainMetadataKey) (datastore.ChainMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return datastore.ChainMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer stream.CloseSend()

	// Send find request
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ChainMetadataFindRequest{
			ChainMetadataFindRequest: &pb.ChainMetadataFindRequest{
				KeyFilter: s.keyToFilter(key),
			},
		},
	}

	if err := stream.Send(findReq); err != nil {
		return datastore.ChainMetadata{}, fmt.Errorf("failed to send find request: %w", err)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return datastore.ChainMetadata{}, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if resp.Status != nil && !resp.Status.Succeeded {
		if strings.Contains(resp.Status.GetError(), "No records found") {
			return datastore.ChainMetadata{}, datastore.ErrChainMetadataNotFound
		}
		return datastore.ChainMetadata{}, fmt.Errorf("request failed: %s", resp.Status.Error)
	}

	findResp := resp.GetChainMetadataFindResponse()
	if findResp == nil {
		return datastore.ChainMetadata{}, fmt.Errorf("unexpected response type")
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
func (s *CatalogChainMetadataStore) Fetch() ([]datastore.ChainMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer stream.CloseSend()

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

	if err := stream.Send(findReq); err != nil {
		return nil, fmt.Errorf("failed to send find request: %w", err)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if resp.Status != nil && !resp.Status.Succeeded {
		return nil, fmt.Errorf("request failed: %s", resp.Status.Error)
	}

	findResp := resp.GetChainMetadataFindResponse()
	if findResp == nil {
		return nil, fmt.Errorf("unexpected response type")
	}

	var records []datastore.ChainMetadata
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
func (s *CatalogChainMetadataStore) Filter(filters ...datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]) []datastore.ChainMetadata {
	records, err := s.Fetch()
	if err != nil {
		return []datastore.ChainMetadata{}
	}

	for _, filter := range filters {
		records = filter(records)
	}
	return records
}

func (s *CatalogChainMetadataStore) Add(record datastore.ChainMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_INSERT)
}

func (s *CatalogChainMetadataStore) Upsert(record datastore.ChainMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPSERT)
}

func (s *CatalogChainMetadataStore) Update(record datastore.ChainMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPDATE)
}

func (s *CatalogChainMetadataStore) Delete(key datastore.ChainMetadataKey) error {
	return fmt.Errorf("delete operation not supported for chain metadata store")
}

// editRecord is a helper method that handles Add, Upsert, and Update operations
func (s *CatalogChainMetadataStore) editRecord(record datastore.ChainMetadata, semantics pb.EditSemantics) error {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer stream.CloseSend()

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

	if err := stream.Send(editReq); err != nil {
		return fmt.Errorf("failed to send edit request: %w", err)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("unexpected end of stream")
		}
		return fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the edit response
	if resp.Status != nil && !resp.Status.Succeeded {
		errorMsg := resp.Status.GetError()

		// Check for specific error conditions
		if strings.Contains(errorMsg, "no record found to update for") && semantics == pb.EditSemantics_SEMANTICS_UPDATE {
			return datastore.ErrChainMetadataNotFound
		} else if strings.Contains(errorMsg, "incorrect row version") && (semantics == pb.EditSemantics_SEMANTICS_UPDATE || semantics == pb.EditSemantics_SEMANTICS_UPSERT) {
			return datastore.ErrChainMetadataStale
		}

		return fmt.Errorf("edit request failed: %s", resp.Status.Error)
	}

	editResp := resp.GetChainMetadataEditResponse()
	if editResp == nil {
		return fmt.Errorf("unexpected response type")
	}

	// Update the version cache - increment the version after successful edit
	newVersion := s.getVersion(key) + 1
	s.setVersion(key, newVersion)

	return nil
}
