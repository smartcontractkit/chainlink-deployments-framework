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
	mu               sync.RWMutex // protects all fields below
	addressRefs      map[string]datastore.AddressRef
	chainMetadata    map[uint64]datastore.ChainMetadata
	contractMetadata map[string]datastore.ContractMetadata
	envMetadata      *datastore.EnvMetadata
	transactions     map[*transaction]*transactionData
}

// transactionData holds the changes made during a transaction
type transactionData struct {
	addressRefs      map[string]datastore.AddressRef
	chainMetadata    map[uint64]datastore.ChainMetadata
	contractMetadata map[string]datastore.ContractMetadata
	envMetadata      *datastore.EnvMetadata
	envMetadataSet   bool // track if env metadata was explicitly set
}

// transaction represents an active transaction
type transaction struct {
	data *transactionData
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
	s.mu.Lock()
	defer s.mu.Unlock()

	tx := &transaction{
		data: &transactionData{
			addressRefs:      make(map[string]datastore.AddressRef),
			chainMetadata:    make(map[uint64]datastore.ChainMetadata),
			contractMetadata: make(map[string]datastore.ContractMetadata),
		},
	}

	s.transactions[tx] = tx.data

	return tx
}

func (s *memoryStorage) commitTransaction(tx *transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, exists := s.transactions[tx]
	if !exists {
		return errors.New("transaction not found")
	}

	// Apply address ref changes
	for key, ref := range data.addressRefs {
		s.addressRefs[key] = ref
	}

	// Apply chain metadata changes
	for key, metadata := range data.chainMetadata {
		s.chainMetadata[key] = metadata
	}

	// Apply contract metadata changes
	for key, metadata := range data.contractMetadata {
		s.contractMetadata[key] = metadata
	}

	// Apply env metadata changes
	if data.envMetadataSet {
		s.envMetadata = data.envMetadata
	}

	delete(s.transactions, tx)

	return nil
}

func (s *memoryStorage) rollbackTransaction(tx *transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
				return ref, nil
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
	defer s.mu.RUnlock()

	baseRefs := make(map[string]datastore.AddressRef)
	for k, v := range s.addressRefs {
		baseRefs[k] = v
	}

	// Apply transaction changes if in transaction
	if tx := s.getTransactionFromContextUnsafe(ctx); tx != nil {
		for key, ref := range tx.data.addressRefs {
			baseRefs[key] = ref
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
		tx.data.addressRefs[key] = ref
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
				return metadata, nil
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
	defer s.mu.RUnlock()

	baseMetadata := make(map[uint64]datastore.ChainMetadata)
	for k, v := range s.chainMetadata {
		baseMetadata[k] = v
	}

	// Apply transaction changes if in transaction
	if tx := s.getTransactionFromContextUnsafe(ctx); tx != nil {
		for key, metadata := range tx.data.chainMetadata {
			baseMetadata[key] = metadata
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
		tx.data.chainMetadata[chainSelector] = metadata
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
				return metadata, nil
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
	defer s.mu.RUnlock()

	baseMetadata := make(map[string]datastore.ContractMetadata)
	for k, v := range s.contractMetadata {
		baseMetadata[k] = v
	}

	// Apply transaction changes if in transaction
	if tx := s.getTransactionFromContextUnsafe(ctx); tx != nil {
		for key, metadata := range tx.data.contractMetadata {
			baseMetadata[key] = metadata
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
		tx.data.contractMetadata[key] = metadata
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getTransactionFromContextUnsafe(ctx)
}

// getTransactionFromContextUnsafe gets transaction from context without acquiring lock.
// Must only be called when caller already holds mu lock (read or write).
func (s *memoryStorage) getTransactionFromContextUnsafe(ctx context.Context) *transaction {
	if tx, ok := ctx.Value(transactionKey{}).(*transaction); ok {
		if _, exists := s.transactions[tx]; exists {
			return tx
		}
	}

	return nil
}
