package keystore

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
)

// Keystore is a simple in-memory key store that holds private keys for signing.
// The `Keys` map holds:
//   - key (string): the string representation of a Tron address
//   - value (*ecdsa.PrivateKey): the private key associated with that address
type Keystore struct {
	Keys map[string]*ecdsa.PrivateKey
}

// Assert that *Keystore implements the loop.Keystore interface
var _ loop.Keystore = &Keystore{}

// NewKeystore initializes a new Keystore with a single ECDSA private key.
// The key is stored using the derived Tron address as the map key.
func NewKeystore(privateKey *ecdsa.PrivateKey) (*Keystore, address.Address) {
	keys := map[string]*ecdsa.PrivateKey{}
	address := address.PubkeyToAddress(privateKey.PublicKey)

	keys[address.String()] = privateKey

	return &Keystore{Keys: keys}, address
}

// Sign signs the given hash using the private key associated with the provided ID (address string).
// If the key does not exist, it returns an error.
// If the hash is nil (e.g., used as an existence check), it returns nil.
func (ks *Keystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
	privateKey, ok := ks.Keys[id]
	if !ok {
		return nil, errors.New("no such key")
	}

	// If hash is nil, don't perform actual signing. This is used to check key existence.
	if hash == nil {
		return nil, nil
	}

	return crypto.Sign(hash, privateKey)
}

func (ks *Keystore) Decrypt(ctx context.Context, id string, ctxt []byte) ([]byte, error) {
	return nil, errors.New("decrypt not implemented in Tron Keystore")
}

// ImportECDSA adds a new private key to the Keystore, deriving its Tron address
// and storing it using that address as the map key.
func (ks *Keystore) ImportECDSA(privateKey *ecdsa.PrivateKey) address.Address {
	address := address.PubkeyToAddress(privateKey.PublicKey)
	ks.Keys[address.String()] = privateKey

	return address
}

// Accounts returns a list of all address strings currently stored in the Keystore.
func (ks *Keystore) Accounts(ctx context.Context) ([]string, error) {
	accounts := make([]string, 0, len(ks.Keys))
	for id := range ks.Keys {
		accounts = append(accounts, id)
	}

	return accounts, nil
}
