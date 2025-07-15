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
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/internal/protos"
)

type CatalogEnvMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      pb.DeploymentsDatastoreClient
}

type CatalogEnvMetadataStore struct {
	domain      string
	environment string
	client      pb.DeploymentsDatastoreClient
	// versionCache tracks the current version of the record for optimistic concurrency control
	// Environment metadata is a single record per domain/environment, so we only need one version
	cachedVersion int32
}

var _ datastore.EnvMetadataStore = &CatalogEnvMetadataStore{}

var _ datastore.MutableEnvMetadataStore = &CatalogEnvMetadataStore{}

// NewCatalogEnvMetadataStore creates a new CatalogEnvMetadataStore instance.
func NewCatalogEnvMetadataStore(cfg CatalogEnvMetadataStoreConfig) *CatalogEnvMetadataStore {
	return &CatalogEnvMetadataStore{
		domain:        cfg.Domain,
		environment:   cfg.Environment,
		client:        cfg.Client,
		cachedVersion: 0, // Default version for new records
	}
}

// getVersion retrieves the cached version for the record, defaulting to 0 for new records
func (s *CatalogEnvMetadataStore) getVersion() int32 {
	return s.cachedVersion
}

// setVersion updates the cached version for the record
func (s *CatalogEnvMetadataStore) setVersion(version int32) {
	s.cachedVersion = version
}

// keyToFilter converts domain/environment to an EnvironmentMetadataKeyFilter for gRPC requests
func (s *CatalogEnvMetadataStore) keyToFilter() *pb.EnvironmentMetadataKeyFilter {
	return &pb.EnvironmentMetadataKeyFilter{
		Domain:      wrapperspb.String(s.domain),
		Environment: wrapperspb.String(s.environment),
	}
}

// protoToEnvMetadata converts a protobuf EnvironmentMetadata to a datastore EnvMetadata
func (s *CatalogEnvMetadataStore) protoToEnvMetadata(protoRecord *pb.EnvironmentMetadata) (datastore.EnvMetadata, error) {
	var metadata any
	if protoRecord.Metadata != "" {
		if err := json.Unmarshal([]byte(protoRecord.Metadata), &metadata); err != nil {
			return datastore.EnvMetadata{}, fmt.Errorf("failed to unmarshal metadata JSON: %w", err)
		}
	}

	return datastore.EnvMetadata{
		Metadata: metadata,
	}, nil
}

// envMetadataToProto converts a datastore EnvMetadata to a protobuf EnvironmentMetadata
func (s *CatalogEnvMetadataStore) envMetadataToProto(record datastore.EnvMetadata, version int32) *pb.EnvironmentMetadata {
	var metadataJSON string
	if record.Metadata != nil {
		if metadataBytes, err := json.Marshal(record.Metadata); err == nil {
			metadataJSON = string(metadataBytes)
		}
	} else {
		// Use null JSON literal for nil metadata instead of empty string
		metadataJSON = "null"
	}

	return &pb.EnvironmentMetadata{
		Domain:      s.domain,
		Environment: s.environment,
		Metadata:    metadataJSON,
		RowVersion:  version,
	}
}

func (s *CatalogEnvMetadataStore) Get() (datastore.EnvMetadata, error) {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return datastore.EnvMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer func() {
		_ = stream.CloseSend()
	}()

	// Send find request
	findReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_EnvironmentMetadataFindRequest{
			EnvironmentMetadataFindRequest: &pb.EnvironmentMetadataFindRequest{
				KeyFilter: s.keyToFilter(),
			},
		},
	}

	if sendErr := stream.Send(findReq); sendErr != nil {
		return datastore.EnvMetadata{}, fmt.Errorf("failed to send find request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		return datastore.EnvMetadata{}, fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the response
	if resp.Status != nil && !resp.Status.Succeeded {
		if strings.Contains(resp.Status.GetError(), "No records found") {
			return datastore.EnvMetadata{}, datastore.ErrEnvMetadataNotSet
		}

		return datastore.EnvMetadata{}, fmt.Errorf("request failed: %s", resp.Status.Error)
	}

	findResp := resp.GetEnvironmentMetadataFindResponse()
	if findResp == nil {
		return datastore.EnvMetadata{}, errors.New("unexpected response type")
	}

	if len(findResp.References) == 0 {
		return datastore.EnvMetadata{}, datastore.ErrEnvMetadataNotSet
	}

	protoRecord := findResp.References[0]
	record, err := s.protoToEnvMetadata(protoRecord)
	if err != nil {
		return datastore.EnvMetadata{}, err
	}

	// Cache the version for future operations
	s.setVersion(protoRecord.RowVersion)

	return record, nil
}

func (s *CatalogEnvMetadataStore) Set(record datastore.EnvMetadata) error {
	stream, err := s.client.DataAccess(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}
	defer func() {
		_ = stream.CloseSend()
	}()

	// First, try to get the current record to sync our version cache
	// This is important for environment metadata since it's a singleton
	_, getErr := s.Get()
	if getErr != nil && !errors.Is(getErr, datastore.ErrEnvMetadataNotSet) {
		return fmt.Errorf("failed to get current record for version sync: %w", getErr)
	}

	// Get the current version for this record
	version := s.getVersion()

	// Send edit request with UPSERT semantics (since Set should always work)
	editReq := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_EnvironmentMetadataEditRequest{
			EnvironmentMetadataEditRequest: &pb.EnvironmentMetadataEditRequest{
				Record:    s.envMetadataToProto(record, version),
				Semantics: pb.EditSemantics_SEMANTICS_UPSERT,
			},
		},
	}

	if sendErr := stream.Send(editReq); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("unexpected end of stream")
		}

		return fmt.Errorf("failed to receive response: %w", err)
	}

	// Check for errors in the edit response
	if resp.Status != nil && !resp.Status.Succeeded {
		errorMsg := resp.Status.GetError()

		// Check for version conflicts
		if strings.Contains(errorMsg, "incorrect row version") {
			return datastore.ErrEnvMetadataStale
		}

		return fmt.Errorf("edit request failed: %s", resp.Status.Error)
	}

	editResp := resp.GetEnvironmentMetadataEditResponse()
	if editResp == nil {
		return errors.New("unexpected response type")
	}

	// Update the version cache - increment the version after successful edit
	newVersion := s.getVersion() + 1
	s.setVersion(newVersion)

	return nil
}
