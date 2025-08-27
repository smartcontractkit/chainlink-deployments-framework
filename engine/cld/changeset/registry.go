package changeset

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// RegistryProvider defines an interface for initializing and managing the changeset registry
// for a domain environment. It provides methods to initialize the registry, archive changesets,
// and retrieve the initialized ChangesetsRegistry.
type RegistryProvider interface {
	// Init initializes the changeset registry by adding changesets specific to the domain
	// environment using the `Add` method on the ChangesetsRegistry.
	Init() error

	// Archive archives a changeset in the registry. This is intended for changesets that have
	// already been applied and are retained only for historical purposes.
	Archive()

	// Registry retrieves the initialized ChangesetsRegistry.
	Registry() *ChangesetsRegistry
}

var _ RegistryProvider = (*BaseRegistryProvider)(nil)

// BaseRegistryProvider is a base implementation of a RegistryProvider that provides a
// ChangesetsRegistry.
type BaseRegistryProvider struct {
	registry *ChangesetsRegistry
}

// NewBaseRegistryProvider is an implementation of a RegistryProvider that provides a struct that
// can be embedded in domain-specific registry providers.
func NewBaseRegistryProvider() *BaseRegistryProvider {
	return &BaseRegistryProvider{
		registry: NewChangesetsRegistry(),
	}
}

// Registry returns the ChangesetsRegistry.
func (p *BaseRegistryProvider) Registry() *ChangesetsRegistry {
	return p.registry
}

// Init is an empty implementation of adding changesets to the registry.
//
// This should be overridden by the domain-specific registry provider.
func (p *BaseRegistryProvider) Init() error {
	return nil
}

// Archive is an empty implementation of archiving changesets in the registry.
//
// This should be overridden by the domain-specific registry provider.
func (p *BaseRegistryProvider) Archive() {}

type registryEntry struct {
	// changeset is the changeset that is registered.
	changeset ChangeSet

	// gitSHA is the git SHA of the buried changeset. This only applies to changesets that are
	// buried.
	gitSHA *string

	// options contains the configuration options for this changeset
	options changesetConfig
}

// newRegistryEntry creates a new registry entry for a changeset.
func newRegistryEntry(c ChangeSet, opts changesetConfig) registryEntry {
	return registryEntry{changeset: c, options: opts}
}

// newArchivedRegistryEntry creates a new registry entry for an archived changeset.
func newArchivedRegistryEntry(gitSHA string) registryEntry {
	return registryEntry{gitSHA: &gitSHA}
}

// IsArchived returns true if the changeset is archived.
func (e registryEntry) IsArchived() bool {
	return e.gitSHA != nil
}

// ChangesetsRegistry is a registry of changesets that can be applied to a domain environment.
type ChangesetsRegistry struct {
	mu sync.Mutex

	// entries is a map of changeset keys to registry entries.
	entries map[string]registryEntry

	// keyHistory is a list of all changeset keys in the order they were added to the registry.
	keyHistory []string

	// validate enables or disables changeset key validation.
	validate bool
}

// NewChangesetsRegistry creates a new ChangesetsRegistry.
func NewChangesetsRegistry() *ChangesetsRegistry {
	return &ChangesetsRegistry{
		entries:    make(map[string]registryEntry),
		keyHistory: []string{},
		validate:   true,
	}
}

// SetValidate sets the validate flag for the registry. If set to true, changeset keys will be validated.
func (r *ChangesetsRegistry) SetValidate(validate bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.validate = validate
}

// Apply applies a changeset.
func (r *ChangesetsRegistry) Apply(
	key string, e cldf.Environment,
) (cldf.ChangesetOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.entries[key]
	if !ok {
		return cldf.ChangesetOutput{}, fmt.Errorf("changeset '%s' not found", key)
	}

	if entry.IsArchived() {
		return cldf.ChangesetOutput{}, fmt.Errorf("changeset '%s' is archived at SHA '%s'", key, *entry.gitSHA)
	}

	return entry.changeset.Apply(e)
}

// GetChangesetOptions retrieves the configuration options for a changeset.
func (r *ChangesetsRegistry) GetChangesetOptions(key string) (changesetConfig, error) {
	entry, ok := r.entries[key]
	if !ok {
		return changesetConfig{}, fmt.Errorf("changeset '%s' not found", key)
	}

	return entry.options, nil
}

