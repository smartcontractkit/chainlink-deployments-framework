package contract

import (
	"context"
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/smartcontractkit/chainlink-evm/gethwrappers/workflow/generated/workflow_registry_wrapper_v2"
	mcms_types "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	contractmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/operations2/contract/mocks"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func cancelledContext(c context.Context) context.Context {
	ctx, cancel := context.WithCancel(c)
	cancel()

	return ctx
}

func TestWriteOutput_Executed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc     string
		output   WriteOutput
		expected bool
	}{
		{
			desc: "not executed",
			output: WriteOutput{
				ExecInfo: nil,
			},
			expected: false,
		},
		{
			desc: "executed",
			output: WriteOutput{
				ExecInfo: &ExecInfo{
					Hash: "0xabc123",
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			result := test.output.Executed()
			require.Equal(t, test.expected, result)
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	address := common.HexToAddress("0x01")
	validChainSel := uint64(5009297550715157269)

	contractABI := `[{
		"inputs": [{"name": "value", "type": "uint256"}],
		"name": "InvalidValue",
		"type": "error"
	}]`

	tests := []struct {
		desc            string
		input           FunctionInput[int]
		deployerAddress common.Address
		expectedErr     string
	}{
		{
			desc: "args validation failure",
			input: FunctionInput[int]{
				Args: 3,
			},
			expectedErr: "invalid args for test-write: input must be even",
		},
		{
			desc: "revert from contract",
			input: FunctionInput[int]{
				Args: 10,
			},
			deployerAddress: OwnerAddress,
			expectedErr:     "due to error -`InvalidValue` args [1]: 6072742c0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			desc: "called by owner",
			input: FunctionInput[int]{
				Args: 2,
			},
			deployerAddress: OwnerAddress,
		},
		{
			desc: "not called by owner",
			input: FunctionInput[int]{
				Args: 2,
			},
			deployerAddress: common.HexToAddress("0x03"),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			boundContract := newTestContract(address)

			write := NewWrite(WriteParams[int, *testContract]{
				Name:            "test-write",
				Version:         semver.MustParse("1.0.0"),
				Description:     "Test write operation",
				ContractType:    testContractType,
				ContractABI:     contractABI,
				Contract:        boundContract,
				IsAllowedCaller: OnlyOwner[*testContract, int],
				Validate: func(input int) error {
					if input%2 != 0 {
						return errors.New("input must be even")
					}

					return nil
				},
				CallContract: func(contract *testContract, opts *bind.TransactOpts, input int) (*types.Transaction, error) {
					return contract.Write(opts, input)
				},
			})

			lggr, err := logger.New()
			require.NoError(t, err, "Failed to create logger")

			bundle := operations.NewBundle(
				func() context.Context { return context.Background() },
				lggr,
				operations.NewMemoryReporter(),
			)

			var confirmed bool
			chain := evm.Chain{
				Selector: validChainSel,
				DeployerKey: &bind.TransactOpts{
					From: test.deployerAddress,
				},
				Confirm: func(tx *types.Transaction) (uint64, error) {
					confirmed = true
					return 1, nil
				},
			}

			report, err := operations.ExecuteOperation(bundle, write, chain, test.input)
			if test.expectedErr != "" {
				require.Error(t, err, "Expected ExecuteOperation error but got none")
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err, "Unexpected ExecuteOperation error")
				if test.deployerAddress == OwnerAddress {
					require.True(t, confirmed, "Expected transaction to be confirmed when called by owner")
					require.True(t, report.Output.Executed(), "Expected Executed to be true when called by owner")
				} else {
					require.False(t, confirmed, "Expected transaction to not be confirmed when not called by owner")
					require.False(t, report.Output.Executed(), "Expected Executed to be false when not called by owner")
				}
				require.Equal(t, validChainSel, report.Output.ChainSelector, "Unexpected ChainSelector in output")
				require.Equal(t, []byte{0xDE, 0xAD, 0xBE, 0xEF}, report.Output.Tx.Data, "Unexpected tx data in output")
				require.Equal(t, address.Hex(), report.Output.Tx.To, "Unexpected to address in output")
				require.Equal(t, string(testContractType), report.Output.Tx.ContractType, "Unexpected ContractType in output")
			}
		})
	}
}

func TestHasRole(t *testing.T) {
	t.Parallel()

	role := [32]byte{0x01, 0x02, 0x03}
	account := common.HexToAddress("0x1234")
	contractAddress := common.HexToAddress("0xabcd")

	tests := []struct {
		desc        string
		opts        *bind.CallOpts
		setupMock   func(*contractmocks.MockAccessControlContract, *bind.CallOpts)
		wantAllowed bool
		wantErr     string
	}{
		{
			desc: "returns true when contract reports account has role",
			setupMock: func(contract *contractmocks.MockAccessControlContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					HasRole(opts, role, account).
					Return(true, nil).
					Once()
			},
			wantAllowed: true,
		},
		{
			desc: "returns false when contract reports account does not have role",
			setupMock: func(contract *contractmocks.MockAccessControlContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					HasRole(opts, role, account).
					Return(false, nil).
					Once()
			},
			wantAllowed: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			contract := contractmocks.NewMockAccessControlContract(t)
			test.setupMock(contract, test.opts)

			allowed, err := HasRole(contract, test.opts, role, account)
			if test.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, test.wantErr)
				require.False(t, allowed)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.wantAllowed, allowed)
			}
		})
	}
}

