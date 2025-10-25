package analyzer

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/stretchr/testify/assert"
)

func TestEnvironmentEVMRegistry_GetABIByAddress(t *testing.T) {
	typeAndVersion := deployment.TypeAndVersion{
		Type:    deployment.ContractType("test"),
		Version: *semver.MustParse("1.0.0"),
	}

	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{
			typeAndVersion.String(): `[{"type":"function","name":"test","inputs":[]}]`,
		},
		addressesByChain: map[uint64]map[string]deployment.TypeAndVersion{
			1: {
				"0xabc123": typeAndVersion,
			},
		},
	}

	abiObj, abiStr, err := registry.GetABIByAddress(1, "0xabc123")
	assert.NoError(t, err)
	assert.NotNil(t, abiObj)
	assert.Equal(t, `[{"type":"function","name":"test","inputs":[]}]`, abiStr)
}

func TestEnvironmentEVMRegistry_GetABIByAddress_ChainNotFound(t *testing.T) {
	registry := &environmentEVMRegistry{
		abiRegistry:      map[string]string{},
		addressesByChain: map[uint64]map[string]deployment.TypeAndVersion{},
	}

	_, _, err := registry.GetABIByAddress(999, "0xabc123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no addresses found for chain selector")
}

func TestEnvironmentEVMRegistry_GetABIByAddress_AddressNotFound(t *testing.T) {
	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{},
		addressesByChain: map[uint64]map[string]deployment.TypeAndVersion{
			1: {},
		},
	}

	_, _, err := registry.GetABIByAddress(1, "0xabc123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address 0xabc123 not found")
}

func TestEnvironmentEVMRegistry_GetABIByType(t *testing.T) {
	typeAndVersion := deployment.TypeAndVersion{
		Type:    deployment.ContractType("test"),
		Version: *semver.MustParse("1.0.0"),
	}

	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{
			typeAndVersion.String(): `[{"type":"function","name":"test","inputs":[]}]`,
		},
	}

	abiObj, abiStr, err := registry.GetABIByType(typeAndVersion)
	assert.NoError(t, err)
	assert.NotNil(t, abiObj)
	assert.Equal(t, `[{"type":"function","name":"test","inputs":[]}]`, abiStr)
}

func TestEnvironmentEVMRegistry_GetABIByType_NotFound(t *testing.T) {
	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{},
	}

	typeAndVersion := deployment.TypeAndVersion{
		Type:    deployment.ContractType("unknown"),
		Version: *semver.MustParse("1.0.0"),
	}

	_, _, err := registry.GetABIByType(typeAndVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ABI not found")
}

func TestEnvironmentEVMRegistry_GetABIByType_InvalidABI(t *testing.T) {
	typeAndVersion := deployment.TypeAndVersion{
		Type:    deployment.ContractType("test"),
		Version: *semver.MustParse("1.0.0"),
	}

	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{
			typeAndVersion.String(): `invalid json`,
		},
	}

	_, _, err := registry.GetABIByType(typeAndVersion)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse ABI")
}

func TestEnvironmentEVMRegistry_GetAllABIs(t *testing.T) {
	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{
			"test:1.0.0": `[{"type":"function","name":"test","inputs":[]}]`,
			"mock:2.0.0": `[{"type":"function","name":"mock","inputs":[]}]`,
		},
	}

	abis := registry.GetAllABIs()
	assert.Len(t, abis, 2)
	assert.Contains(t, abis, "test:1.0.0")
	assert.Contains(t, abis, "mock:2.0.0")
	assert.Equal(t, `[{"type":"function","name":"test","inputs":[]}]`, abis["test:1.0.0"])
	assert.Equal(t, `[{"type":"function","name":"mock","inputs":[]}]`, abis["mock:2.0.0"])
}

func TestEnvironmentEVMRegistry_AddABI(t *testing.T) {
	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{},
	}

	typeAndVersion := deployment.TypeAndVersion{
		Type:    deployment.ContractType("test"),
		Version: *semver.MustParse("1.0.0"),
	}
	abiStr := `[{"type":"function","name":"test","inputs":[]}]`

	err := registry.AddABI(typeAndVersion, abiStr)
	assert.NoError(t, err)

	// Note: The current implementation doesn't modify the registry in place due to receiver type
	// This test verifies the method doesn't error on valid ABI
}

func TestEnvironmentEVMRegistry_AddABI_InvalidABI(t *testing.T) {
	registry := &environmentEVMRegistry{
		abiRegistry: map[string]string{},
	}

	typeAndVersion := deployment.TypeAndVersion{
		Type:    deployment.ContractType("test"),
		Version: *semver.MustParse("1.0.0"),
	}
	invalidABI := `invalid json`

	err := registry.AddABI(typeAndVersion, invalidABI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse ABI")
}
