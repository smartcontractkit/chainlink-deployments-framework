package remote

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote/internal/protos"
)

type CatalogDataStoreConfig struct {
	Domain      string
	Environment string
	Client      *CatalogClient
}

var _ datastore.CatalogStore = &catalogDataStore{}

type catalogDataStore struct {
	client                *CatalogClient
	addressRefStore       *catalogAddressRefStore
	chainMetadataStore    *catalogChainMetadataStore
	contractMetadataStore *catalogContractMetadataStore
	envMetadataStore      *catalogEnvMetadataStore
}

func (s *catalogDataStore) beginTransaction() error {
	request := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_BeginTransactionRequest{
			BeginTransactionRequest: &pb.BeginTransactionRequest{},
		},
	}
	_, err := ThrowAndCatch(s, request)

	return err
}

func (s *catalogDataStore) commitTransaction() error {
	request := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_CommitTransactionRequest{
			CommitTransactionRequest: &pb.CommitTransactionRequest{},
		},
	}
	_, err := ThrowAndCatch(s, request)

	return err
}

func (s *catalogDataStore) rollbackTransaction() error {
	request := &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_BeginTransactionRequest{
			BeginTransactionRequest: &pb.BeginTransactionRequest{},
		},
	}
	_, err := ThrowAndCatch(s, request)

	return err
}

func (s *catalogDataStore) WithTransaction(ctx context.Context, fn datastore.TransactionLogic) error {
	err := s.beginTransaction()
	if err != nil {
		return err
	}
	err = fn(ctx)
	if err != nil {
		err2 := s.rollbackTransaction()
		if err2 != nil {
			return fmt.Errorf("failed to rollback transaction: %w: %w", err, err2)
		}

		return err
	}
	err = s.commitTransaction()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func NewCatalogDataStore(config CatalogDataStoreConfig) *catalogDataStore {
	return &catalogDataStore{
		client:                config.Client,
		addressRefStore:       newCatalogAddressRefStore(catalogAddressRefStoreConfig(config)),
		chainMetadataStore:    newCatalogChainMetadataStore(catalogChainMetadataStoreConfig(config)),
		contractMetadataStore: newCatalogContractMetadataStore(catalogContractMetadataStoreConfig(config)),
		envMetadataStore:      newCatalogEnvMetadataStore(catalogEnvMetadataStoreConfig(config)),
	}
}

func (s *catalogDataStore) Addresses() datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] {
	return s.addressRefStore
}

func (s *catalogDataStore) ChainMetadata() datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] {
	return s.chainMetadataStore
}

func (s *catalogDataStore) ContractMetadata() datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] {
	return s.contractMetadataStore
}

func (s *catalogDataStore) EnvMetadata() datastore.MutableUnaryStoreV2[datastore.EnvMetadata] {
	return s.envMetadataStore
}
