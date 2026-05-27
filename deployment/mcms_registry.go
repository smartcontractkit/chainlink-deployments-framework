package deployment

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

var (
	// ErrInvalidMCMSTimelockProposalInput indicates MCMSTimelockProposalInput failed validation.
	ErrInvalidMCMSTimelockProposalInput = errors.New("invalid MCMS timelock proposal input")
	// ErrDuplicateMCMSReader indicates a reader is already registered for a chain family.
	ErrDuplicateMCMSReader = errors.New("MCMS reader already registered for chain family")
	// ErrEmptyChainFamily indicates Register was called with an empty chain family.
	ErrEmptyChainFamily = errors.New("chain family must not be empty")
	// ErrNilMCMSReader indicates Register was called with a nil reader.
	ErrNilMCMSReader = errors.New("MCMS reader must not be nil")
)

// MCMSTimelockProposalInput is an input for creating an MCMS timelock proposal.
type MCMSTimelockProposalInput struct {
	// OverridePreviousRoot indicates whether to override the root of the MCMS contract.
	OverridePreviousRoot bool
	// ValidUntil is a Unix timestamp indicating when the proposal expires.
	// Root can't be set or executed after this time.
	ValidUntil uint32
	// TimelockDelay is the amount of time each operation in the proposal must wait before it can be executed.
	TimelockDelay mcmstypes.Duration
	// TimelockAction is the action to perform on the timelock contract (schedule, bypass, or cancel).
	TimelockAction mcmstypes.TimelockAction
	// Qualifier is a string used to qualify the MCMS + Timelock contract addresses.
	Qualifier string
	// Description is a human-readable description of the proposal.
	Description string
}

// minValidUntil returns the earliest ValidUntil timestamp accepted by Validate.
func minValidUntil(now time.Time) uint32 {
	return uint32(now.Add(10 * time.Minute).UTC().Unix()) //nolint:gosec // Unix timestamp fits uint32 until year 2106
}

// Validate validates the MCMSTimelockProposalInput.
func (c *MCMSTimelockProposalInput) Validate() error {
	return c.validateAt(time.Now())
}

func (c *MCMSTimelockProposalInput) validateAt(now time.Time) error {
	if c.TimelockAction != mcmstypes.TimelockActionSchedule &&
		c.TimelockAction != mcmstypes.TimelockActionBypass &&
		c.TimelockAction != mcmstypes.TimelockActionCancel {
		return fmt.Errorf("%w: invalid timelock action %q", ErrInvalidMCMSTimelockProposalInput, c.TimelockAction)
	}
	if c.TimelockDelay.Duration < 0 {
		return fmt.Errorf("%w: timelock delay must not be negative", ErrInvalidMCMSTimelockProposalInput)
	}
	if c.TimelockAction == mcmstypes.TimelockActionSchedule && c.TimelockDelay.Duration <= 0 {
		return fmt.Errorf("%w: timelock delay must be positive for schedule action", ErrInvalidMCMSTimelockProposalInput)
	}
	if c.ValidUntil == 0 {
		return fmt.Errorf("%w: valid until must be set", ErrInvalidMCMSTimelockProposalInput)
	}
	// valid until must be far enough in the future to account for clock drift and proposal creation delay.
	if c.ValidUntil < minValidUntil(now) {
		return fmt.Errorf("%w: valid until must be at least 10 minutes in the future", ErrInvalidMCMSTimelockProposalInput)
	}

	return nil
}

// MCMSReader is an interface for reading MCMS state from a chain type.
type MCMSReader interface {
	// GetChainMetadata returns the chain metadata for a given MCMS input.
	// Each chain family defines its own implementation of this method.
	GetChainMetadata(e Environment, chainSelector uint64, input MCMSTimelockProposalInput) (mcmstypes.ChainMetadata, error)
	// GetTimelockRef returns the timelock contract address reference for a given MCMS input.
	GetTimelockRef(e Environment, chainSelector uint64, input MCMSTimelockProposalInput) (datastore.AddressRef, error)
	// GetMCMSRef returns the MCMS contract address reference for a given MCMS input.
	GetMCMSRef(e Environment, chainSelector uint64, input MCMSTimelockProposalInput) (datastore.AddressRef, error)
}

// MCMSReaderRegistry maintains a registry of MCMS readers.
type MCMSReaderRegistry struct {
	mu sync.RWMutex
	m  map[string]MCMSReader
}

func newMCMSReaderRegistry() *MCMSReaderRegistry {
	return &MCMSReaderRegistry{
		m: make(map[string]MCMSReader),
	}
}

var (
	singletonRegistry *MCMSReaderRegistry
	once              sync.Once
)

// GetMCMSReaderRegistry returns the global singleton MCMS reader registry.
// The first call creates the registry; subsequent calls return the same pointer.
// This is the recommended way to get the registry, as it ensures a single instance is created and shared.
func GetMCMSReaderRegistry() *MCMSReaderRegistry {
	once.Do(func() {
		singletonRegistry = newMCMSReaderRegistry()
	})

	return singletonRegistry
}

// Register registers an MCMSReader for a specific chain family.
func (r *MCMSReaderRegistry) Register(chainFamily string, reader MCMSReader) error {
	chainFamily = strings.TrimSpace(chainFamily)
	if chainFamily == "" {
		return ErrEmptyChainFamily
	}
	if reader == nil {
		return ErrNilMCMSReader
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.m == nil {
		r.m = make(map[string]MCMSReader)
	}
	if _, exists := r.m[chainFamily]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateMCMSReader, chainFamily)
	}
	r.m[chainFamily] = reader

	return nil
}

// Get retrieves an MCMSReader for a specific chain family.
func (r *MCMSReaderRegistry) Get(chainFamily string) (MCMSReader, bool) {
	chainFamily = strings.TrimSpace(chainFamily)
	r.mu.RLock()
	defer r.mu.RUnlock()
	reader, ok := r.m[chainFamily]

	return reader, ok
}
