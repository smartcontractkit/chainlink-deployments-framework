package analyzer

import (
	"fmt"
	"maps"

	"github.com/gagliardetto/solana-go"

	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/memo"
	"github.com/gagliardetto/solana-go/programs/stake"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/programs/tokenregistry"
	"github.com/gagliardetto/solana-go/programs/vote"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/solana/programs/loaderv3"
	verify "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/solana/programs/otter_verify"
)

// SolanaDecoderRegistry is an interface for retrieving and managing Solana instruction decoders.
type SolanaDecoderRegistry interface {
	Decoders() map[string]DecodeInstructionFn
	GetSolanaInstructionDecoderByAddress(chainSelector uint64, address string) (DecodeInstructionFn, error)
	GetSolanaInstructionDecoderByType(typeAndVersion deployment.TypeAndVersion) (DecodeInstructionFn, error)
	AddSolanaInstructionDecoder(contractType deployment.TypeAndVersion, decoder DecodeInstructionFn)
}

var _ SolanaDecoderRegistry = (*environmentSolanaRegistry)(nil)

// environmentSolanaRegistry is an implementation of SolanaDecoderRegistry that retrieves sol decoders from the provided environment using DataStore.
type environmentSolanaRegistry struct {
	registry         map[string]DecodeInstructionFn
	env              deployment.Environment
	addressesByChain deployment.AddressesByChain
}

func (reg environmentSolanaRegistry) Decoders() map[string]DecodeInstructionFn {
	out := make(map[string]DecodeInstructionFn, len(reg.registry))
	for k, v := range reg.registry {
		out[k] = v
	}

	return out
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

func (reg environmentSolanaRegistry) GetSolanaInstructionDecoderByType(typeAndVersion deployment.TypeAndVersion) (DecodeInstructionFn, error) {
	registryKey := deployment.TypeAndVersion{Type: typeAndVersion.Type, Version: typeAndVersion.Version}.String()
	decoder, found := reg.registry[registryKey]
	if !found {
		return nil, fmt.Errorf("ABI not found for type and version %v", typeAndVersion)
	}

	return decoder, nil
}

func (reg environmentSolanaRegistry) AddSolanaInstructionDecoder(typeAndVersion deployment.TypeAndVersion, decoder DecodeInstructionFn) {
	reg.registry[typeAndVersion.String()] = decoder
}

var nativePrograms = map[solana.PublicKey]deployment.TypeAndVersion{
	solana.BPFLoaderUpgradeableProgramID: deployment.MustTypeAndVersionFromString("BPFLoaderUpgradeable 1.0.0"),
	solana.SystemProgramID:               deployment.MustTypeAndVersionFromString("System 1.0.0"),
	solana.ComputeBudget:                 deployment.MustTypeAndVersionFromString("ComputeBudget 1.0.0"),
	solana.MemoProgramID:                 deployment.MustTypeAndVersionFromString("Memo 1.0.0"),
	solana.StakeProgramID:                deployment.MustTypeAndVersionFromString("Stake 1.0.0"),
	solana.TokenProgramID:                deployment.MustTypeAndVersionFromString("Token 1.0.0"),
	solana.VoteProgramID:                 deployment.MustTypeAndVersionFromString("Vote 1.0.0"),
	verify.ProgramID:                     deployment.MustTypeAndVersionFromString("OtterVerify 1.0.0"),
	tokenregistry.ProgramID():            deployment.MustTypeAndVersionFromString("TokenRegistry 1.0.0"),
}

var nativeDecoderMappings = map[string]DecodeInstructionFn{
	nativePrograms[solana.BPFLoaderUpgradeableProgramID].String(): DIFn(loaderv3.DecodeInstruction),
	nativePrograms[solana.ComputeBudget].String():                 DIFn(computebudget.DecodeInstruction),
	nativePrograms[solana.MemoProgramID].String():                 DIFn(memo.DecodeInstruction),
	nativePrograms[solana.StakeProgramID].String():                DIFn(stake.DecodeInstruction),
	nativePrograms[solana.SystemProgramID].String():               DIFn(system.DecodeInstruction),
	nativePrograms[solana.TokenProgramID].String():                DIFn(token.DecodeInstruction),
	nativePrograms[solana.VoteProgramID].String():                 DIFn(vote.DecodeInstruction),
	nativePrograms[tokenregistry.ProgramID()].String():            DIFn(tokenregistry.DecodeInstruction),
	nativePrograms[verify.ProgramID].String():                     DIFn(verify.DecodeInstruction),
}

// NewEnvironmentSolanaRegistry creates a new environmentSolanaRegistry from the provided ABI mappings and domain name.
func NewEnvironmentSolanaRegistry(env deployment.Environment, decoderMappings map[string]DecodeInstructionFn) (*environmentSolanaRegistry, error) {
	if decoderMappings == nil {
		decoderMappings = map[string]DecodeInstructionFn{}
	} else {
		decoderMappings = maps.Clone(decoderMappings)
	}

	addressesByChain, errAddrBook := env.ExistingAddresses.Addresses() //nolint:staticcheck // using deprecated API intentionally
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
			chainAddresses = map[string]deployment.TypeAndVersion{}
		}
		chainAddresses[address.Address] = deployment.TypeAndVersion{
			Type:    deployment.ContractType(address.Type),
			Version: pointer.DerefOrEmpty(address.Version),
			Labels:  deployment.NewLabelSet(address.Labels.List()...),
		}
		addressesByChain[address.ChainSelector] = chainAddresses
	}

	// add native programs and mappings
	for chainSelector, addresses := range addressesByChain {
		for pk, contractTypeAndVersion := range nativePrograms {
			addresses[pk.String()] = contractTypeAndVersion
		}
		addressesByChain[chainSelector] = addresses
	}
	maps.Insert(decoderMappings, maps.All(nativeDecoderMappings))

	return &environmentSolanaRegistry{
		registry:         decoderMappings,
		env:              env,
		addressesByChain: addressesByChain,
	}, nil
}
