package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/protos"
)

type CatalogContractMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      pb.DeploymentsDatastoreClient
}

type CatalogContractMetadataStore struct {
	domain      string
	environment string
	client      pb.DeploymentsDatastoreClient
	// versionCache tracks the current version of each record for optimistic concurrency control
	versionCache map[string]int32
}

var _ datastore.ContractMetadataStore = &CatalogContractMetadataStore{}
var _ datastore.MutableContractMetadataStore = &CatalogContractMetadataStore{}

func NewCatalogContractMetadataStore(cfg CatalogContractMetadataStoreConfig) *CatalogContractMetadataStore {
	return &CatalogContractMetadataStore{
		domain:       cfg.Domain,
		environment:  cfg.Environment,
		client:       cfg.Client,
		versionCache: make(map[string]int32),
	}
}

// getVersion retrieves the cached version for a record, defaulting to 0 for new records
func (s *CatalogContractMetadataStore) getVersion(key datastore.ContractMetadataKey) int32 {
	cacheKey := key.String()
	if version, exists := s.versionCache[cacheKey]; exists {
		return version
	}

	return 0 // Default version for new records
}

// setVersion updates the cached version for a record
func (s *CatalogContractMetadataStore) setVersion(key datastore.ContractMetadataKey, version int32) {
	cacheKey := key.String()
	s.versionCache[cacheKey] = version
}

// keyToFilter converts a ContractMetadataKey to a ContractMetadataKeyFilter for gRPC requests
func (s *CatalogContractMetadataStore) keyToFilter(key datastore.ContractMetadataKey) *pb.ContractMetadataKeyFilter {
	return &pb.ContractMetadataKeyFilter{
		Domain:        wrapperspb.String(s.domain),
		Environment:   wrapperspb.String(s.environment),
		ChainSelector: wrapperspb.UInt64(key.ChainSelector()),
		Address:       wrapperspb.String(key.Address()),
	}
}

// protoToContractMetadata converts a protobuf ContractMetadata to a datastore ContractMetadata
func (s *CatalogContractMetadataStore) protoToContractMetadata(protoRecord *pb.ContractMetadata) (datastore.ContractMetadata, error) {
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
func (s *CatalogContractMetadataStore) contractMetadataToProto(record datastore.ContractMetadata, version int32) *pb.ContractMetadata {
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

func (s *CatalogContractMetadataStore) Get(key datastore.ContractMetadataKey) (datastore.ContractMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return datastore.ContractMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer func() {
		_ = stream.CloseSend()
	}()

	// Send find request
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ContractMetadataFindRequest{
			ContractMetadataFindRequest: &pb.ContractMetadataFindRequest{
				KeyFilter: s.keyToFilter(key),
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
	if resp.Status != nil && !resp.Status.Succeeded {
		if strings.Contains(resp.Status.GetError(), "No records found") {
			return datastore.ContractMetadata{}, datastore.ErrContractMetadataNotFound
		}

		return datastore.ContractMetadata{}, fmt.Errorf("request failed: %s", resp.Status.Error)
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
func (s *CatalogContractMetadataStore) Fetch() ([]datastore.ContractMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer func() {
		_ = stream.CloseSend()
	}()

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
	if resp.Status != nil && !resp.Status.Succeeded {
		return nil, fmt.Errorf("request failed: %s", resp.Status.Error)
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
func (s *CatalogContractMetadataStore) Filter(filters ...datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]) []datastore.ContractMetadata {
	records, err := s.Fetch()
	if err != nil {
		return []datastore.ContractMetadata{}
	}

	for _, filter := range filters {
		records = filter(records)
	}

	return records
}

func (s *CatalogContractMetadataStore) Add(record datastore.ContractMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_INSERT)
}

func (s *CatalogContractMetadataStore) Upsert(record datastore.ContractMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPSERT)
}

func (s *CatalogContractMetadataStore) Update(record datastore.ContractMetadata) error {
	return s.editRecord(record, pb.EditSemantics_SEMANTICS_UPDATE)
}

func (s *CatalogContractMetadataStore) Delete(key datastore.ContractMetadataKey) error {
	return errors.New("delete operation not supported for contract metadata store")
}

// editRecord is a helper method that handles Add, Upsert, and Update operations
func (s *CatalogContractMetadataStore) editRecord(record datastore.ContractMetadata, semantics pb.EditSemantics) error {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer func() {
		_ = stream.CloseSend()
	}()
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
	if resp.Status != nil && !resp.Status.Succeeded {
		errorMsg := resp.Status.GetError()

		// Check for specific error conditions
		if strings.Contains(errorMsg, "no record found to update for") && semantics == pb.EditSemantics_SEMANTICS_UPDATE {
			return datastore.ErrContractMetadataNotFound
		} else if strings.Contains(errorMsg, "incorrect row version") && (semantics == pb.EditSemantics_SEMANTICS_UPDATE || semantics == pb.EditSemantics_SEMANTICS_UPSERT) {
			return datastore.ErrContractMetadataStale
		}

		return fmt.Errorf("edit request failed: %s", resp.Status.Error)
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