func TestIsAuthorizedCaller(t *testing.T) {
	t.Parallel()

	account := common.HexToAddress("0x1234")
	contractAddress := common.HexToAddress("0xabcd")

	tests := []struct {
		desc           string
		opts           *bind.CallOpts
		setupMock      func(*contractmocks.MockAuthorizedCallersContract, *bind.CallOpts)
		wantAuthorized bool
	}{
		{
			desc: "returns true when caller is authorized",
			setupMock: func(contract *contractmocks.MockAuthorizedCallersContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					GetAllAuthorizedCallers(opts).
					Return([]common.Address{account}, nil).
					Once()
			},
			wantAuthorized: true,
		},
		{
			desc: "returns false when caller is not authorized",
			setupMock: func(contract *contractmocks.MockAuthorizedCallersContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					GetAllAuthorizedCallers(opts).
					Return([]common.Address{}, nil).
					Once()
			},
			wantAuthorized: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			contract := contractmocks.NewMockAuthorizedCallersContract(t)
			test.setupMock(contract, test.opts)

			allowed, err := IsAuthorizedCaller(contract, test.opts, account)
			require.NoError(t, err)
			require.Equal(t, test.wantAuthorized, allowed)
		})
	}
}

func TestIsWorkflowsOwner(t *testing.T) {
	t.Parallel()

	caller := common.HexToAddress("0x1234")
	other := common.HexToAddress("0x5678")
	contractAddress := common.HexToAddress("0xabcd")
	workflowId1 := [32]byte{0x01}
	workflowId2 := [32]byte{0x02}
	callErr := errors.New("get workflow failed")

	tests := []struct {
		desc        string
		workflowIds [][32]byte
		setupMock   func(*contractmocks.MockWorkflowRegistryContract, *bind.CallOpts)
		wantAllowed bool
		wantErr     string
	}{
		{
			desc:        "returns true when caller owns all workflows",
			workflowIds: [][32]byte{workflowId1, workflowId2},
			setupMock: func(contract *contractmocks.MockWorkflowRegistryContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					GetWorkflowById(opts, workflowId1).
					Return(workflow_registry_wrapper_v2.WorkflowRegistryWorkflowMetadataView{Owner: caller}, nil).
					Once()
				contract.EXPECT().
					GetWorkflowById(opts, workflowId2).
					Return(workflow_registry_wrapper_v2.WorkflowRegistryWorkflowMetadataView{Owner: caller}, nil).
					Once()
			},
			wantAllowed: true,
		},
		{
			desc:        "returns false when caller does not own a workflow",
			workflowIds: [][32]byte{workflowId1, workflowId2},
			setupMock: func(contract *contractmocks.MockWorkflowRegistryContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					GetWorkflowById(opts, workflowId1).
					Return(workflow_registry_wrapper_v2.WorkflowRegistryWorkflowMetadataView{Owner: caller}, nil).
					Once()
				contract.EXPECT().
					GetWorkflowById(opts, workflowId2).
					Return(workflow_registry_wrapper_v2.WorkflowRegistryWorkflowMetadataView{Owner: other}, nil).
					Once()
			},
			wantAllowed: false,
		},
		{
			desc:        "returns error when workflow lookup fails",
			workflowIds: [][32]byte{workflowId1},
			setupMock: func(contract *contractmocks.MockWorkflowRegistryContract, opts *bind.CallOpts) {
				contract.EXPECT().
					Address().
					Return(contractAddress).
					Once()
				contract.EXPECT().
					GetWorkflowById(opts, workflowId1).
					Return(workflow_registry_wrapper_v2.WorkflowRegistryWorkflowMetadataView{}, callErr).
					Once()
			},
			wantErr: "failed to check workflow ownership",
		},
		{
			desc:        "returns error for empty workflow list",
			workflowIds: nil,
			setupMock:   func(contract *contractmocks.MockWorkflowRegistryContract, opts *bind.CallOpts) {},
			wantErr:     "no workflow IDs provided",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			opts := &bind.CallOpts{}
			contract := contractmocks.NewMockWorkflowRegistryContract(t)
			test.setupMock(contract, opts)

			allowed, err := IsWorkflowsOwner(contract, opts, caller, test.workflowIds)
			if test.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, test.wantErr)
				require.False(t, allowed)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.wantAllowed, allowed)
		})
	}
}

