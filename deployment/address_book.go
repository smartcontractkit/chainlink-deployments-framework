package deployment

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"golang.org/x/exp/maps"
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
type AddressBook interface {
	Save(chainSelector uint64, address string, tv TypeAndVersion) error
	Addresses() (map[uint64]map[string]TypeAndVersion, error)
	AddressesForChain(chain uint64) (map[string]TypeAndVersion, error)
	// Allows for merging address books (e.g. new deployments with existing ones)
	Merge(other AddressBook) error
	Remove(ab AddressBook) error
}

type AddressesByChain map[uint64]map[string]TypeAndVersion

// OrderedChain represents a chain with its addresses in a deterministic order
type OrderedChain struct {
	ChainSelector uint64                `json:"chainSelector"`
	Addresses     []OrderedAddressEntry `json:"addresses"`
}

// OrderedAddressEntry represents an address with its type and version
type OrderedAddressEntry struct {
	Address        string         `json:"address"`
	TypeAndVersion TypeAndVersion `json:"typeAndVersion"`
}

// MarshalJSON implements custom JSON marshaling for AddressesByChain
// to ensure deterministic ordering: chain selectors numerically, addresses alphabetically (case-insensitive)
func (abc AddressesByChain) MarshalJSON() ([]byte, error) {
	// Get all chain selectors and sort them numerically
	chainSelectors := make([]uint64, 0, len(abc))
	for chainSelector := range abc {
		chainSelectors = append(chainSelectors, chainSelector)
	}
	sort.Slice(chainSelectors, func(i, j int) bool {
		return chainSelectors[i] < chainSelectors[j]
	})

	// Build ordered result
	result := make([]OrderedChain, 0, len(chainSelectors))

	for _, chainSelector := range chainSelectors {
		chainAddresses := abc[chainSelector]

		// Get all addresses and sort them alphabetically
		addresses := make([]string, 0, len(chainAddresses))
		for address := range chainAddresses {
			addresses = append(addresses, address)
		}
		sort.Slice(addresses, func(i, j int) bool {
			return strings.ToLower(addresses[i]) < strings.ToLower(addresses[j]) // Using ToLower to ensure case-insensitive sorting
		})

		// Build ordered address entries
		orderedAddresses := make([]OrderedAddressEntry, 0, len(addresses))
		for _, address := range addresses {
			orderedAddresses = append(orderedAddresses, OrderedAddressEntry{
				Address:        address,
				TypeAndVersion: chainAddresses[address],
			})
		}

		result = append(result, OrderedChain{
			ChainSelector: chainSelector,
			Addresses:     orderedAddresses,
		})
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements custom JSON unmarshaling for AddressesByChain
func (abc *AddressesByChain) UnmarshalJSON(data []byte) error {
	var orderedChains []OrderedChain
	if err := json.Unmarshal(data, &orderedChains); err != nil {
		return err
	}

	// Initialize the map if it's nil
	if *abc == nil {
		*abc = make(AddressesByChain)
	}

	// Convert back to map structure
	for _, chain := range orderedChains {
		chainAddresses := make(map[string]TypeAndVersion)
		for _, addrEntry := range chain.Addresses {
			tv := addrEntry.TypeAndVersion
			// Ensure LabelSet is properly initialized if nil
			if tv.Labels == nil {
				tv.Labels = make(LabelSet)
			}
			chainAddresses[addrEntry.Address] = tv
		}
		(*abc)[chain.ChainSelector] = chainAddresses
	}

	return nil
}

type AddressBookMap struct {
	addressesByChain AddressesByChain
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

	if _, exists := m.addressesByChain[chainSelector]; !exists {
		// First time chain add, create map
		m.addressesByChain[chainSelector] = make(map[string]TypeAndVersion)
	}
	if _, exists := m.addressesByChain[chainSelector][address]; exists {
		return fmt.Errorf("address %s already exists for chain %d", address, chainSelector)
	}
	m.addressesByChain[chainSelector][address] = typeAndVersion

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

	// maps are mutable and pass via a pointer
	// creating a copy of the map to prevent concurrency
	// read and changes outside object-bound
	return m.cloneAddresses(m.addressesByChain), nil
}

func (m *AddressBookMap) AddressesForChain(chainSelector uint64) (map[string]TypeAndVersion, error) {
	_, err := chainsel.GetChainIDFromSelector(chainSelector)
	if err != nil {
		return nil, fmt.Errorf("chain selector %d: %w", chainSelector, ErrInvalidChainSelector)
	}

	m.mtx.RLock()
	defer m.mtx.RUnlock()

	if _, exists := m.addressesByChain[chainSelector]; !exists {
		return nil, fmt.Errorf("chain selector %d: %w", chainSelector, ErrChainNotFound)
	}

	// maps are mutable and pass via a pointer
	// creating a copy of the map to prevent concurrency
	// read and changes outside object-bound
	return maps.Clone(m.addressesByChain[chainSelector]), nil
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
		for address := range chainAddresses {
			if _, exists := m.addressesByChain[chainSelector][address]; !exists {
				return errors.New("AddressBookMap does not contain address from the given address book")
			}
		}
	}

	for chainSelector, chainAddresses := range addresses {
		for address := range chainAddresses {
			delete(m.addressesByChain[chainSelector], address)
		}
	}

	return nil
}

// cloneAddresses creates a deep copy of map[uint64]map[string]TypeAndVersion object
func (m *AddressBookMap) cloneAddresses(input map[uint64]map[string]TypeAndVersion) map[uint64]map[string]TypeAndVersion {
	result := make(map[uint64]map[string]TypeAndVersion)
	for chainSelector, chainAddresses := range input {
		result[chainSelector] = maps.Clone(chainAddresses)
	}

	return result
}

// TODO: Maybe could add an environment argument
// which would ensure only mainnet/testnet chain selectors are used
// for further safety?
func NewMemoryAddressBook() *AddressBookMap {
	return &AddressBookMap{
		addressesByChain: make(map[uint64]map[string]TypeAndVersion),
	}
}

func NewMemoryAddressBookFromMap(addressesByChain map[uint64]map[string]TypeAndVersion) *AddressBookMap {
	return &AddressBookMap{
		addressesByChain: addressesByChain,
	}
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

// MarshalJSON implements custom JSON marshaling for AddressBookMap
// to ensure deterministic ordering via the AddressesByChain marshaling
func (m *AddressBookMap) MarshalJSON() ([]byte, error) {
	m.mtx.RLock()
	defer m.mtx.RUnlock()

	// Use the custom marshaling of AddressesByChain
	return json.Marshal(m.addressesByChain)
}

// UnmarshalJSON implements custom JSON unmarshaling for AddressBookMap
func (m *AddressBookMap) UnmarshalJSON(data []byte) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	// Initialize if needed
	if m.addressesByChain == nil {
		m.addressesByChain = make(AddressesByChain)
	}

	// Use the custom unmarshaling of AddressesByChain
	return json.Unmarshal(data, &m.addressesByChain)
}
