package ocr

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOCRSecrets_IsEmpty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		secrets  OCRSecrets
		expected bool
	}{
		{
			name: "both fields empty - should be empty",
			secrets: OCRSecrets{
				SharedSecret: [16]byte{},
				EphemeralSk:  [32]byte{},
			},
			expected: true,
		},
		{
			name: "shared secret empty, ephemeral sk not empty - should be empty",
			secrets: OCRSecrets{
				SharedSecret: [16]byte{},
				EphemeralSk:  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			},
			expected: true,
		},
		{
			name: "shared secret not empty, ephemeral sk empty - should be empty",
			secrets: OCRSecrets{
				SharedSecret: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				EphemeralSk:  [32]byte{},
			},
			expected: true,
		},
		{
			name: "both fields not empty - should not be empty",
			secrets: OCRSecrets{
				SharedSecret: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				EphemeralSk:  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.secrets.IsEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestXXXGenerateTestOCRSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "generates deterministic secrets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test that the function generates non-empty secrets
			secrets1 := XXXGenerateTestOCRSecrets()
			assert.False(t, secrets1.IsEmpty(), "generated secrets should not be empty")

			// Test deterministic behavior - should generate same secrets each time
			secrets2 := XXXGenerateTestOCRSecrets()
			assert.Equal(t, secrets1, secrets2, "function should be deterministic")

			// Test that generated values match expected keccak256 hashes
			expectedSharedSecret := crypto.Keccak256([]byte("shared"))[:16]
			expectedEphemeralSk := crypto.Keccak256([]byte("ephemeral"))

			assert.Equal(t, expectedSharedSecret, secrets1.SharedSecret[:], "shared secret should match keccak256 of 'shared'")
			assert.Equal(t, expectedEphemeralSk, secrets1.EphemeralSk[:], "ephemeral sk should match keccak256 of 'ephemeral'")

			// Verify that the secrets are properly sized
			assert.Len(t, secrets1.SharedSecret, 16, "shared secret should be 16 bytes")
			assert.Len(t, secrets1.EphemeralSk, 32, "ephemeral sk should be 32 bytes")
		})
	}
}

func TestGenerateSharedSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		xSigners    string
		xProposers  string
		expectError bool
		errorType   error
	}{
		// Basic valid cases
		{
			name:        "valid signers and proposers",
			xSigners:    "test-signers",
			xProposers:  "test-proposers",
			expectError: false,
		},
		// Error cases
		{
			name:        "empty signers",
			xSigners:    "",
			xProposers:  "test-proposers",
			expectError: true,
			errorType:   ErrMnemonicRequired,
		},
		{
			name:        "empty proposers",
			xSigners:    "test-signers",
			xProposers:  "",
			expectError: true,
			errorType:   ErrMnemonicRequired,
		},
		{
			name:        "both empty",
			xSigners:    "",
			xProposers:  "",
			expectError: true,
			errorType:   ErrMnemonicRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			secrets, err := GenerateSharedSecrets(tt.xSigners, tt.xProposers)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
				assert.True(t, secrets.IsEmpty(), "secrets should be empty when error occurs")
			} else {
				require.NoError(t, err)
				assert.False(t, secrets.IsEmpty(), "secrets should not be empty when successful")

				// Verify that the secrets are properly sized
				assert.Len(t, secrets.SharedSecret, 16, "shared secret should be 16 bytes")
				assert.Len(t, secrets.EphemeralSk, 32, "ephemeral sk should be 32 bytes")

				// Verify deterministic behavior - same inputs should produce same outputs
				secrets2, err2 := GenerateSharedSecrets(tt.xSigners, tt.xProposers)
				require.NoError(t, err2)
				assert.Equal(t, secrets, secrets2, "function should be deterministic")

				// Verify that the generated secrets match the expected algorithm
				xSignersHash := crypto.Keccak256([]byte(tt.xSigners))
				xProposersHash := crypto.Keccak256([]byte(tt.xProposers))
				xSignersHashxProposersHashZero := append(append(append([]byte{}, xSignersHash...), xProposersHash...), 0)
				xSignersHashxProposersHashOne := append(append(append([]byte{}, xSignersHash...), xProposersHash...), 1)

				expectedSharedSecret := crypto.Keccak256(xSignersHashxProposersHashZero)[:16]
				expectedEphemeralSk := crypto.Keccak256(xSignersHashxProposersHashOne)

				assert.Equal(t, expectedSharedSecret, secrets.SharedSecret[:], "shared secret should match expected hash")
				assert.Equal(t, expectedEphemeralSk, secrets.EphemeralSk[:], "ephemeral sk should match expected hash")
			}
		})
	}
}
