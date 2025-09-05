package analyzer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
)

type EVMABIRegistry interface {
	GetABIByAddress(chainSelector uint64, address string) (*abi.ABI, string, error)
	GetAllABIs() map[string]string
	GetABIByType(typeAndVersion cldf.TypeAndVersion) (*abi.ABI, string, error)
	AddABI(contractType cldf.TypeAndVersion, abi string) error
}

var _ EVMABIRegistry = (*environmentEVMRegistry)(nil)

// environmentEVMRegistry is an implementation of EVMABIRegistry that retrieves ABIs from the provided environment using DataStore.
type environmentEVMRegistry struct {
	abiRegistry      map[string]string
	env              cldf.Environment
	addressesByChain cldf.AddressesByChain
}

func (reg environmentEVMRegistry) GetAllABIs() map[string]string {
	return reg.abiRegistry
}

func (reg environmentEVMRegistry) GetABIByAddress(chainSelector uint64, address string) (*abi.ABI, string, error) {
	addressesForChain, ok := reg.addressesByChain[chainSelector]
	if !ok {
		return nil, "", fmt.Errorf("no addresses found for chain selector %d", chainSelector)
	}
	addressTypeAndVersion, ok := addressesForChain[address]
	if !ok {
		return nil, "", fmt.Errorf("address %s not found for chain selector %d", address, chainSelector)
	}

	return reg.GetABIByType(addressTypeAndVersion)
}

func (reg environmentEVMRegistry) GetABIByType(typeAndVersion cldf.TypeAndVersion) (*abi.ABI, string, error) {
	registryKey := cldf.TypeAndVersion{Type: typeAndVersion.Type, Version: typeAndVersion.Version}.String()
	abiStr, found := reg.abiRegistry[registryKey]
	if !found {
		return nil, "", fmt.Errorf("ABI not found for type and version %v", typeAndVersion)
	}

	abiObj, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse ABI for typeAndVersion %v: %w", typeAndVersion, err)
	}

	return &abiObj, abiStr, nil
}

func (reg environmentEVMRegistry) AddABI(typeAndVersion cldf.TypeAndVersion, abiStr string) error {
	_, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return fmt.Errorf("failed to parse ABI for typeAndVersion %s", typeAndVersion.String())
	}
	reg.abiRegistry[typeAndVersion.String()] = abiStr

	return nil
}

// NewEnvironmentEVMRegistry creates a new environmentEVMRegistry from the provided ABI mappings and domain name.
func NewEnvironmentEVMRegistry(env cldf.Environment, abiMappings map[string]string) (*environmentEVMRegistry, error) {
	addressesByChain, errAddrBook := env.ExistingAddresses.Addresses()
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

	return &environmentEVMRegistry{
		abiRegistry:      abiMappings,
		env:              env,
		addressesByChain: addressesByChain,
	}, nil
}
