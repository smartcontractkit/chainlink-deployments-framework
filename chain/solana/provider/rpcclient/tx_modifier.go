package rpcclient

import (
	sollib "github.com/gagliardetto/solana-go"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/fees"
)

// TxModifier is a dynamic function used to flexibly add components to a transaction such as
// additional signers, and compute budget parameters
type TxModifier func(tx *sollib.Transaction, signers map[sollib.PublicKey]sollib.PrivateKey) error

// AddSigners adds additional signers to the signers map for signing the transaction.
func AddSigners(additionalSigners ...sollib.PrivateKey) TxModifier {
	return func(_ *sollib.Transaction, s map[sollib.PublicKey]sollib.PrivateKey) error {
		for _, v := range additionalSigners {
			s[v.PublicKey()] = v
		}

		return nil
	}
}

// WithComputeUnitLimit sets the total compute unit limit for a transaction.
// The Solana network default is 200K units, with a maximum of 1.4M units.
// Note: Signature verification may consume varying compute units depending on the number of
// signatures.
func WithComputeUnitLimit(v fees.ComputeUnitLimit) TxModifier {
	return func(tx *sollib.Transaction, _ map[sollib.PublicKey]sollib.PrivateKey) error {
		return fees.SetComputeUnitLimit(tx, v)
	}
}

// WithComputeUnitPrice sets the compute unit price for a transaction.
//
// The compute unit price is the price per compute unit, in micro-lamports.
// This is useful for customizing transaction fees or prioritization in Solana-based workflows.
func WithComputeUnitPrice(v fees.ComputeUnitPrice) TxModifier {
	return func(tx *sollib.Transaction, _ map[sollib.PublicKey]sollib.PrivateKey) error {
		return fees.SetComputeUnitPrice(tx, v)
	}
}
