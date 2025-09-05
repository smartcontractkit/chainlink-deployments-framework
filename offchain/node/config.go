package node

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/internal/p2pkey"
)

// MinimalNodeCfg is the minimal configuration that can be used to identify a node.
// It represents information that cannot be programmatically derived from Job Distributor.
type MinimalNodeCfg struct {
	Name                string `json:"name" toml:"name" yaml:"name"`
	CSAKey              string `json:"csa_key" toml:"csa_key" yaml:"csa_key"`
	NOP                 string `json:"nop" toml:"nop" yaml:"nop"`
	EncryptionPublicKey string `json:"encryption_public_key" toml:"encryption_public_key" yaml:"encryption_public_key"`
}

// NodeCfg is the configuration for a node.
// It is used to register a node with the job distributor and contains only public information, no secrets or api keys.
type NodeCfg struct {
	MinimalNodeCfg
	P2PID string `json:"p2p_id" toml:"p2p_id" yaml:"p2p_id"`

	AdminAddr    string            `json:"admin_addr" toml:"admin_addr" yaml:"admin_addr"`
	MultiAddress *string           `json:"multi_address,omitempty" toml:"multi_address,omitempty" yaml:"multi_address,omitempty"` // needed only for bootstrap nodes, bootstrap IP:P2P PORT
	Tags         map[string]string `json:"tags,omitempty" toml:"tags,omitempty" yaml:"tags,omitempty"`
}

func (n NodeCfg) Validate() error {
	if n.Name == "" {
		return errors.New("no name in node")
	}
	if n.CSAKey == "" {
		return errors.New("no CSAKey in node")
	}
	if n.P2PID == "" {
		return errors.New("no P2PID in node")
	}
	_, err := p2pkey.MakePeerID(n.P2PID)
	if err != nil {
		return fmt.Errorf("invalid P2PID '%s' in node: %w", n.P2PID, err)
	}
	if n.NOP == "" {
		return errors.New("no Nop in node")
	}
	if n.AdminAddr == "" {
		return errors.New("no AdminAddr in node")
	}
	var a = new(common.Address)
	err = a.UnmarshalText([]byte(n.AdminAddr))
	if err != nil {
		return fmt.Errorf("invalid AdminAddr '%s' in node: %w", n.AdminAddr, err)
	}

	return nil
}

// Labels returns the labels for the node, containing the p2p_id, nop, admin_addr and all tags.
func (n NodeCfg) Labels() map[string]string {
	m := map[string]string{
		"p2p_id":     n.P2PID,
		"nop":        strings.ReplaceAll(n.NOP, " ", "_"),
		"admin_addr": n.AdminAddr,
	}
	for k, v := range n.Tags {
		m[k] = v
	}
	if n.MultiAddress != nil {
		m["multi_address"] = *n.MultiAddress
	}

	return m
}

func (n NodeCfg) IsBootstrap() bool {
	return n.MultiAddress != nil
}
