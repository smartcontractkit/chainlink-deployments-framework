package network

import (
	"errors"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// NetworkType represents the type of network, which can either be mainnet or testnet.
type NetworkType string

const (
	NetworkTypeMainnet NetworkType = "mainnet"
	NetworkTypeTestnet NetworkType = "testnet"
)

// Network represents a network configuration.
type Network struct {
	Type          NetworkType   `yaml:"type"`
	ChainSelector uint64        `yaml:"chain_selector"`
	BlockExplorer BlockExplorer `yaml:"block_explorer"`
	RPCs          []RPC         `yaml:"rpcs"`
	Metadata      any           `yaml:"metadata"`
}

// ChainFamily returns the family of the network based on its chain selector.
func (n *Network) ChainFamily() (string, error) {
	return chain_selectors.GetSelectorFamily(n.ChainSelector)
}

// ChainID returns the chain ID as a string based on the chain selector.
func (n *Network) ChainID() (string, error) {
	return chain_selectors.GetChainIDFromSelector(n.ChainSelector)
}

// Validate validates the network configuration to ensure that all required fields are set.
func (n *Network) Validate() error {
	if n.Type == "" {
		return errors.New("type is required")
	}

	if n.ChainSelector == 0 {
		return errors.New("chain selector is required")
	}

	if len(n.RPCs) == 0 {
		return errors.New("at least one RPC is required")
	}

	return nil
}

// RPC represents an RPC configuration in the flattened structure
type RPC struct {
	RPCName            string `yaml:"rpc_name"`
	PreferredURLScheme string `yaml:"preferred_url_scheme"`
	HTTPURL            string `yaml:"http_url"`
	WSURL              string `yaml:"ws_url"`
}

// PreferredEndpoint returns the correct endpoint based on the preferred URL scheme. By default, it
// returns the HTTP URL.
func (rpc *RPC) PreferredEndpoint() string {
	if rpc.PreferredURLScheme == "ws" {
		return rpc.WSURL
	}

	return rpc.HTTPURL
}

// BlockExplorer represents a block explorer configuration in the flattened structure
type BlockExplorer struct {
	Type   string `yaml:"type"`
	APIKey string `yaml:"api_key"`
	URL    string `yaml:"url"`
}
