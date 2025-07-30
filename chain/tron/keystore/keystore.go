package keystore

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
)

type Keystore struct {
	Keys map[string]*ecdsa.PrivateKey
}

var _ loop.Keystore = &Keystore{}

func NewKeystore(privateKey *ecdsa.PrivateKey) (*Keystore, address.Address) {
	keys := map[string]*ecdsa.PrivateKey{}
	address := address.PubkeyToAddress(privateKey.PublicKey)

	keys[address.String()] = privateKey

	return &Keystore{Keys: keys}, address
}

func (ks *Keystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
	privateKey, ok := ks.Keys[id]
	if !ok {
		return nil, errors.New("no such key")
	}

	// used to check if the account exists.
	if hash == nil {
		return nil, nil
	}

	return crypto.Sign(hash, privateKey)
}

func (ks *Keystore) ImportECDSA(privateKey *ecdsa.PrivateKey) address.Address {
	address := address.PubkeyToAddress(privateKey.PublicKey)
	ks.Keys[address.String()] = privateKey

	return address
}

func (ks *Keystore) Accounts(ctx context.Context) ([]string, error) {
	accounts := make([]string, 0, len(ks.Keys))
	for id := range ks.Keys {
		accounts = append(accounts, id)
	}

	return accounts, nil
}
