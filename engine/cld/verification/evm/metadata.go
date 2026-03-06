package evm

import (
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// SolidityContractMetadata defines the metadata for a Solidity contract used for verification.
// Domains provide this via ContractInputsProvider.
type SolidityContractMetadata struct {
	Version  string         `json:"version"`
	Language string         `json:"language"`
	Settings map[string]any `json:"settings"`
	Sources  map[string]any `json:"sources"`
	Bytecode string         `json:"bytecode"`
	Name     string         `json:"name"`
}

// SourceCode returns the source code portion for explorer API submission.
func (s SolidityContractMetadata) SourceCode() (string, error) {
	sourceCodeMap := map[string]any{
		"language": s.Language,
		"settings": s.Settings,
		"sources":  s.Sources,
	}
	jsonBytes, err := json.Marshal(sourceCodeMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal source code: %w", err)
	}

	return string(jsonBytes), nil
}

// ContractInputsProvider is implemented by domains to provide contract metadata for verification.
// Framework has no knowledge of domain-specific contract types; it only passes the metadata to explorers.
type ContractInputsProvider interface {
	// GetInputs returns metadata for the given contract type and version.
	// Returns an error if the domain does not support this contract type/version.
	GetInputs(contractType datastore.ContractType, version *semver.Version) (SolidityContractMetadata, error)
}
