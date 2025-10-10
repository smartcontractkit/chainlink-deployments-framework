package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// memoryStorage provides the core in-memory storage backend for all stores
type memoryStorage struct {
	mu               sync.RWMutex
	addressRefs      map[string]datastore.AddressRef
	chainMetadata    map[uint64]datastore.ChainMetadata
	contractMetadata map[string]datastore.ContractMetadata
	envMetadata      *datastore.EnvMetadata

	// Transaction support
	txMu         sync.RWMutex
	transactions map[*transaction]*transactionData
}

// transactionData holds the changes made during a transaction
type transactionData struct {
	addressRefs      map[string]*datastore.AddressRef       // nil means delete
	chainMetadata    map[uint64]*datastore.ChainMetadata    // nil means delete
	contractMetadata map[string]*datastore.ContractMetadata // nil means delete
	envMetadata      *datastore.EnvMetadata
	envMetadataSet   bool // track if env metadata was explicitly set
}

// transaction represents an active transaction
type transaction struct {
	storage *memoryStorage
	data    *transactionData
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{
		addressRefs:      make(map[string]datastore.AddressRef),
		chainMetadata:    make(map[uint64]datastore.ChainMetadata),
		contractMetadata: make(map[string]datastore.ContractMetadata),
		transactions:     make(map[*transaction]*transactionData),
	}
}

// Transaction management
func (s *memoryStorage) beginTransaction() *transaction {
	s.txMu.Lock()
	defer s.txMu.Unlock()

	tx := &transaction{
		storage: s,
		data: &transactionData{
			addressRefs:      make(map[string]*datastore.AddressRef),
			chainMetadata:    make(map[uint64]*datastore.ChainMetadata),
			contractMetadata: make(map[string]*datastore.ContractMetadata),
		},
	}

	s.transactions[tx] = tx.data

	return tx
}

