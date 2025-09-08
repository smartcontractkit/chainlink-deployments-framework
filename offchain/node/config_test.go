package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

const (
	validP2PID     = "12D3KooWM1111111111111111111111111111111111111111111"
	validCSAKey    = "csa_key_123"
	validAdminAddr = "0x1234567890123456789012345678901234567890"
	validNOP       = "Test NOP"
	validName      = "test-node"
	validEncKey    = "encryption_key_123"
)

func TestNodeCfg_Validate_Success(t *testing.T) {
	t.Parallel()

	cfg := NodeCfg{
		MinimalNodeCfg: MinimalNodeCfg{
			Name:                validName,
			CSAKey:              validCSAKey,
			NOP:                 validNOP,
			EncryptionPublicKey: validEncKey,
		},
		P2PID:     validP2PID,
		AdminAddr: validAdminAddr,
	}

	err := cfg.Validate()
	require.NoError(t, err)
}

func TestNodeCfg_Validate_Failures(t *testing.T) {
	t.Parallel()

	baseConfig := NodeCfg{
		MinimalNodeCfg: MinimalNodeCfg{
			Name:                validName,
			CSAKey:              validCSAKey,
			NOP:                 validNOP,
			EncryptionPublicKey: validEncKey,
		},
		P2PID:     validP2PID,
		AdminAddr: validAdminAddr,
	}

	tests := []struct {
		name      string
		modifyFn  func(*NodeCfg)
		wantError string
	}{
		{
			name: "empty name",
			modifyFn: func(cfg *NodeCfg) {
				cfg.Name = ""
			},
			wantError: "no name in node",
		},
		{
			name: "empty CSAKey",
			modifyFn: func(cfg *NodeCfg) {
				cfg.CSAKey = ""
			},
			wantError: "no CSAKey in node",
		},
		{
			name: "empty P2PID",
			modifyFn: func(cfg *NodeCfg) {
				cfg.P2PID = ""
			},
			wantError: "no P2PID in node",
		},
		{
			name: "invalid P2PID",
			modifyFn: func(cfg *NodeCfg) {
				cfg.P2PID = "invalid_p2p_id"
			},
			wantError: "invalid P2PID 'invalid_p2p_id' in node",
		},
		{
			name: "empty NOP",
			modifyFn: func(cfg *NodeCfg) {
				cfg.NOP = ""
			},
			wantError: "no Nop in node",
		},
		{
			name: "empty AdminAddr",
			modifyFn: func(cfg *NodeCfg) {
				cfg.AdminAddr = ""
			},
			wantError: "no AdminAddr in node",
		},
		{
			name: "invalid AdminAddr",
			modifyFn: func(cfg *NodeCfg) {
				cfg.AdminAddr = "invalid_address"
			},
			wantError: "invalid AdminAddr 'invalid_address' in node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := baseConfig // copy
			tt.modifyFn(&cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantError)
		})
	}
}

func TestNodeCfg_Labels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      NodeCfg
		expected map[string]string
	}{
		{
			name: "basic labels without tags",
			cfg: NodeCfg{
				MinimalNodeCfg: MinimalNodeCfg{
					Name:   validName,
					CSAKey: validCSAKey,
					NOP:    validNOP,
				},
				P2PID:     validP2PID,
				AdminAddr: validAdminAddr,
			},
			expected: map[string]string{
				"p2p_id":     validP2PID,
				"nop":        "Test_NOP", // spaces replaced with underscores
				"admin_addr": validAdminAddr,
			},
		},
		{
			name: "labels with tags",
			cfg: NodeCfg{
				MinimalNodeCfg: MinimalNodeCfg{
					Name:   validName,
					CSAKey: validCSAKey,
					NOP:    "Test NOP With Spaces",
				},
				P2PID:     validP2PID,
				AdminAddr: validAdminAddr,
				Tags: map[string]string{
					"region":      "us-east-1",
					"environment": "testnet",
				},
			},
			expected: map[string]string{
				"p2p_id":      validP2PID,
				"nop":         "Test_NOP_With_Spaces", // spaces replaced with underscores
				"admin_addr":  validAdminAddr,
				"region":      "us-east-1",
				"environment": "testnet",
			},
		},
		{
			name: "bootstrap node with multi_address",
			cfg: NodeCfg{
				MinimalNodeCfg: MinimalNodeCfg{
					Name:   validName,
					CSAKey: validCSAKey,
					NOP:    validNOP,
				},
				P2PID:        validP2PID,
				AdminAddr:    validAdminAddr,
				MultiAddress: pointer.To("192.168.1.1:8080"),
			},
			expected: map[string]string{
				"p2p_id":        validP2PID,
				"nop":           "Test_NOP",
				"admin_addr":    validAdminAddr,
				"multi_address": "192.168.1.1:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			labels := tt.cfg.Labels()
			assert.Equal(t, tt.expected, labels)
		})
	}
}

func TestNodeCfg_IsBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cfg      NodeCfg
		expected bool
	}{
		{
			name: "regular node (not bootstrap)",
			cfg: NodeCfg{
				MinimalNodeCfg: MinimalNodeCfg{
					Name:   validName,
					CSAKey: validCSAKey,
					NOP:    validNOP,
				},
				P2PID:     validP2PID,
				AdminAddr: validAdminAddr,
			},
			expected: false,
		},
		{
			name: "bootstrap node with multi_address",
			cfg: NodeCfg{
				MinimalNodeCfg: MinimalNodeCfg{
					Name:   validName,
					CSAKey: validCSAKey,
					NOP:    validNOP,
				},
				P2PID:        validP2PID,
				AdminAddr:    validAdminAddr,
				MultiAddress: pointer.To("192.168.1.1:8080"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.cfg.IsBootstrap()
			assert.Equal(t, tt.expected, result)
		})
	}
}
