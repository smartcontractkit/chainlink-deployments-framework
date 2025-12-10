package ocr

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBIP39Mnemonic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entropySize int
		expectError bool
	}{
		{
			name:        "valid entropy 128 bits",
			entropySize: 128,
			expectError: false,
		},
		{
			name:        "valid entropy 160 bits",
			entropySize: 160,
			expectError: false,
		},
		{
			name:        "valid entropy 192 bits",
			entropySize: 192,
			expectError: false,
		},
		{
			name:        "valid entropy 224 bits",
			entropySize: 224,
			expectError: false,
		},
		{
			name:        "valid entropy 256 bits",
			entropySize: 256,
			expectError: false,
		},
		{
			name:        "invalid entropy size - too small",
			entropySize: 127,
			expectError: true,
		},
		{
			name:        "invalid entropy size - too large",
			entropySize: 257,
			expectError: true,
		},
		{
			name:        "invalid entropy size - not multiple of 32",
			entropySize: 129,
			expectError: true,
		},
		{
			name:        "zero entropy size",
			entropySize: 0,
			expectError: true,
		},
		{
			name:        "negative entropy size",
			entropySize: -128,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mnemonic, err := NewBIP39Mnemonic(tt.entropySize)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, mnemonic)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, mnemonic)

				// Validate that the returned mnemonic is a valid BIP39 mnemonic
				assert.True(t, bip39.IsMnemonicValid(mnemonic), "generated mnemonic should be valid")

				// Check word count based on entropy size
				words := strings.Fields(mnemonic)
				expectedWordCount := (tt.entropySize + tt.entropySize/32) / 11
				assert.Len(t, words, expectedWordCount, "word count should match expected for entropy size")

				// Verify mnemonic can be converted back to bytes
				_, err := bip39.MnemonicToByteArray(mnemonic)
				require.NoError(t, err, "should be able to convert mnemonic to byte array")
			}
		})
	}
}

func TestNewBIP39Mnemonic_Uniqueness(t *testing.T) {
	t.Parallel()

	const entropySize = 256
	const numMnemonics = 100

	mnemonics := make(map[string]bool)

	for range numMnemonics {
		mnemonic, err := NewBIP39Mnemonic(entropySize)
		require.NoError(t, err)
		require.NotEmpty(t, mnemonic)

		// Ensure each generated mnemonic is unique
		assert.False(t, mnemonics[mnemonic], "generated mnemonic should be unique")
		mnemonics[mnemonic] = true

		// Validate the mnemonic
		assert.True(t, bip39.IsMnemonicValid(mnemonic))
	}

	assert.Len(t, mnemonics, numMnemonics, "should have generated exactly %d unique mnemonics", numMnemonics)
}

func TestNewBIP39Mnemonic_WordCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		entropySize       int
		expectedWordCount int
	}{
		{128, 12}, // 128 bits -> 12 words
		{160, 15}, // 160 bits -> 15 words
		{192, 18}, // 192 bits -> 18 words
		{224, 21}, // 224 bits -> 21 words
		{256, 24}, // 256 bits -> 24 words
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("entropy_%d_bits", tt.entropySize), func(t *testing.T) {
			t.Parallel()

			mnemonic, err := NewBIP39Mnemonic(tt.entropySize)
			require.NoError(t, err)

			words := strings.Fields(mnemonic)
			assert.Len(t, words, tt.expectedWordCount,
				"entropy size %d should generate %d words", tt.entropySize, tt.expectedWordCount)
		})
	}
}
