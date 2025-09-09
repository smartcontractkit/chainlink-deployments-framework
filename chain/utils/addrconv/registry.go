package addrconv

import (
	"fmt"
	"sync"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
)

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *addressConverterRegistry
)

func registry() *addressConverterRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = newAddressConverterRegistry()
	})

	return defaultRegistry
}

// ToBytes converts an address string to bytes based on the chain family.
//
// Usage:
//
//	bytes, err := addrconv.ToBytes("evm", "0x742d35Cc...")
func ToBytes(family, address string) ([]byte, error) {
	return registry().convertAddressByFamily(family, address)
}

// addressConverterRegistry manages address conversion strategies for different chain families.
// It uses the strategy pattern to delegate address conversion to the appropriate implementation.
type addressConverterRegistry struct {
	converters map[string]Converter
}

// newAddressConverterRegistry creates a new registry with all supported chain converters pre-registered.
func newAddressConverterRegistry() *addressConverterRegistry {
	registry := &addressConverterRegistry{
		converters: make(map[string]Converter),
	}

	// Register all supported converters using constants
	registry.converters[chain_selectors.FamilyEVM] = evm.AddressConverter{}
	registry.converters[chain_selectors.FamilySolana] = solana.AddressConverter{}
	registry.converters[chain_selectors.FamilyAptos] = aptos.AddressConverter{}
	registry.converters[chain_selectors.FamilySui] = sui.AddressConverter{}
	registry.converters[chain_selectors.FamilyTon] = ton.AddressConverter{}
	registry.converters[chain_selectors.FamilyTron] = tron.AddressConverter{}

	return registry
}

// convertAddressByFamily converts an address string to bytes using the family name.
func (r *addressConverterRegistry) convertAddressByFamily(family, address string) ([]byte, error) {
	converter, exists := r.converters[family]

	if !exists {
		return nil, fmt.Errorf("no address converter registered for family: %s", family)
	}

	return converter.ConvertToBytes(address)
}
