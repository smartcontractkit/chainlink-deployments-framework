package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_KeypairGenKMS_Generate(t *testing.T) {
	t.Parallel()

	// Both cases are rejected before any AWS call is made, so they exercise the
	// wiring without needing credentials.
	tests := []struct {
		name        string
		giveKeyID   string
		giveRegion  string
		wantErrText string
	}{
		{
			name:        "missing key region",
			giveKeyID:   "test-ed25519-key-id",
			giveRegion:  "",
			wantErrText: "KMS key region is required",
		},
		{
			name:        "missing key ID",
			giveKeyID:   "",
			giveRegion:  "us-west-2",
			wantErrText: "KMS key ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gen := KeypairGenKMS(t.Context(), tt.giveKeyID, tt.giveRegion)
			require.NotNil(t, gen)

			signer, err := gen.Generate()
			require.ErrorContains(t, err, tt.wantErrText)
			assert.Nil(t, signer)
		})
	}
}
