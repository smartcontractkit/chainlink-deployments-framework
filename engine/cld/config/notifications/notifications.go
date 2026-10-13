package notifications

import (
	"errors"
	"fmt"
	"strings"
)

// Config is the notification routing for one domain environment.
// The slack section has the same shape as a domain's
// .config/mcms/<env>/notifications-slack.yaml, so such a file maps onto it
// unchanged.
type Config struct {
	Slack SlackConfig `mapstructure:"slack" yaml:"slack"`
}

// SlackConfig lists the Slack channels notifications are posted to.
type SlackConfig struct {
	Channels []SlackChannel `mapstructure:"channels" yaml:"channels"`
}

// SlackChannel is a Slack channel and the proposals routed to it.
//
// A channel with neither timelock nor MCM contracts receives every proposal.
// Otherwise it receives only proposals that target one of its contracts.
type SlackChannel struct {
	// Name is the channel name, including the leading '#'.
	Name string `mapstructure:"name" yaml:"name"`
	// SlackHandles are the users or groups mentioned in the message, e.g.
	// "@signers".
	SlackHandles      []string      `mapstructure:"slack_handles" yaml:"slack_handles,omitempty"`
	TimelockContracts []ContractRef `mapstructure:"timelock_contracts" yaml:"timelock_contracts,omitempty"`
	MCMContracts      []ContractRef `mapstructure:"mcm_contracts" yaml:"mcm_contracts,omitempty"`
}

// ContractRef identifies a contract by chain selector and address.
type ContractRef struct {
	ChainSelector uint64 `mapstructure:"chain_selector" yaml:"chain_selector"`
	Address       string `mapstructure:"address" yaml:"address"`
}

// Validate checks that every channel is named and every contract is fully
// identified.
func (c *Config) Validate() error {
	var errs []error
	for i, channel := range c.Slack.Channels {
		if err := channel.validate(); err != nil {
			errs = append(errs, fmt.Errorf("slack.channels[%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func (c SlackChannel) validate() error {
	if !strings.HasPrefix(c.Name, "#") || len(c.Name) == 1 {
		return fmt.Errorf("name %q must be a channel name starting with '#'", c.Name)
	}

	var errs []error
	for i, ref := range c.TimelockContracts {
		if err := ref.validate(); err != nil {
			errs = append(errs, fmt.Errorf("timelock_contracts[%d]: %w", i, err))
		}
	}
	for i, ref := range c.MCMContracts {
		if err := ref.validate(); err != nil {
			errs = append(errs, fmt.Errorf("mcm_contracts[%d]: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func (r ContractRef) validate() error {
	if r.ChainSelector == 0 {
		return errors.New("chain_selector is required")
	}
	if r.Address == "" {
		return errors.New("address is required")
	}

	return nil
}
