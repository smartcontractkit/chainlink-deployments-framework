package deployment

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
)

var (
	ErrInvalidChainSelector = errors.New("invalid chain selector")
	ErrInvalidAddress       = errors.New("invalid address")
	ErrChainNotFound        = errors.New("chain not found")
)

// ContractType is a simple string type for identifying contract types.
type ContractType string

func (ct ContractType) String() string {
	return string(ct)
}

type TypeAndVersion struct {
	Type    ContractType   `json:"Type"`
	Version semver.Version `json:"Version"`
	Labels  LabelSet       `json:"Labels,omitempty"`
}

func (tv TypeAndVersion) String() string {
	if len(tv.Labels) == 0 {
		return fmt.Sprintf("%s %s", tv.Type, tv.Version.String())
	}

	// Use the LabelSet's String method for sorted labels
	sortedLabels := tv.Labels.String()

	return fmt.Sprintf("%s %s %s",
		tv.Type,
		tv.Version.String(),
		sortedLabels,
	)
}

func (tv TypeAndVersion) Equal(other TypeAndVersion) bool {
	// Compare Type
	if tv.Type != other.Type {
		return false
	}
	// Compare Versions
	if !tv.Version.Equal(&other.Version) {
		return false
	}
	// Compare Labels
	return tv.Labels.Equal(other.Labels)
}

func MustTypeAndVersionFromString(s string) TypeAndVersion {
	tv, err := TypeAndVersionFromString(s)
	if err != nil {
		panic(err)
	}

	return tv
}

// Note this will become useful for validation. When we want
// to assert an onchain call to typeAndVersion yields whats expected.
func TypeAndVersionFromString(s string) (TypeAndVersion, error) {
	parts := strings.Fields(s) // Ignores consecutive spaces
	if len(parts) < 2 {
		return TypeAndVersion{}, fmt.Errorf("invalid type and version string: %s", s)
	}
	v, err := semver.NewVersion(parts[1])
	if err != nil {
		return TypeAndVersion{}, err
	}
	labels := make(LabelSet)
	if len(parts) > 2 {
		labels = NewLabelSet(parts[2:]...)
	}

	return TypeAndVersion{
		Type:    ContractType(parts[0]),
		Version: *v,
		Labels:  labels,
	}, nil
}

func NewTypeAndVersion(t ContractType, v semver.Version) TypeAndVersion {
	return TypeAndVersion{
		Type:    t,
		Version: v,
		Labels:  make(LabelSet), // empty set,
	}
}

// AddressBook is a simple interface for storing and retrieving contract addresses across
// chains. It is family agnostic as the keys are chain selectors.
// We store rather than derive typeAndVersion as some contracts do not support it.
// For ethereum addresses are always stored in EIP55 format.
// All methods return results in sorted order for consistent behavior.
type AddressBook interface {
	Save(chainSelector uint64, address string, tv TypeAndVersion) error
	Addresses() (map[uint64]map[string]TypeAndVersion, error)
	AddressesForChain(chain uint64) (map[string]TypeAndVersion, error)
	// Allows for merging address books (e.g. new deployments with existing ones)
	Merge(other AddressBook) error
	Remove(ab AddressBook) error
}

type AddressesByChain map[uint64]map[string]TypeAndVersion

type AddressBookMap struct {
	// Use TreeMap to maintain sorted order automatically
	addressesByChain *treemap.Map // map[uint64]*treemap.Map[string]TypeAndVersion
	mtx              sync.RWMutex
}

