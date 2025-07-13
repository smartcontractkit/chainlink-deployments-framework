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

type CatalogContractMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      pb.DeploymentsDatastoreClient
}

type CatalogContractMetadataStore struct {
	domain      string
	environment string
	client      pb.DeploymentsDatastoreClient
}

var _ datastore.ContractMetadataStore = &CatalogContractMetadataStore{}
var _ datastore.MutableContractMetadataStore = &CatalogContractMetadataStore{}

func NewCatalogContractMetadataStore(cfg CatalogContractMetadataStoreConfig) *CatalogContractMetadataStore {
	return &CatalogContractMetadataStore{
		domain:      cfg.Domain,
		environment: cfg.Environment,
		client:      cfg.Client,
	}
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
func (s *CatalogContractMetadataStore) contractMetadataToProto(record datastore.ContractMetadata) *pb.ContractMetadata {
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
		RowVersion:    1, // Will be set by the server
	}
}

func (s *CatalogContractMetadataStore) Get(key datastore.ContractMetadataKey) (datastore.ContractMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return datastore.ContractMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer stream.CloseSend()

	// Send find request
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ContractMetadataFindRequest{
			ContractMetadataFindRequest: &pb.ContractMetadataFindRequest{
				KeyFilter: s.keyToFilter(key),
			},
		},
	}

	if err := stream.Send(findReq); err != nil {
		return datastore.ContractMetadata{}, fmt.Errorf("failed to send find request: %w", err)
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
		return datastore.ContractMetadata{}, fmt.Errorf("unexpected response type")
	}

	if len(findResp.References) == 0 {
		return datastore.ContractMetadata{}, datastore.ErrContractMetadataNotFound
	}

	return s.protoToContractMetadata(findResp.References[0])
}

// Fetch returns a copy of all ContractMetadata in the catalog.
func (s *CatalogContractMetadataStore) Fetch() ([]datastore.ContractMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer stream.CloseSend()

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

	findResp := resp.GetContractMetadataFindResponse()
	if findResp == nil {
		return nil, fmt.Errorf("unexpected response type")
	}

	var records []datastore.ContractMetadata
	for _, protoRecord := range findResp.References {
		record, err := s.protoToContractMetadata(protoRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to convert proto to contract metadata: %w", err)
		}
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
	return fmt.Errorf("delete operation not supported for contract metadata store")
}

// editRecord is a helper method that handles Add, Upsert, and Update operations
func (s *CatalogContractMetadataStore) editRecord(record datastore.ContractMetadata, semantics pb.EditSemantics) error {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer stream.CloseSend()

	// Send edit request
	editReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_ContractMetadataEditRequest{
			ContractMetadataEditRequest: &pb.ContractMetadataEditRequest{
				Record:    s.contractMetadataToProto(record),
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
		if strings.Contains(resp.Status.GetError(), "no record found to update for") && semantics == pb.EditSemantics_SEMANTICS_UPDATE {
			return datastore.ErrContractMetadataNotFound
		}

		return fmt.Errorf("edit request failed: %s", resp.Status.Error)
	}

	editResp := resp.GetContractMetadataEditResponse()
	if editResp == nil {
		return fmt.Errorf("unexpected response type")
	}

	return nil
}