// GetConfigurations retrieves the configurations for a changeset.
func (r *ChangesetsRegistry) GetConfigurations(key string) (Configurations, error) {
	entry, ok := r.entries[key]
	if !ok {
		return Configurations{}, fmt.Errorf("changeset '%s' not found", key)
	}

	return entry.changeset.Configurations()
}

// ChangesetOption defines an option for configuring a changeset
type ChangesetOption func(*changesetConfig)

// changesetConfig holds configuration options for a changeset
type changesetConfig struct {
	chainsToLoad      []uint64
	withoutJD         bool
	operationRegistry *operations.OperationRegistry
}

// OnlyLoadChainsFor will configure the environment to load only the specified chains.
// By default, if option is not specified, all chains are loaded.
// This is useful for changesets that are only applicable to a subset of chains.
func OnlyLoadChainsFor(chainSelectors ...uint64) ChangesetOption {
	return func(o *changesetConfig) {
		o.chainsToLoad = chainSelectors
	}
}

// WithoutJD will configure the environment to not load Job Distributor.
// By default, if option is not specified, Job Distributor is loaded.
// This is useful for changesets that do not require Job Distributor to be loaded.
func WithoutJD() ChangesetOption {
	return func(o *changesetConfig) {
		o.withoutJD = true
	}
}

// WithOperationRegistry will configure the changeset to use the specified operation registry.
func WithOperationRegistry(registry *operations.OperationRegistry) ChangesetOption {
	return func(o *changesetConfig) {
		o.operationRegistry = registry
	}
}

// Add adds a changeset to the registry.
// Options can be passed to configure the changeset.
// - OnlyLoadChainsFor: will configure the environment to load only the specified chains.
// - WithoutJD: will configure the environment to not load Job Distributor.
// - WithOperationRegistry: will configure the changeset to use the specified operation registry.
func (r *ChangesetsRegistry) Add(key string, cs ChangeSet, opts ...ChangesetOption) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate the key format and index
	if r.validate {
		if err := r.validateKey(key); err != nil {
			panic(fmt.Errorf("invalid changeset key '%s': %w", key, err))
		}
	}

	// Process the options
	options := changesetConfig{}
	for _, opt := range opts {
		opt(&options)
	}

	r.entries[key] = newRegistryEntry(cs, options)
	r.keyHistory = append(r.keyHistory, key)
}

func (r *ChangesetsRegistry) validateKey(key string) error {
	// Extract the numerical prefix from the new key
	currentIndex, err := extractIndexFromKey(key)
	if err != nil {
		return fmt.Errorf("invalid changeset key format '%s': %w", key, err)
	}

	// If there are existing changesets, validate that the new index is greater than the last one
	if len(r.keyHistory) > 0 {
		lastKey := r.keyHistory[len(r.keyHistory)-1]
		lastIndex, err := extractIndexFromKey(lastKey)
		if err != nil {
			return fmt.Errorf("invalid previous changeset key format '%s': %w", lastKey, err)
		}

		if currentIndex <= lastIndex {
			return fmt.Errorf("changeset index must be monotonically increasing: got %d, previous was %d",
				currentIndex, lastIndex)
		}
	}

	return nil
}

// extractIndexFromKey extracts the numerical index from a changeset key.
// Expected format: "0001_changeset_name" where "0001" is the index.
func extractIndexFromKey(key string) (int, error) {
	parts := strings.Split(key, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("key '%s' does not follow the format 'index_name'", key)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("could not parse index from key '%s': %w", key, err)
	}

	return index, nil
}

// Archive buries a changeset in the registry.
func (r *ChangesetsRegistry) Archive(key string, gitSHA string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[key] = newArchivedRegistryEntry(gitSHA)
	r.keyHistory = append(r.keyHistory, key)
}

// LatestKey returns the most recent changeset key.
func (r *ChangesetsRegistry) LatestKey() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.keyHistory) == 0 {
		return "", errors.New("no changesets found")
	}

	return r.keyHistory[len(r.keyHistory)-1], nil
}

// ListKeys returns a copy of all registered changeset keys.
func (r *ChangesetsRegistry) ListKeys() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.keyHistory)
}