// Save will save an address for a given chain selector. It will error if there is a conflicting existing address.
func (m *AddressBookMap) save(chainSelector uint64, address string, typeAndVersion TypeAndVersion) error {
	family, err := chainsel.GetSelectorFamily(chainSelector)
	if err != nil {
		return fmt.Errorf("chain selector %d: %w", chainSelector, ErrInvalidChainSelector)
	}
	if family == chainsel.FamilyEVM {
		if address == "" || address == common.HexToAddress("0x0").Hex() {
			return fmt.Errorf("address cannot be empty: %w", ErrInvalidAddress)
		}
		if common.IsHexAddress(address) {
			// IMPORTANT: WE ALWAYS STANDARDIZE ETHEREUM ADDRESS STRINGS TO EIP55
			address = common.HexToAddress(address).Hex()
		} else {
			return fmt.Errorf("address %s is not a valid Ethereum address, only Ethereum addresses supported for EVM chains: %w", address, ErrInvalidAddress)
		}
	}

	// TODO NONEVM-960: Add validation for non-EVM chain addresses

	if typeAndVersion.Type == "" {
		return errors.New("type cannot be empty")
	}

	// Get or create the TreeMap for this chain
	chainAddresses, exists := m.addressesByChain.Get(chainSelector)
	if !exists {
		// First time chain add, create TreeMap with string comparator for sorted addresses
		chainAddresses = treemap.NewWithStringComparator()
		m.addressesByChain.Put(chainSelector, chainAddresses)
	}

	chainMap := chainAddresses.(*treemap.Map)
	if _, exists := chainMap.Get(address); exists {
		return fmt.Errorf("address %s already exists for chain %d", address, chainSelector)
	}
	chainMap.Put(address, typeAndVersion)

	return nil
}

// Save will save an address for a given chain selector. It will error if there is a conflicting existing address.
// thread safety version of the save method
func (m *AddressBookMap) Save(chainSelector uint64, address string, typeAndVersion TypeAndVersion) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	return m.save(chainSelector, address, typeAndVersion)
}

func (m *AddressBookMap) Addresses() (map[uint64]map[string]TypeAndVersion, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	result := make(map[uint64]map[string]TypeAndVersion)

	// Iterate through chains in sorted order (TreeMap automatically sorts by key)
	it := m.addressesByChain.Iterator()
	for it.Next() {
		chainSelector := it.Key().(uint64)
		chainMap := it.Value().(*treemap.Map)

		// Convert TreeMap to regular map for return value
		addresses := make(map[string]TypeAndVersion)
		chainIt := chainMap.Iterator()
		for chainIt.Next() {
			address := chainIt.Key().(string)
			tv := chainIt.Value().(TypeAndVersion)
			addresses[address] = tv
		}

		result[chainSelector] = addresses
	}

	return result, nil
}

func (m *AddressBookMap) AddressesForChain(chainSelector uint64) (map[string]TypeAndVersion, error) {
	_, err := chainsel.GetChainIDFromSelector(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("chain selector %d: %w", chainSelector, ErrInvalidChainSelector)
	}

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	chainAddresses, exists := m.addressesByChain.Get(chainSelector)
	if !exists {
		return nil, fmt.Errorf("chain selector %d: %w", chainSelector, ErrChainNotFound)
	}

	chainMap := chainAddresses.(*treemap.Map)
	result := make(map[string]TypeAndVersion)

	// Iterate through addresses in sorted order
	it := chainMap.Iterator()
	for it.Next() {
		address := it.Key().(string)
		tv := it.Value().(TypeAndVersion)
		result[address] = tv
	}

	return result, nil
}

