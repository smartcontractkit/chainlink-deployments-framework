package analyzer

import (
	"encoding/json"
	"strings"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const aptosTestAddress = "0xe86f0e5a8b9cb6ab31b656baa83a0d2eb761b32eb31b9a9c74abb7d0cffd26fa"

func TestDescribeBatchOperations(t *testing.T) {
	t.Parallel()

	defaultProposalCtx := &DefaultProposalContext{
		AddressesByChain: deployment.AddressesByChain{
			chainsel.APTOS_TESTNET.Selector: {
				aptosTestAddress: deployment.MustTypeAndVersionFromString("TestCCIP 1.0.0"),
			},
		},
	}

	tests := []struct {
		name         string
		operations   []types.BatchOperation
		wantContains [][][]string // batches -> ops -> substrings
		wantErr      bool
	}{
		{
			name:       "Single operation",
			operations: getOperations(1),
			wantContains: [][][]string{
				{
					{"**Address:** `" + aptosTestAddress + "`", "<sub><i>address of TestCCIP 1.0.0 from aptos-testnet</i></sub>", "**Method:** `ccip_onramp::onramp::initialize`", "- `chain_selector`: `4457093679053095497`", "- `fee_aggregator`:", "- `allowlist_admin`:", "- `dest_chain_selectors`: []", "- `dest_chain_routers`: []", "- `dest_chain_allowlist_enabled`: []"},
					{"**Method:** `ccip_offramp::offramp::initialize`", "- `permissionless_execution_threshold_seconds`: `28800`", "- `source_chains_selector`: array[1]: [11155111]", "- `source_chains_is_enabled`: array[1]: [true]", "- `source_chains_is_rmn_verification_disabled`: array[1]: [false]", "- `source_chains_on_ramp`: array[1]:", "<details><summary>source_chains_selector</summary>", "[11155111]", "0x0bf3de8c5d3e8a2b34d2beeb17abfcebaf363a59"},
					{"**Method:** `ccip::rmn_remote::initialize`", "- `local_chain_selector`: `4457093679053095497`"},
					{"**Method:** `ccip_token_pool::token_pool::initialize`", "- `local_token`: `0x0000000000000000000000000000000000000000000000000000000000000003`", "- `allowlist`: array[2]:", "<details><summary>allowlist</summary>", "0x0000000000000000000000000000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000000000000000000000000000002"},
					{"**Method:** `ccip_offramp::offramp::apply_source_chain_config_updates`", "- `source_chains_selector`: array[2]:", "- `source_chains_is_enabled`: array[2]: [true, false]", "- `source_chains_is_rmn_verification_disabled`: array[2]: [true, true]", "- `source_chains_on_ramp`: array[2]:", "<details><summary>source_chains_selector</summary>", "[743186221051783445,16015286601757825753]", "0xc23071a8ae83671f37bda1dadbc745a9780f632a", "0x1c179c2c67953478966a6b460ab4873585b2f341"},
				},
			},
			wantErr: false,
		},
		{
			name:       "Multiple operations",
			operations: getOperations(2),
			wantContains: [][][]string{
				{
					{"**Address:** `" + aptosTestAddress + "`", "<sub><i>address of TestCCIP 1.0.0 from aptos-testnet</i></sub>", "**Method:** `ccip_onramp::onramp::initialize`", "- `chain_selector`: `4457093679053095497`", "- `fee_aggregator`:", "- `allowlist_admin`:", "- `dest_chain_selectors`: []", "- `dest_chain_routers`: []", "- `dest_chain_allowlist_enabled`: []"},
					{"**Method:** `ccip_offramp::offramp::initialize`", "- `permissionless_execution_threshold_seconds`: `28800`", "- `source_chains_selector`: array[1]: [11155111]", "- `source_chains_is_enabled`: array[1]: [true]", "- `source_chains_is_rmn_verification_disabled`: array[1]: [false]", "- `source_chains_on_ramp`: array[1]:", "0x0bf3de8c5d3e8a2b34d2beeb17abfcebaf363a59"},
				},
				{
					{"**Method:** `ccip::rmn_remote::initialize`", "- `local_chain_selector`: `4457093679053095497`"},
					{"**Method:** `ccip_token_pool::token_pool::initialize`", "- `local_token`: `0x0000000000000000000000000000000000000000000000000000000000000003`", "- `allowlist`: array[2]:", "0x0000000000000000000000000000000000000000000000000000000000000001", "0x0000000000000000000000000000000000000000000000000000000000000002"},
					{"**Method:** `ccip_offramp::offramp::apply_source_chain_config_updates`", "- `source_chains_selector`: array[2]:", "- `source_chains_is_enabled`: array[2]: [true, false]", "- `source_chains_is_rmn_verification_disabled`: array[2]: [true, true]", "- `source_chains_on_ramp`: array[2]:", "0xc23071a8ae83671f37bda1dadbc745a9780f632a", "0x1c179c2c67953478966a6b460ab4873585b2f341"},
				},
			},
			wantErr: false,
		},
		{
			name:       "Unknown module - non-blocking",
			operations: getBadOperations(),
			wantContains: [][][]string{
				{
					{"failed to decode Aptos transaction", "could not find function info for ccip_offramp::bad_module::initialize"},
					{"**Address:** `" + aptosTestAddress + "`", "<sub><i>address of TestCCIP 1.0.0 from aptos-testnet</i></sub>", "**Method:** `ccip::rmn_remote::initialize`", "- `local_chain_selector`: `4457093679053095497`"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := DescribeTimelockProposal(defaultProposalCtx, &mcms.TimelockProposal{Operations: tt.operations})
			if (err != nil) != tt.wantErr {
				t.Errorf("DescribeTimelockProposal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, batchContains := range tt.wantContains {
				for _, operationContains := range batchContains {
					for _, sub := range operationContains {
						if !strings.Contains(got, sub) {
							t.Errorf("missing substring %q in output:\n%s", sub, got)
						}
					}
				}
			}
		})
	}
}

func getOperations(n int) []types.BatchOperation {
	// Mock operation values were got from Aptos changesets unit tests.
	mcmsTxs := []types.Transaction{
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x13, 0xa9, 0xf1, 0xa1, 0x09, 0x36, 0x87, 0x30,
				0xf2, 0xe3, 0x55, 0xd8, 0x31, 0xba, 0x8f, 0xbf, 0x59, 0x42, 0xfb, 0x82, 0x32, 0x18, 0x63, 0xd5,
				0x5d, 0xe5, 0x4c, 0xb4, 0xeb, 0xe5, 0xd1, 0x8f, 0x13, 0xa9, 0xf1, 0xa1, 0x09, 0x36, 0x87, 0x30,
				0xf2, 0xe3, 0x55, 0xd8, 0x31, 0xba, 0x8f, 0xbf, 0x59, 0x42, 0xfb, 0x82, 0x32, 0x18, 0x63, 0xd5,
				0x5d, 0xe5, 0x4c, 0xb4, 0xeb, 0xe5, 0xd1, 0x8f, 0x00, 0x00, 0x00,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_onramp","module_name":"onramp","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x80, 0x70, 0x00, 0x00, 0x01, 0xa7, 0x36, 0xaa,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x14, 0x0b, 0xf3, 0xde, 0x8c, 0x5d,
				0x3e, 0x8a, 0x2b, 0x34, 0xd2, 0xbe, 0xeb, 0x17, 0xab, 0xfc, 0xeb, 0xaf, 0x36, 0x3a, 0x59,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_offramp","module_name":"offramp","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data:              []byte{0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d},
			AdditionalFields:  json.RawMessage(`{"package_name":"ccip","module_name":"rmn_remote","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03,
				0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x02,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_token_pool","module_name":"token_pool","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x02, 0x15, 0xa9, 0xc1, 0x33, 0xee, 0x53, 0x50, 0x0a, 0xd9, 0x1a, 0xd9, 0xc9, 0x4f, 0xba, 0x41,
				0xde, 0x02, 0x01, 0x00, 0x02, 0x01, 0x01, 0x02, 0x14, 0xc2, 0x30, 0x71, 0xa8, 0xae, 0x83, 0x67,
				0x1f, 0x37, 0xbd, 0xa1, 0xda, 0xdb, 0xc7, 0x45, 0xa9, 0x78, 0x0f, 0x63, 0x2a, 0x14, 0x1c, 0x17,
				0x9c, 0x2c, 0x67, 0x95, 0x34, 0x78, 0x96, 0x6a, 0x6b, 0x46, 0x0a, 0xb4, 0x87, 0x35, 0x85, 0xb2,
				0xf3, 0x41,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_offramp","module_name":"offramp","function":"apply_source_chain_config_updates"}`),
		},
	}
	switch n {
	case 1:
		return []types.BatchOperation{{
			ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
			Transactions:  mcmsTxs,
		}}
	case 2:
		return []types.BatchOperation{
			{
				ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
				Transactions:  mcmsTxs[:2],
			},
			{
				ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
				Transactions:  mcmsTxs[2:],
			},
		}
	default:
		return []types.BatchOperation{}
	}
}

func getBadOperations() []types.BatchOperation {
	// Mock operation with one bad module in it and one good module.
	mcmsTxs := []types.Transaction{
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data: []byte{
				0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d, 0x80, 0x70, 0x00, 0x00, 0x01, 0xa7, 0x36, 0xaa,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x00, 0x01, 0x14, 0x0b, 0xf3, 0xde, 0x8c, 0x5d,
				0x3e, 0x8a, 0x2b, 0x34, 0xd2, 0xbe, 0xeb, 0x17, 0xab, 0xfc, 0xeb, 0xaf, 0x36, 0x3a, 0x59,
			},
			AdditionalFields: json.RawMessage(`{"package_name":"ccip_offramp","module_name":"bad_module","function":"initialize"}`),
		},
		{
			OperationMetadata: types.OperationMetadata{},
			To:                aptosTestAddress,
			Data:              []byte{0x49, 0x42, 0x99, 0x1e, 0x16, 0xc7, 0xda, 0x3d},
			AdditionalFields:  json.RawMessage(`{"package_name":"ccip","module_name":"rmn_remote","function":"initialize"}`),
		},
	}

	return []types.BatchOperation{{
		ChainSelector: types.ChainSelector(chainsel.APTOS_TESTNET.Selector),
		Transactions:  mcmsTxs,
	}}
}
