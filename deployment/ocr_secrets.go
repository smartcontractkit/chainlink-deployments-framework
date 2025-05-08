package deployment

import "github.com/ethereum/go-ethereum/crypto"

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
