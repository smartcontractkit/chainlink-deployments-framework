package deployment

import (
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	// ErrMnemonicRequired is returned when the OCR mnemonic is not set
	ErrMnemonicRequired = errors.New("xsigners or xproposers required")
)

// OCRSecrets are used to disseminate a shared secret to OCR nodes
// through the blockchain where OCR configuration is stored. Its a low value secret used
// to derive transmission order etc. They are extracted here such that they can common
// across signers when multiple signers are signing the same OCR config.
type OCRSecrets struct {
	SharedSecret [16]byte
	EphemeralSk  [32]byte
}

func (s OCRSecrets) IsEmpty() bool {
	return s.SharedSecret == [16]byte{} || s.EphemeralSk == [32]byte{}
}

func XXXGenerateTestOCRSecrets() OCRSecrets {
	var s OCRSecrets
	copy(s.SharedSecret[:], crypto.Keccak256([]byte("shared"))[:16])
	copy(s.EphemeralSk[:], crypto.Keccak256([]byte("ephemeral")))

	return s
}

// SharedSecrets generates shared secrets from the BIP39 mnemonic phrases for the OCR signers
// and proposers.
//
// Lifted from here
// https://github.com/smartcontractkit/offchain-reporting/blob/14a57d70e50474a2104aa413214e464d6bc69e16/lib/offchainreporting/internal/config/shared_secret_test.go#L32
// Historically signers (fixed secret) and proposers (ephemeral secret) were
// combined in this manner. We simply leave that as is.
func GenerateSharedSecrets(xSigners, xProposers string) (OCRSecrets, error) {
	if xSigners == "" || xProposers == "" {
		return OCRSecrets{}, ErrMnemonicRequired
	}

	xSignersHash := crypto.Keccak256([]byte(xSigners))
	xProposersHash := crypto.Keccak256([]byte(xProposers))
	xSignersHashxProposersHashZero := append(append(append([]byte{}, xSignersHash...), xProposersHash...), 0)
	xSignersHashxProposersHashOne := append(append(append([]byte{}, xSignersHash...), xProposersHash...), 1)
	var sharedSecret [16]byte
	copy(sharedSecret[:], crypto.Keccak256(xSignersHashxProposersHashZero))
	var sk [32]byte
	copy(sk[:], crypto.Keccak256(xSignersHashxProposersHashOne))

	return OCRSecrets{
		SharedSecret: sharedSecret,
		EphemeralSk:  sk,
	}, nil
}