// Merge will merge the addresses from another address book into this one.
// It will error on any existing addresses.
func (m *AddressBookMap) Merge(ab AddressBook) error {
	addresses, err := ab.Addresses()
	if err != nil {
		return err
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	for chainSelector, chainAddresses := range addresses {
		for address, typeAndVersion := range chainAddresses {
			if err := m.save(chainSelector, address, typeAndVersion); err != nil {
				return err
			}
		}
	}

	return nil
}

// Remove removes the address book addresses specified via the argument from the AddressBookMap.
// Errors if all the addresses in the given address book are not contained in the AddressBookMap.
func (m *AddressBookMap) Remove(ab AddressBook) error {
	addresses, err := ab.Addresses()
	if err != nil {
		return err
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	// State of m.addressesByChain storage must not be changed in case of an error
	// need to do double iteration over the address book. First validation, second actual deletion
	for chainSelector, chainAddresses := range addresses {
		chainMap, exists := m.addressesByChain.Get(chainSelector)
		if !exists {
			return errors.New("AddressBookMap does not contain chain selector from the given address book")
		}

		treeMap := chainMap.(*treemap.Map)
		for address := range chainAddresses {
			if _, exists := treeMap.Get(address); !exists {
				return errors.New("AddressBookMap does not contain address from the given address book")
			}
		}
	}

	for chainSelector, chainAddresses := range addresses {
		chainMap, _ := m.addressesByChain.Get(chainSelector)
		treeMap := chainMap.(*treemap.Map)
		for address := range chainAddresses {
			treeMap.Remove(address)
		}
	}

	return nil
}

// TODO: Maybe could add an environment argument
// which would ensure only mainnet/testnet chain selectors are used
// for further safety?
func NewMemoryAddressBook() *AddressBookMap {
	return &AddressBookMap{
		// Use TreeMap with uint64 comparator to keep chains sorted
		addressesByChain: treemap.NewWith(utils.UInt64Comparator),
	}
}

func NewMemoryAddressBookFromMap(addressesByChain map[uint64]map[string]TypeAndVersion) *AddressBookMap {
	ab := &AddressBookMap{
		addressesByChain: treemap.NewWith(utils.UInt64Comparator),
	}

	// Convert the input map to TreeMaps
	for chainSelector, addresses := range addressesByChain {
		chainMap := treemap.NewWithStringComparator()
		for address, tv := range addresses {
			chainMap.Put(address, tv)
		}
		ab.addressesByChain.Put(chainSelector, chainMap)
	}

	return ab
}

// SearchAddressBook search an address book for a given chain and contract type and return the first matching address.
func SearchAddressBook(ab AddressBook, chain uint64, typ ContractType) (string, error) {
	addrs, err := ab.AddressesForChain(chain)
	if err != nil {
		return "", err
	}

	for addr, tv := range addrs {
		if tv.Type == typ {
			return addr, nil
		}
	}

	return "", errors.New("not found")
}

func AddressBookContains(ab AddressBook, chain uint64, addrToFind string) (bool, error) {
	addrs, err := ab.AddressesForChain(chain)
	if err != nil {
		return false, err
	}

	for addr := range addrs {
		if addr == addrToFind {
			return true, nil
		}
	}

	return false, nil
}

type typeVersionKey struct {
	Type    ContractType
	Version string
	Labels  string // store labels in a canonical form (comma-joined sorted list)
}

func tvKey(tv TypeAndVersion) typeVersionKey {
	sortedLabels := tv.Labels.String()
	return typeVersionKey{
		Type:    tv.Type,
		Version: tv.Version.String(),
		Labels:  sortedLabels,
	}
}

// EnsureDeduped ensures that each contract in the bundle only appears once
// in the address map.  It returns an error if there are more than one instance of a contract.
// Returns true if every value in the bundle is found once, false otherwise.
func EnsureDeduped(addrs map[string]TypeAndVersion, bundle []TypeAndVersion) (bool, error) {
	var (
		grouped = toTypeAndVersionMap(addrs)
		found   = make([]TypeAndVersion, 0)
	)
	for _, btv := range bundle {
		key := tvKey(btv)
		matched, ok := grouped[key]
		if ok {
			found = append(found, btv)
		}
		if len(matched) > 1 {
			return false, fmt.Errorf("found more than one instance of contract %s v%s (labels=%s)",
				key.Type, key.Version, key.Labels)
		}
	}

	// Indicate if each TypeAndVersion in the bundle is found at least once
	return len(found) == len(bundle), nil
}

// toTypeAndVersionMap groups contract addresses by unique TypeAndVersion.
func toTypeAndVersionMap(addrs map[string]TypeAndVersion) map[typeVersionKey][]string {
	tvkMap := make(map[typeVersionKey][]string)
	for k, v := range addrs {
		tvkMap[tvKey(v)] = append(tvkMap[tvKey(v)], k)
	}

	return tvkMap
}

// AddLabel adds a string to the LabelSet in the TypeAndVersion.
func (tv *TypeAndVersion) AddLabel(label string) {
	if tv.Labels == nil {
		tv.Labels = make(LabelSet)
	}
	tv.Labels.Add(label)
}
