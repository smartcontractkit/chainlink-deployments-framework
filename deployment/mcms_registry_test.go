package deployment

import (
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	mcms_types "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// testValidUntilUnix is a stable far-future timestamp for tests (year 2057).
const testValidUntilUnix = uint32(2756219818)

const testOpCount = uint64(42)

func validTestMCMSInput() MCMSTimelockProposalInput {
	return MCMSTimelockProposalInput{
		TimelockAction: mcms_types.TimelockActionSchedule,
		ValidUntil:     testValidUntilUnix,
		TimelockDelay:  mcms_types.NewDuration(time.Hour),
		Description:    "proposal",
	}
}

func TestMCMSTimelockProposalInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   MCMSTimelockProposalInput
		wantErr error
		errMsg  string
	}{
		{
			name:  "valid schedule",
			input: validTestMCMSInput(),
		},
		{
			name: "valid bypass with zero delay",
			input: MCMSTimelockProposalInput{
				TimelockAction: mcms_types.TimelockActionBypass,
				ValidUntil:     testValidUntilUnix,
				TimelockDelay:  mcms_types.NewDuration(0),
			},
		},
		{
			name: "valid cancel with zero delay",
			input: MCMSTimelockProposalInput{
				TimelockAction: mcms_types.TimelockActionCancel,
				ValidUntil:     testValidUntilUnix,
				TimelockDelay:  mcms_types.NewDuration(0),
			},
		},
		{
			name: "invalid timelock action",
			input: MCMSTimelockProposalInput{
				TimelockAction: "not-an-action",
				ValidUntil:     testValidUntilUnix,
				TimelockDelay:  mcms_types.NewDuration(time.Hour),
			},
			wantErr: ErrInvalidMCMSTimelockProposalInput,
			errMsg:  "invalid timelock action",
		},
		{
			name: "valid schedule with zero delay",
			input: MCMSTimelockProposalInput{
				TimelockAction: mcms_types.TimelockActionSchedule,
				ValidUntil:     testValidUntilUnix,
				TimelockDelay:  mcms_types.NewDuration(0),
			},
		},
		{
			name: "negative timelock delay",
			input: MCMSTimelockProposalInput{
				TimelockAction: mcms_types.TimelockActionSchedule,
				ValidUntil:     testValidUntilUnix,
				TimelockDelay:  mcms_types.NewDuration(-time.Hour),
			},
			wantErr: ErrInvalidMCMSTimelockProposalInput,
			errMsg:  "timelock delay must not be negative",
		},
		{
			name: "valid until zero",
			input: MCMSTimelockProposalInput{
				TimelockAction: mcms_types.TimelockActionSchedule,
				ValidUntil:     0,
				TimelockDelay:  mcms_types.NewDuration(time.Hour),
			},
			wantErr: ErrInvalidMCMSTimelockProposalInput,
			errMsg:  "valid until must be set",
		},
		{
			name: "valid until in the past",
			input: MCMSTimelockProposalInput{
				TimelockAction: mcms_types.TimelockActionSchedule,
				ValidUntil:     1,
				TimelockDelay:  mcms_types.NewDuration(time.Hour),
			},
			wantErr: ErrInvalidMCMSTimelockProposalInput,
			errMsg:  "valid until must be at least 10 minutes in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.input.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
			require.ErrorContains(t, err, tt.errMsg)
		})
	}
}

func TestMCMSTimelockProposalInput_Validate_validUntilBoundary(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	input := validTestMCMSInput()
	input.ValidUntil = minValidUntil(fixedNow)
	require.NoError(t, input.validateAt(fixedNow))

	input.ValidUntil = minValidUntil(fixedNow) - 1
	err := input.validateAt(fixedNow)
	require.ErrorIs(t, err, ErrInvalidMCMSTimelockProposalInput)
	require.ErrorContains(t, err, "valid until must be at least 10 minutes in the future")
}

func TestMCMSReaderRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	registry := newMCMSReaderRegistry()
	reader := &mockReader{}
	require.NoError(t, registry.Register("evm", reader))

	got, ok := registry.Get("evm")
	require.True(t, ok)
	require.Same(t, reader, got)

	got, ok = registry.Get("  evm  ")
	require.True(t, ok)
	require.Same(t, reader, got)

	_, ok = registry.Get("solana")
	require.False(t, ok)
}

func TestMCMSReaderRegistry_RegisterDuplicateReturnsError(t *testing.T) {
	t.Parallel()

	registry := newMCMSReaderRegistry()
	require.NoError(t, registry.Register("evm", &mockReader{}))

	err := registry.Register("evm", &mockReader{})
	require.ErrorIs(t, err, ErrDuplicateMCMSReader)
	require.ErrorContains(t, err, "evm")

	got, ok := registry.Get("evm")
	require.True(t, ok)
	require.NotNil(t, got)
}

func TestMCMSReaderRegistry_RegisterRejectsEmptyChainFamily(t *testing.T) {
	t.Parallel()

	registry := newMCMSReaderRegistry()
	err := registry.Register("", &mockReader{})
	require.ErrorIs(t, err, ErrEmptyChainFamily)
}

func TestMCMSReaderRegistry_RegisterRejectsNilReader(t *testing.T) {
	t.Parallel()

	registry := newMCMSReaderRegistry()
	err := registry.Register("evm", nil)
	require.ErrorIs(t, err, ErrNilMCMSReader)
}

func TestMCMSReaderRegistry_GetWithNilMap(t *testing.T) {
	t.Parallel()

	registry := &MCMSReaderRegistry{}
	_, ok := registry.Get("evm")
	require.False(t, ok)
}

func TestMCMSReaderRegistry_RegisterOnZeroValue(t *testing.T) {
	t.Parallel()

	registry := &MCMSReaderRegistry{}
	require.NoError(t, registry.Register("evm", &mockReader{}))

	got, ok := registry.Get("evm")
	require.True(t, ok)
	require.NotNil(t, got)
}

func TestGetMCMSReaderRegistry_ReturnsSingleton(t *testing.T) {
	t.Parallel()

	first := GetMCMSReaderRegistry()
	second := GetMCMSReaderRegistry()
	require.Same(t, first, second)
}

type mockReader struct{}

func (m *mockReader) GetChainMetadata(_ Environment, _ uint64, _ MCMSTimelockProposalInput) (mcms_types.ChainMetadata, error) {
	return mcms_types.ChainMetadata{StartingOpCount: testOpCount}, nil
}

func (m *mockReader) GetTimelockRef(_ Environment, selector uint64, _ MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{
		ChainSelector: selector,
		Address:       "0x01",
		Type:          datastore.ContractType("Timelock"),
		Version:       semver.MustParse("1.0.0"),
	}, nil
}

func (m *mockReader) GetMCMSRef(_ Environment, selector uint64, _ MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{
		ChainSelector: selector,
		Address:       "0x02",
		Type:          datastore.ContractType("MCM"),
		Version:       semver.MustParse("1.0.0"),
	}, nil
}
