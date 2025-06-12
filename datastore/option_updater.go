package datastore

// RecordUpdater is a generic function type for updating records
type RecordUpdater[R any] func(update R, orig R) (R, error)

// UpdateOptionF is a generic functional option type for record updating
type UpdateOptionF[R any] struct {
	Updater RecordUpdater[R]
}

// WithUpdaterF creates a generic functional option with a custom updater function
func WithUpdaterF[R any](updater RecordUpdater[R]) UpdateOptionF[R] {
	return UpdateOptionF[R]{Updater: updater}
}

// IdentityUpdaterF returns a generic identity updater that just returns the update record unchanged
func IdentityUpdaterF[R any](update R, _ R) (R, error) {
	return update, nil
}

// ApplyUpdater is a helper function that applies the updater from options or uses identity updater
func ApplyUpdater[R any](update R, orig R, opts ...UpdateOptionF[R]) (R, error) {
	updater := IdentityUpdaterF[R]
	if len(opts) > 0 && opts[0].Updater != nil {
		updater = opts[0].Updater
	}
	return updater(update, orig)
}

type ContractMetadataUpdater = RecordUpdater[ContractMetadata]

type ContractMetadataUpdateOption = UpdateOptionF[ContractMetadata]

// WithUpdater creates an option with a custom ContractMetadata updater function
func WithUpdater(updater ContractMetadataUpdater) ContractMetadataUpdateOption {
	return WithUpdaterF(updater)
}

func (s *MemoryContractMetadataStore) UpdateV2(record ContractMetadata, opts ...ContractMetadataUpdateOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx == -1 {
		return ErrContractMetadataNotFound
	}

	updated, err := ApplyUpdater(record, s.Records[idx], opts...)
	if err != nil {
		return err
	}

	s.Records[idx] = updated

	return nil
}

func (s *MemoryContractMetadataStore) UpsertV2(record ContractMetadata, opts ...ContractMetadataUpdateOption) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexOf(record.Key())
	if idx == -1 {
		s.Records = append(s.Records, record)
		return nil
	}

	updated, err := ApplyUpdater(record, s.Records[idx], opts...)
	if err != nil {
		return err
	}

	s.Records[idx] = updated

	return nil
}
