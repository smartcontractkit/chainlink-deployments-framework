package decoder_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/stretchr/testify/require"
)

// TestDecoderOptions verifies that decoder options work correctly
func TestDecoderOptions(t *testing.T) {
	t.Run("can create decoder with no options", func(t *testing.T) {
		d := decoder.NewLegacyDecoder()
		require.NotNil(t, d)
	})

	t.Run("can configure registry options", func(t *testing.T) {
		d := decoder.NewLegacyDecoder(
			decoder.WithEVMABIMappings(nil),
			decoder.WithSolanaDecoders(nil),
		)
		require.NotNil(t, d)
	})
}
