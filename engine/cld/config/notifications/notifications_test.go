package notifications

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	contract := ContractRef{ChainSelector: 16015286601757825753, Address: "0xabc"}

	tests := []struct {
		name    string
		give    Config
		wantErr string
	}{
		{
			name: "no channels",
		},
		{
			name: "channel for every proposal",
			give: Config{Slack: SlackConfig{Channels: []SlackChannel{{Name: "#mcms-proposals"}}}},
		},
		{
			name: "channel routed by contract",
			give: Config{Slack: SlackConfig{Channels: []SlackChannel{{
				Name:              "#ccip",
				SlackHandles:      []string{"@ccip-signers"},
				TimelockContracts: []ContractRef{contract},
				MCMContracts:      []ContractRef{contract},
			}}}},
		},
		{
			name:    "missing channel name",
			give:    Config{Slack: SlackConfig{Channels: []SlackChannel{{}}}},
			wantErr: `slack.channels[0]: name "" must be a channel name starting with '#'`,
		},
		{
			name:    "channel name without '#'",
			give:    Config{Slack: SlackConfig{Channels: []SlackChannel{{Name: "ccip"}}}},
			wantErr: `slack.channels[0]: name "ccip" must be a channel name starting with '#'`,
		},
		{
			name: "contract without chain selector",
			give: Config{Slack: SlackConfig{Channels: []SlackChannel{{
				Name:              "#ccip",
				TimelockContracts: []ContractRef{{Address: "0xabc"}},
			}}}},
			wantErr: "slack.channels[0]: timelock_contracts[0]: chain_selector is required",
		},
		{
			name: "contract without address",
			give: Config{Slack: SlackConfig{Channels: []SlackChannel{
				{Name: "#ok"},
				{Name: "#ccip", MCMContracts: []ContractRef{{ChainSelector: 1}}},
			}}},
			wantErr: "slack.channels[1]: mcm_contracts[0]: address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.give.Validate()

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
		})
	}
}

// TestConfig_DecodesNotificationsSlackFile pins that a domain's
// notifications-slack.yaml decodes into the slack section unchanged.
func TestConfig_DecodesNotificationsSlackFile(t *testing.T) {
	t.Parallel()

	const file = `
channels:
  - name: "#ccip"
    slack_handles: ["@ccip-signers"]
    timelock_contracts:
      - chain_selector: 16015286601757825753
        address: "0xabc"
    mcm_contracts:
      - chain_selector: 16015286601757825753
        address: "0xdef"
  - name: "#mcms-proposals"
`

	var got SlackConfig
	require.NoError(t, yaml.Unmarshal([]byte(file), &got))

	assert.Equal(t, SlackConfig{Channels: []SlackChannel{
		{
			Name:              "#ccip",
			SlackHandles:      []string{"@ccip-signers"},
			TimelockContracts: []ContractRef{{ChainSelector: 16015286601757825753, Address: "0xabc"}},
			MCMContracts:      []ContractRef{{ChainSelector: 16015286601757825753, Address: "0xdef"}},
		},
		{Name: "#mcms-proposals"},
	}}, got)
}
