package addrconv

import (
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAddressConverterRegistry(t *testing.T) {
	t.Parallel()

	registry := newAddressConverterRegistry()

	// Test that registry is properly initialized
	require.NotNil(t, registry)
	require.NotNil(t, registry.converters)

	// Test that all expected converters are registered with correct functionality
	expectedFamilies := []string{
		chain_selectors.FamilyEVM,
		chain_selectors.FamilySolana,
		chain_selectors.FamilyAptos,
		chain_selectors.FamilySui,
		chain_selectors.FamilyTon,
		chain_selectors.FamilyTron,
	}

	for _, family := range expectedFamilies {
		t.Run(family, func(t *testing.T) {
			t.Parallel()

			converter, exists := registry.converters[family]
			assert.True(t, exists, "Converter for family %s should be registered", family)
			assert.NotNil(t, converter, "Converter for family %s should not be nil", family)

			// Verify the converter actually supports its registered family
			assert.True(t, converter.Supports(family), "Converter should support its own family %s", family)
		})
	}

	// Test that the correct number of converters are registered
	assert.Len(t, registry.converters, len(expectedFamilies))
}

func TestAddressToBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		family         string
		address        string
		expectedLength int
		shouldError    bool
		errorContains  string
	}{
		{
			name:           "EVM family conversion",
			family:         chain_selectors.FamilyEVM,
			address:        "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8",
			expectedLength: 20,
			shouldError:    false,
		},
		{
			name:           "Solana family conversion",
			family:         chain_selectors.FamilySolana,
			address:        "11111111111111111111111111111112",
			expectedLength: 32,
			shouldError:    false,
		},
		{
			name:           "Aptos family conversion",
			family:         chain_selectors.FamilyAptos,
			address:        "0x1",
			expectedLength: 32,
			shouldError:    false,
		},
		{
			name:          "Unsupported family",
			family:        "unknown",
			address:       "0x742d35Cc6634C0532925a3b8D4c8C1B8c4c8C1B8",
			shouldError:   true,
			errorContains: "no address converter registered for family: unknown",
		},
		{
			name:          "Invalid address",
			family:        chain_selectors.FamilyEVM,
			address:       "invalid",
			shouldError:   true,
			errorContains: "invalid EVM address format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bytes, err := ToBytes(tc.family, tc.address)

			if tc.shouldError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorContains)
				assert.Nil(t, bytes)
			} else {
				require.NoError(t, err)
				require.NotNil(t, bytes)
				assert.Len(t, bytes, tc.expectedLength)
			}
		})
	}
}

func TestRegistrySingleton(t *testing.T) {
	t.Parallel()

	// Test that registry() returns the same instance
	registry1 := registry()
	registry2 := registry()

	assert.Same(t, registry1, registry2, "registry() should return the same singleton instance")
}