func (s *memoryStorage) commitTransaction(tx *transaction) error {
	s.txMu.Lock()
	defer s.txMu.Unlock()

	data, exists := s.transactions[tx]
	if !exists {
		return errors.New("transaction not found")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply address ref changes
	for key, ref := range data.addressRefs {
		if ref == nil {
			delete(s.addressRefs, key)
		} else {
			s.addressRefs[key] = *ref
		}
	}

	// Apply chain metadata changes
	for key, metadata := range data.chainMetadata {
		if metadata == nil {
			delete(s.chainMetadata, key)
		} else {
			s.chainMetadata[key] = *metadata
		}
	}

	// Apply contract metadata changes
	for key, metadata := range data.contractMetadata {
		if metadata == nil {
			delete(s.contractMetadata, key)
		} else {
			s.contractMetadata[key] = *metadata
		}
	}

	// Apply env metadata changes
	if data.envMetadataSet {
		s.envMetadata = data.envMetadata
	}

	delete(s.transactions, tx)

	return nil
}

func (s *memoryStorage) rollbackTransaction(tx *transaction) error {
	s.txMu.Lock()
	defer s.txMu.Unlock()

	if _, exists := s.transactions[tx]; !exists {
		return errors.New("transaction not found")
	}

	delete(s.transactions, tx)

	return nil
}

// Helper functions to create composite keys
func addressRefKey(chainSelector uint64, contractType, version, qualifier string) string {
	return fmt.Sprintf("%d:%s:%s:%s", chainSelector, contractType, version, qualifier)
}

func contractMetadataKey(chainSelector uint64, address string) string {
	return fmt.Sprintf("%d:%s", chainSelector, address)
}

// Address reference operations
func (s *memoryStorage) getAddressRef(ctx context.Context, key string, ignoreTransactions bool) (datastore.AddressRef, error) {
	// Check transaction first if not ignoring transactions
	if !ignoreTransactions {
		if tx := s.getTransactionFromContext(ctx); tx != nil {
			if ref, exists := tx.data.addressRefs[key]; exists {
				if ref == nil {
					return datastore.AddressRef{}, datastore.ErrAddressRefNotFound
				}

				return *ref, nil
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	ref, exists := s.addressRefs[key]
	if !exists {
		return datastore.AddressRef{}, datastore.ErrAddressRefNotFound
	}

	return ref, nil
}

func (s *memoryStorage) getAllAddressRefs(ctx context.Context) ([]datastore.AddressRef, error) {
	s.mu.RLock()
	baseRefs := make(map[string]datastore.AddressRef)
	for k, v := range s.addressRefs {
		baseRefs[k] = v
	}
	s.mu.RUnlock()

	// Apply transaction changes if in transaction
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		for key, ref := range tx.data.addressRefs {
			if ref == nil {
				delete(baseRefs, key)
			} else {
				baseRefs[key] = *ref
			}
		}
	}

	refs := make([]datastore.AddressRef, 0, len(baseRefs))
	for _, ref := range baseRefs {
		refs = append(refs, ref)
	}

	return refs, nil
}

func (s *memoryStorage) setAddressRef(ctx context.Context, key string, ref datastore.AddressRef) error {
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		tx.data.addressRefs[key] = &ref
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.addressRefs[key] = ref

	return nil
}

// Chain metadata operations
func (s *memoryStorage) getChainMetadata(ctx context.Context, chainSelector uint64, ignoreTransactions bool) (datastore.ChainMetadata, error) {
	// Check transaction first if not ignoring transactions
	if !ignoreTransactions {
		if tx := s.getTransactionFromContext(ctx); tx != nil {
			if metadata, exists := tx.data.chainMetadata[chainSelector]; exists {
				if metadata == nil {
					return datastore.ChainMetadata{}, datastore.ErrChainMetadataNotFound
				}

				return *metadata, nil
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.chainMetadata[chainSelector]
	if !exists {
		return datastore.ChainMetadata{}, datastore.ErrChainMetadataNotFound
	}

	return metadata, nil
}

func (s *memoryStorage) getAllChainMetadata(ctx context.Context) ([]datastore.ChainMetadata, error) {
	s.mu.RLock()
	baseMetadata := make(map[uint64]datastore.ChainMetadata)
	for k, v := range s.chainMetadata {
		baseMetadata[k] = v
	}
	s.mu.RUnlock()

	// Apply transaction changes if in transaction
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		for key, metadata := range tx.data.chainMetadata {
			if metadata == nil {
				delete(baseMetadata, key)
			} else {
				baseMetadata[key] = *metadata
			}
		}
	}

	metadataList := make([]datastore.ChainMetadata, 0, len(baseMetadata))
	for _, metadata := range baseMetadata {
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func (s *memoryStorage) setChainMetadata(ctx context.Context, chainSelector uint64, metadata datastore.ChainMetadata) error {
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		tx.data.chainMetadata[chainSelector] = &metadata
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.chainMetadata[chainSelector] = metadata

	return nil
}

// Contract metadata operations
func (s *memoryStorage) getContractMetadata(ctx context.Context, key string, ignoreTransactions bool) (datastore.ContractMetadata, error) {
	// Check transaction first if not ignoring transactions
	if !ignoreTransactions {
		if tx := s.getTransactionFromContext(ctx); tx != nil {
			if metadata, exists := tx.data.contractMetadata[key]; exists {
				if metadata == nil {
					return datastore.ContractMetadata{}, datastore.ErrContractMetadataNotFound
				}

				return *metadata, nil
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	metadata, exists := s.contractMetadata[key]
	if !exists {
		return datastore.ContractMetadata{}, datastore.ErrContractMetadataNotFound
	}

	return metadata, nil
}

func (s *memoryStorage) getAllContractMetadata(ctx context.Context) ([]datastore.ContractMetadata, error) {
	s.mu.RLock()
	baseMetadata := make(map[string]datastore.ContractMetadata)
	for k, v := range s.contractMetadata {
		baseMetadata[k] = v
	}
	s.mu.RUnlock()

	// Apply transaction changes if in transaction
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		for key, metadata := range tx.data.contractMetadata {
			if metadata == nil {
				delete(baseMetadata, key)
			} else {
				baseMetadata[key] = *metadata
			}
		}
	}

	metadataList := make([]datastore.ContractMetadata, 0, len(baseMetadata))
	for _, metadata := range baseMetadata {
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func (s *memoryStorage) setContractMetadata(ctx context.Context, key string, metadata datastore.ContractMetadata) error {
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		tx.data.contractMetadata[key] = &metadata
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.contractMetadata[key] = metadata

	return nil
}

// Environment metadata operations
func (s *memoryStorage) getEnvMetadata(ctx context.Context, ignoreTransactions bool) (datastore.EnvMetadata, error) {
	// Check transaction first if not ignoring transactions
	if !ignoreTransactions {
		if tx := s.getTransactionFromContext(ctx); tx != nil {
			if tx.data.envMetadataSet {
				if tx.data.envMetadata == nil {
					return datastore.EnvMetadata{}, datastore.ErrEnvMetadataNotSet
				}

				return *tx.data.envMetadata, nil
			}
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.envMetadata == nil {
		return datastore.EnvMetadata{}, datastore.ErrEnvMetadataNotSet
	}

	return *s.envMetadata, nil
}

func (s *memoryStorage) setEnvMetadata(ctx context.Context, metadata datastore.EnvMetadata) error {
	if tx := s.getTransactionFromContext(ctx); tx != nil {
		tx.data.envMetadata = &metadata
		tx.data.envMetadataSet = true

		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.envMetadata = &metadata

	return nil
}

// Helper to get transaction from context
func (s *memoryStorage) getTransactionFromContext(ctx context.Context) *transaction {
	if tx, ok := ctx.Value(transactionKey{}).(*transaction); ok {
		s.txMu.RLock()
		defer s.txMu.RUnlock()
		if _, exists := s.transactions[tx]; exists {
			return tx
		}
	}

	return nil
}
