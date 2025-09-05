package deployment

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"
)

var (
	// ErrMnemonicRequired is returned when the OCR mnemonic is not set
	ErrMnemonicRequired = ocr.ErrMnemonicRequired
)

// OCRSecrets are used to disseminate a shared secret to OCR nodes
// through the blockchain where OCR configuration is stored. Its a low value secret used
// to derive transmission order etc. They are extracted here such that they can common
// across signers when multiple signers are signing the same OCR config.
type OCRSecrets = ocr.OCRSecrets

var XXXGenerateTestOCRSecrets = ocr.XXXGenerateTestOCRSecrets

// SharedSecrets generates shared secrets from the BIP39 mnemonic phrases for the OCR signers
// and proposers.
//
// Lifted from here
// https://github.com/smartcontractkit/offchain-reporting/blob/14a57d70e50474a2104aa413214e464d6bc69e16/lib/offchainreporting/internal/config/shared_secret_test.go#L32
// Historically signers (fixed secret) and proposers (ephemeral secret) were
// combined in this manner. We simply leave that as is.
var GenerateSharedSecrets = ocr.GenerateSharedSecrets
