package mcmsutils

import (
	"testing"

	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAptosInspectorFactory(t *testing.T) {
	t.Parallel()

	chain := stubAptosChain()
	action := mcmstypes.TimelockActionSchedule

	factory := newAptosInspectorFactory(chain, action)

	assert.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, chain.URL, factory.chain.URL)
	assert.Equal(t, action, factory.action)
}

func TestAptosInspectorFactory_Make(t *testing.T) {
	t.Parallel()

	t.Run("valid actions", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			action   mcmstypes.TimelockAction
			wantRole mcmsaptossdk.TimelockRole
			wantErr  string
		}{
			{
				name:     "schedule action",
				action:   mcmstypes.TimelockActionSchedule,
				wantRole: mcmsaptossdk.TimelockRoleProposer,
			},
			{
				name:     "bypass action",
				action:   mcmstypes.TimelockActionBypass,
				wantRole: mcmsaptossdk.TimelockRoleBypasser,
			},
			{
				name:     "cancel action",
				action:   mcmstypes.TimelockActionCancel,
				wantRole: mcmsaptossdk.TimelockRoleCanceller,
			},
			{
				name:    "invalid action",
				action:  mcmstypes.TimelockAction("invalid"),
				wantErr: "invalid action",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				chain := stubAptosChain()
				factory := newAptosInspectorFactory(chain, tt.action)

				inspector, err := factory.Make()

				if tt.wantErr != "" {
					require.Error(t, err)
					assert.ErrorContains(t, err, tt.wantErr)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, inspector)
				}
			})
		}
	})
}

func TestNewAptosConverterFactory(t *testing.T) {
	t.Parallel()

	factory := newAptosConverterFactory()

	assert.NotNil(t, factory, "Factory should not be nil")
}

func TestAptosConverterFactory_Make(t *testing.T) {
	t.Parallel()

	factory := newAptosConverterFactory()

	converter, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, converter)
}

func TestNewAptosExecutorFactory(t *testing.T) {
	t.Parallel()

	chain := stubAptosChain()
	encoder := &mcmsaptossdk.Encoder{} // Empty encoder for testing

	factory := newAptosExecutorFactory(chain, encoder)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, chain.URL, factory.chain.URL)
	assert.Equal(t, encoder, factory.encoder)
}

func TestAptosExecutorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubAptosChain()
	encoder := &mcmsaptossdk.Encoder{} // Empty encoder for testing

	factory := newAptosExecutorFactory(chain, encoder)

	executor, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestNewAptosTimelockExecutorFactory(t *testing.T) {
	t.Parallel()

	chain := stubAptosChain()

	factory := newAptosTimelockExecutorFactory(chain)

	require.NotNil(t, factory)
	assert.Equal(t, chain.Selector, factory.chain.Selector)
	assert.Equal(t, chain.URL, factory.chain.URL)
}

func TestAptosTimelockExecutorFactory_Make(t *testing.T) {
	t.Parallel()

	chain := stubAptosChain()
	factory := newAptosTimelockExecutorFactory(chain)

	executor, err := factory.Make()
	require.NoError(t, err)
	assert.NotNil(t, executor)
}
