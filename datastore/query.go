package datastore

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAddressRefQueryEmpty     = errors.New("address ref query is empty")
	ErrAddressRefQueryNoMatch   = errors.New("no address ref matched query")
	ErrAddressRefQueryAmbiguous = errors.New("multiple address refs matched query")
	ErrAddressRefFormatFailed   = errors.New("failed to format address ref")
)

// FormatFn formats a resolved AddressRef into type T (for example a chain-native address).
type FormatFn[T any] func(ref AddressRef) (T, error)

// FindUniqueRef queries store for the single AddressRef whose fields match the non-zero
// criteria on ref. The returned value is a clone of the record from the store, not a merge with ref.
//
// Query criteria (only non-zero / non-nil fields on ref are applied):
//   - ChainSelector — omitted when 0 (cannot filter by chain selector 0)
//   - Type, Version, Qualifier, Address
//
// Labels on ref are ignored; they are not part of the query.
//
// Partial criteria are allowed but may match multiple records and return
// [ErrAddressRefQueryAmbiguous]. Set ChainSelector when the chain is known, and include
// Version when multiple versions of the same contract type exist on a chain.
func FindUniqueRef(store AddressRefStore, ref AddressRef) (AddressRef, error) {
	return findUniqueRef(store, ref)
}

// FindAndFormatRef resolves a unique AddressRef via [FindUniqueRef] and formats it with format.
// Use [FindUniqueRef] when T is AddressRef. format must be non-nil.
func FindAndFormatRef[T any](store AddressRefStore, ref AddressRef, format FormatFn[T]) (T, error) {
	var empty T
	if format == nil {
		return empty, fmt.Errorf("%w: format function is required", ErrAddressRefFormatFailed)
	}
	refFromStore, err := findUniqueRef(store, ref)
	if err != nil {
		return empty, err
	}
	formattedRef, err := format(refFromStore)
	if err != nil {
		return empty, fmt.Errorf("%w: ref %s: %w", ErrAddressRefFormatFailed, sprintQueryRef(refFromStore), err)
	}

	return formattedRef, nil
}

func findUniqueRef(store AddressRefStore, ref AddressRef) (AddressRef, error) {
	if isAddressRefQueryEmpty(ref) {
		return AddressRef{}, fmt.Errorf("%w", ErrAddressRefQueryEmpty)
	}

	filterFns := make([]FilterFunc[AddressRefKey, AddressRef], 0, 5)
	if ref.ChainSelector != 0 {
		filterFns = append(filterFns, AddressRefByChainSelector(ref.ChainSelector))
	}
	if ref.Type != "" {
		filterFns = append(filterFns, AddressRefByType(ref.Type))
	}
	if ref.Version != nil {
		filterFns = append(filterFns, AddressRefByVersion(ref.Version))
	}
	if ref.Qualifier != "" {
		filterFns = append(filterFns, AddressRefByQualifier(ref.Qualifier))
	}
	if ref.Address != "" {
		filterFns = append(filterFns, AddressRefByAddress(ref.Address))
	}

	refs := store.Filter(filterFns...)
	switch len(refs) {
	case 1:
		return refs[0].Clone(), nil
	case 0:
		return AddressRef{}, fmt.Errorf("%w: expected exactly 1 ref matching query %s, found 0", ErrAddressRefQueryNoMatch, sprintQueryRef(ref))
	default:
		return AddressRef{}, fmt.Errorf("%w: expected exactly 1 ref matching query %s, found %d", ErrAddressRefQueryAmbiguous, sprintQueryRef(ref), len(refs))
	}
}

func isAddressRefQueryEmpty(ref AddressRef) bool {
	return ref.ChainSelector == 0 &&
		ref.Type == "" &&
		ref.Version == nil &&
		ref.Qualifier == "" &&
		ref.Address == ""
}

func sprintQueryRef(ref AddressRef) string {
	parts := make([]string, 0, 5)
	if ref.ChainSelector != 0 {
		parts = append(parts, fmt.Sprintf("ChainSelector: %d", ref.ChainSelector))
	}
	if ref.Type != "" {
		parts = append(parts, fmt.Sprintf("Type: %s", ref.Type))
	}
	if ref.Version != nil {
		parts = append(parts, fmt.Sprintf("Version: %s", ref.Version))
	}
	if ref.Qualifier != "" {
		parts = append(parts, "Qualifier: "+ref.Qualifier)
	}
	if ref.Address != "" {
		parts = append(parts, "Address: "+ref.Address)
	}

	return "{" + strings.Join(parts, ", ") + "}"
}