func TestRetryContractCall(t *testing.T) {
	t.Parallel()

	contractAddress := common.HexToAddress("0xabcd")
	tests := []struct {
		desc       string
		opts       *bind.CallOpts
		check      func() (bool, error)
		want       bool
		wantErr    string
		wantChecks int
	}{
		{
			desc: "returns successful check result",
			check: func() (bool, error) {
				return true, nil
			},
			want:       true,
			wantChecks: 1,
		},
		{
			desc: "returns non retryable errors immediately",
			check: func() (bool, error) {
				return false, errors.New("rpc unavailable")
			},
			wantErr:    "failed to check role of 0x000000000000000000000000000000000000ABcD: rpc unavailable",
			wantChecks: 1,
		},
		{
			desc: "returns context error while retrying empty response errors",
			opts: &bind.CallOpts{
				Context: cancelledContext(t.Context()),
			},
			check: func() (bool, error) {
				return false, errors.New("attempting to unmarshal an empty string")
			},
			wantErr:    "context cancelled while waiting for role check of 0x000000000000000000000000000000000000ABcD: context canceled",
			wantChecks: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			checks := 0
			got, err := RetryContractCall(
				test.opts,
				"role check",
				"check role",
				contractAddress,
				func() (bool, error) {
					checks++

					return test.check()
				},
			)
			if test.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.want, got)
			}
			require.Equal(t, test.wantChecks, checks)
		})
	}
}

func TestBatchOperationFromWrites(t *testing.T) {
	t.Parallel()
	tests := []struct {
		desc        string
		outputs     []WriteOutput
		expected    mcms_types.BatchOperation
		expectedErr string
	}{
		{
			desc: "single output",
			outputs: []WriteOutput{
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
			expected: mcms_types.BatchOperation{
				ChainSelector: 5009297550715157269,
				Transactions: []mcms_types.Transaction{
					{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
		},
		{
			desc: "multiple outputs same chain",
			outputs: []WriteOutput{
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x02").Hex(),
						Data:             common.Hex2Bytes("0xcafebabe"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
			expected: mcms_types.BatchOperation{
				ChainSelector: 5009297550715157269,
				Transactions: []mcms_types.Transaction{
					{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
					{
						To:               common.HexToAddress("0x02").Hex(),
						Data:             common.Hex2Bytes("0xcafebabe"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
		},
		{
			desc: "multiple outputs different chains",
			outputs: []WriteOutput{
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
				{
					ChainSelector: 4340886533089894000,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x02").Hex(),
						Data:             common.Hex2Bytes("0xcafebabe"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
			expected:    mcms_types.BatchOperation{},
			expectedErr: "writes target multiple chains",
		},
		{
			desc:     "no outputs",
			outputs:  []WriteOutput{},
			expected: mcms_types.BatchOperation{},
		},
		{
			desc: "all executed outputs",
			outputs: []WriteOutput{
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
					ExecInfo: &ExecInfo{
						Hash: "0xabc123",
					},
				},
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x02").Hex(),
						Data:             common.Hex2Bytes("0xcafebabe"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
					ExecInfo: &ExecInfo{
						Hash: "0xdef456",
					},
				},
			},
			expected: mcms_types.BatchOperation{},
		},
		{
			desc: "executed prefix then unexecuted same chain",
			outputs: []WriteOutput{
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x01").Hex(),
						Data:             common.Hex2Bytes("0xdeadbeef"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
					ExecInfo: &ExecInfo{
						Hash: "0xabc123",
					},
				},
				{
					ChainSelector: 5009297550715157269,
					Tx: mcms_types.Transaction{
						To:               common.HexToAddress("0x02").Hex(),
						Data:             common.Hex2Bytes("0xcafebabe"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
			expected: mcms_types.BatchOperation{
				ChainSelector: 5009297550715157269,
				Transactions: []mcms_types.Transaction{
					{
						To:               common.HexToAddress("0x02").Hex(),
						Data:             common.Hex2Bytes("0xcafebabe"),
						AdditionalFields: []byte{0x7B, 0x7D}, // "{}" in bytes
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			batchOp, err := NewBatchOperationFromWrites(test.outputs)
			if test.expectedErr != "" {
				require.Error(t, err, "Expected error from NewBatchOperationFromWrites")
				require.ErrorContains(t, err, test.expectedErr, "Unexpected error message")

				return
			}
			require.NoError(t, err, "Unexpected error from NewBatchOperationFromWrites")
			require.Equal(t, test.expected, batchOp, "Unexpected BatchOperation result")
		})
	}
}
