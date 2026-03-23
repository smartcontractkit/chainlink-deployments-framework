package contract

import (
	"errors"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/text"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

var (
	contractShort = "Contract verification commands"

	contractLong = text.LongDesc(`
		Commands for verifying deployed contracts on block explorers.

		Verify contracts in an environment using the datastore or catalog to discover
		contracts. Requires a domain-specific ContractInputsProvider to supply
		contract metadata (Standard JSON, bytecode) for verification.

		Currently supports only EVM-based chains, but can be extended to other chain types in the future.
	`)
)

// Config holds the configuration for contract commands.
type Config struct {
	Logger                 logger.Logger
	Domain                 domain.Domain
	ContractInputsProvider evm.ContractInputsProvider
	// VerifierHTTPClient is optional; when set, verifiers use it for API calls (e.g. in tests).
	VerifierHTTPClient *http.Client

	Deps Deps
}

// Validate checks that all required configuration fields are set.
func (c Config) Validate() error {
	var missing []string
	if c.Logger == nil {
		missing = append(missing, "Logger")
	}
	if c.Domain.RootPath() == "" {
		missing = append(missing, "Domain")
	}
	if c.ContractInputsProvider == nil {
		missing = append(missing, "ContractInputsProvider")
	}
	if len(missing) > 0 {
		return errors.New("contract.Config: missing required fields: " + strings.Join(missing, ", "))
	}

	return nil
}

func (c *Config) deps() {
	c.Deps.applyDefaults()
}

// NewCommand creates the contract command with verify-env subcommand.
func NewCommand(cfg Config) (*cobra.Command, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.deps()

	cmd := &cobra.Command{
		Use:   "contract",
		Short: contractShort,
		Long:  contractLong,
	}
	cmd.AddCommand(NewVerifyEnvCmd(cfg))

	return cmd, nil
}

// NewVerifyEnvCmd creates the verify-env subcommand. Exported for domains that want
// to add it as a top-level command (e.g. verify-evm) while using the same implementation.
func NewVerifyEnvCmd(cfg Config) *cobra.Command {
	return newVerifyEnvCmdWithUse(cfg, "verify-env")
}

// NewVerifyEnvCmdWithUse creates the verify command with a custom Use string.
// Use this when adding as a top-level command with a custom name (e.g. "verify-evm").
func NewVerifyEnvCmdWithUse(cfg Config, use string) *cobra.Command {
	return newVerifyEnvCmdWithUse(cfg, use)
}
