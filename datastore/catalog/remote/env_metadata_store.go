package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	datastore2 "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote/internal/protos"
)

type catalogEnvMetadataStoreConfig struct {
	Domain      string
	Environment string
	Client      *CatalogClient
}

// Ensure catalogEnvMetadataStore implements the V2 interface
var _ datastore.MutableUnaryStoreV2[datastore.EnvMetadata] = &catalogEnvMetadataStore{}

type catalogEnvMetadataStore struct {
	domain      string
	environment string
	client      *CatalogClient
	// versionCache tracks the current version of the record for optimistic concurrency control
	// Environment metadata is a single record per domain/environment, so we only need one version
	mu            sync.RWMutex
	cachedVersion int32
}

// newCatalogEnvMetadataStore creates a new CatalogEnvMetadataStore instance.
func newCatalogEnvMetadataStore(cfg catalogEnvMetadataStoreConfig) *catalogEnvMetadataStore {
	return &catalogEnvMetadataStore{
		domain:        cfg.Domain,
		environment:   cfg.Environment,
		client:        cfg.Client,
		cachedVersion: 0, // Default version for new records
	}
}

// getVersion retrieves the cached version for the record, defaulting to 0 for new records
func (s *catalogEnvMetadataStore) getVersion() int32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cachedVersion
}

// setVersion updates the cached version for the record
func (s *catalogEnvMetadataStore) setVersion(version int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedVersion = version
}

// keyToFilter converts domain/environment to an EnvironmentMetadataKeyFilter for gRPC requests
func (s *catalogEnvMetadataStore) keyToFilter() *datastore2.EnvironmentMetadataKeyFilter {
	return &datastore2.EnvironmentMetadataKeyFilter{
		Domain:      wrapperspb.String(s.domain),
		Environment: wrapperspb.String(s.environment),
	}
}

// protoToEnvMetadata converts a protobuf EnvironmentMetadata to a datastore EnvMetadata
func (s *catalogEnvMetadataStore) protoToEnvMetadata(protoRecord *datastore2.EnvironmentMetadata) (datastore.EnvMetadata, error) {
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
func (s *catalogEnvMetadataStore) envMetadataToProto(record datastore.EnvMetadata, version int32) *datastore2.EnvironmentMetadata {
	var metadataJSON string
	if record.Metadata != nil {
		if metadataBytes, err := json.Marshal(record.Metadata); err == nil {
			metadataJSON = string(metadataBytes)
		}
	} else {
		// Use null JSON literal for nil metadata instead of empty string
		metadataJSON = "null"
	}

	return &datastore2.EnvironmentMetadata{
		Domain:      s.domain,
		Environment: s.environment,
		Metadata:    metadataJSON,
		RowVersion:  version,
	}
}
func (s *catalogEnvMetadataStore) Get() (datastore.EnvMetadata, error) {
	return s.get(true)
}
func (s *catalogEnvMetadataStore) GetIgnoringTransactions() (datastore.EnvMetadata, error) {
	return s.get(true)
}

func (s *catalogEnvMetadataStore) get(ignoreTransaction bool) (datastore.EnvMetadata, error) {
	stream, err := s.client.DataAccess()
	if err != nil {
		return datastore.EnvMetadata{}, fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	// Send find request
	findReq := &datastore2.DataAccessRequest{
		Operation: &datastore2.DataAccessRequest_EnvironmentMetadataFindRequest{
			EnvironmentMetadataFindRequest: &datastore2.EnvironmentMetadataFindRequest{
				KeyFilter:         s.keyToFilter(),
				IgnoreTransaction: ignoreTransaction,
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

func (s *catalogEnvMetadataStore) Set(metadata any, opts ...datastore.UpdateOption) error {
	// Build options with defaults
	options := &datastore.UpdateOptions{
		Updater: datastore.IdentityUpdaterF, // default updater
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	// Get current record for merging
	currentRecord, err := s.Get()
	if err != nil {
		if errors.Is(err, datastore.ErrEnvMetadataNotSet) {
			// Record doesn't exist, just insert the new record directly
			record := datastore.EnvMetadata{
				Metadata: metadata,
			}

			return s.editRecord(record)
		}

		return fmt.Errorf("failed to get current record for version sync: %w", err)
	}

	// Record exists, apply the updater to merge with existing metadata
	finalMetadata, updateErr := options.Updater(currentRecord.Metadata, metadata)
	if updateErr != nil {
		return fmt.Errorf("failed to apply metadata updater: %w", updateErr)
	}

	// Create record with final metadata
	record := datastore.EnvMetadata{
		Metadata: finalMetadata,
	}

	return s.editRecord(record)
}

// editRecord is a helper method that handles the edit operation
func (s *catalogEnvMetadataStore) editRecord(record datastore.EnvMetadata) error {
	// Get the current version for this record
	version := s.getVersion()
	// Create the protobuf record
	protoRecord := s.envMetadataToProto(record, version)

	// Send edit request with UPSERT semantics (since Set should always work)
	stream, err := s.client.DataAccess()
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	editReq := &datastore2.DataAccessRequest{
		Operation: &datastore2.DataAccessRequest_EnvironmentMetadataEditRequest{
			EnvironmentMetadataEditRequest: &datastore2.EnvironmentMetadataEditRequest{
				Record:    protoRecord,
				Semantics: datastore2.EditSemantics_SEMANTICS_UPSERT,
			},
		},
	}

	if sendErr := stream.Send(editReq); sendErr != nil {
		return fmt.Errorf("failed to send edit request: %w", sendErr)
	}

	// Receive response
	resp, err := stream.Recv()
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("request canceled or deadline exceeded: %w", err)
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
