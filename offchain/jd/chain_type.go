package jd

import (
	"errors"
	"fmt"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
)

// ChainTypeToFamily converts a JD proto ChainType to a chain selector family.
func ChainTypeToFamily(chainType nodev1.ChainType) (string, error) {
	var family string
	switch chainType {
	case nodev1.ChainType_CHAIN_TYPE_EVM:
		family = chain_selectors.FamilyEVM
	case nodev1.ChainType_CHAIN_TYPE_APTOS:
		family = chain_selectors.FamilyAptos
	case nodev1.ChainType_CHAIN_TYPE_SOLANA:
		family = chain_selectors.FamilySolana
	case nodev1.ChainType_CHAIN_TYPE_STARKNET:
		family = chain_selectors.FamilyStarknet
	case nodev1.ChainType_CHAIN_TYPE_TRON:
		family = chain_selectors.FamilyTron
	case nodev1.ChainType_CHAIN_TYPE_TON:
		family = chain_selectors.FamilyTon
	case nodev1.ChainType_CHAIN_TYPE_SUI:
		family = chain_selectors.FamilySui
	case nodev1.ChainType_CHAIN_TYPE_UNSPECIFIED:
		return "", errors.New("chain type must be specified")
	default:
		return "", fmt.Errorf("unsupported chain type %s", chainType)
	}

	return family, nil
}
