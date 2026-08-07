package config

import (
	"fmt"

	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadBinaryConfig retrieves the binary configuration for a given domain.
// When the binary section is omitted from domain.yaml, the config defaults to
// building from source with version "latest".
func LoadBinaryConfig(dom fdomain.Domain) (*cfgdomain.BinaryConfig, error) {
	domainConfig, err := cfgdomain.Load(dom.ConfigDomainFilePath())
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config: %w", err)
	}

	return domainConfig.Binary, nil
}
