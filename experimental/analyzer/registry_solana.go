package analyzer

import (
	"fmt"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
)

// SolanaDecoderRegistry is an interface for retrieving and managing Solana instruction decoders.
type SolanaDecoderRegistry interface {
	GetSolanaInstructionDecoderByAddress(chainSelector uint64, address string) (DecodeInstructionFn, error)
	GetSolanaInstructionDecoderByType(typeAndVersion cldf.TypeAndVersion) (DecodeInstructionFn, error)
	AddSolanaInstructionDecoder(contractType cldf.TypeAndVersion, decoder DecodeInstructionFn)
}

var _ SolanaDecoderRegistry = (*environmentSolanaRegistry)(nil)

// environmentSolanaRegistry is an implementation of SolanaDecoderRegistry that retrieves sol decoders from the provided environment using DataStore.
type environmentSolanaRegistry struct {
	registry         map[string]DecodeInstructionFn
	env              cldf.Environment
	addressesByChain cldf.AddressesByChain
}

func (reg environmentSolanaRegistry) GetSolanaInstructionDecoderByAddress(chainSelector uint64, address string) (DecodeInstructionFn, error) {
	addressesForChain, ok := reg.addressesByChain[chainSelector]
	if !ok {
		return nil, fmt.Errorf("no addresses found for chain selector %d", chainSelector)
	}
	addressTypeAndVersion, ok := addressesForChain[address]
	if !ok {
		return nil, fmt.Errorf("address %s not found for chain selector %d", address, chainSelector)
	}

	return reg.GetSolanaInstructionDecoderByType(addressTypeAndVersion)
}

func (reg environmentSolanaRegistry) GetSolanaInstructionDecoderByType(typeAndVersion cldf.TypeAndVersion) (DecodeInstructionFn, error) {
	registryKey := cldf.TypeAndVersion{Type: typeAndVersion.Type, Version: typeAndVersion.Version}.String()
	decoder, found := reg.registry[registryKey]
	if !found {
		return nil, fmt.Errorf("ABI not found for type and version %v", typeAndVersion)
	}

	return decoder, nil
}

func (reg environmentSolanaRegistry) AddSolanaInstructionDecoder(typeAndVersion cldf.TypeAndVersion, decoder DecodeInstructionFn) {
	reg.registry[typeAndVersion.String()] = decoder
}

// NewEnvironmentSolanaRegistry creates a new environmentSolanaRegistry from the provided ABI mappings and domain name.
func NewEnvironmentSolanaRegistry(env cldf.Environment, decoderMappings map[string]DecodeInstructionFn) (*environmentSolanaRegistry, error) {
	addressesByChain, errAddrBook := env.ExistingAddresses.Addresses() //nolint:staticcheck
	if errAddrBook != nil {
		return nil, errAddrBook
	}
	dataStoreAddresses, err := env.DataStore.Addresses().Fetch()
	if err != nil {
		return nil, err
	}
	for _, address := range dataStoreAddresses {
		chainAddresses, exists := addressesByChain[address.ChainSelector]
		if !exists {
			chainAddresses = map[string]cldf.TypeAndVersion{}
		}
		chainAddresses[address.Address] = cldf.TypeAndVersion{
			Type:    cldf.ContractType(address.Type),
			Version: pointer.DerefOrEmpty(address.Version),
			Labels:  cldf.NewLabelSet(address.Labels.List()...),
		}
		addressesByChain[address.ChainSelector] = chainAddresses
	}

	return &environmentSolanaRegistry{
		registry:         decoderMappings,
		env:              env,
		addressesByChain: addressesByChain,
	}, nil
}
