// Package p2pkey contains code adapted from the Chainlink core node with minor modifications.
// These changes allow us to avoid a direct dependency on the Chainlink core node.
//
// https://github.com/smartcontractkit/chainlink/blob/develop/core/services/keystore/keys/p2pkey/peer_id.go
//
// Modifications include:
// - Retained only the functionality for marshaling and unmarshaling PeerID strings.
// - Updated error handling to use the standard library `errors` package.
// - Updates to pass linting and formatting checks.
package p2pkey

import (
	"fmt"
	"strings"

	"github.com/smartcontractkit/libocr/ragep2p/types"
)

const peerIDPrefix = "p2p_"

type PeerID types.PeerID

// MakePeerID creates a PeerID from a string. It returns an error if the string
// cannot be parsed as a PeerID.
func MakePeerID(s string) (PeerID, error) {
	var peerID PeerID

	return peerID, peerID.UnmarshalString(s)
}

// PeerID returns the string representation of the PeerID. It is prefixed with
// "p2p_" to indicate that it is a PeerID. If the PeerID is zero, it returns an
// empty string.
func (p PeerID) String() string {
	// Handle a zero peerID more gracefully, i.e. print it as empty string rather
	// than `p2p_`
	if p == (PeerID{}) {
		return ""
	}

	return fmt.Sprintf("%s%s", peerIDPrefix, p.Raw())
}

// Raw returns the raw string representation of the PeerID without the
// "p2p_" prefix.
func (p PeerID) Raw() string {
	return types.PeerID(p).String()
}

// UnmarshalString unmarshals a string into a PeerID.
func (p *PeerID) UnmarshalString(s string) error {
	return p.UnmarshalText([]byte(s))
}

// MarshalText marshals the PeerID into a byte slice. If the PeerID is zero, it
// returns nil.
func (p *PeerID) MarshalText() ([]byte, error) {
	if *p == (PeerID{}) {
		return nil, nil
	}

	return []byte(p.Raw()), nil
}

// UnmarshalText unmarshals a byte slice into a PeerID. It expects the byte
// slice to be a string representation of a PeerID. If the byte slice is empty,
// it returns nil. If the byte slice has the "p2p_" prefix, it removes it before
// unmarshaling. If the unmarshaling fails, it returns an error.
func (p *PeerID) UnmarshalText(bs []byte) error {
	input := string(bs)
	if strings.HasPrefix(input, peerIDPrefix) {
		input = string(bs[len(peerIDPrefix):])
	}

	if input == "" {
		return nil
	}

	var peerID types.PeerID
	err := peerID.UnmarshalText([]byte(input))
	if err != nil {
		return fmt.Errorf(`PeerID#UnmarshalText("%v"): %w`, input, err)
	}
	*p = PeerID(peerID)

	return nil
}
