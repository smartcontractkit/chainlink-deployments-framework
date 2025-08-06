package remote

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	datastore2 "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote/internal/protos"
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

func (s *catalogDataStore) BeginTransaction() error {
	request := &datastore2.DataAccessRequest{
		Operation: &datastore2.DataAccessRequest_BeginTransactionRequest{
			BeginTransactionRequest: &datastore2.BeginTransactionRequest{},
		},
	}
	_, err := ThrowAndCatch(s, request)
	return err
}

func (s *catalogDataStore) CommitTransaction() error {
	request := &datastore2.DataAccessRequest{
		Operation: &datastore2.DataAccessRequest_CommitTransactionRequest{
			CommitTransactionRequest: &datastore2.CommitTransactionRequest{},
		},
	}
	_, err := ThrowAndCatch(s, request)
	return err
}

func (s *catalogDataStore) RollbackTransaction() error {
	request := &datastore2.DataAccessRequest{
		Operation: &datastore2.DataAccessRequest_BeginTransactionRequest{
			BeginTransactionRequest: &datastore2.BeginTransactionRequest{},
		},
	}
	_, err := ThrowAndCatch(s, request)
	return err

}

func (s *catalogDataStore) WithTransaction(ctx context.Context, fn datastore.TransactionLogic) error {
	err := s.BeginTransaction()
	if err != nil {
		return err
	}
	err = fn(ctx)
	if err != nil {
		err2 := s.RollbackTransaction()
		if err2 != nil {
			return fmt.Errorf("failed to rollback transaction: %s: %s", err, err2)
		}
		return err
	} else {
		err := s.CommitTransaction()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %s", err)
		}
		return nil
	}
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
