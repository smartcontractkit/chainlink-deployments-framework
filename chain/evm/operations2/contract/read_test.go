package contract

import (
	"context"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestRead(t *testing.T) {
	t.Parallel()
	address := common.HexToAddress("0x01")
	validChainSel := uint64(5009297550715157269)

	tests := []struct {
		desc        string
		input       FunctionInput[int]
		expectedErr string
	}{
		{
			desc: "valid even input",
			input: FunctionInput[int]{
				Args: 2,
			},
		},
		{
			desc: "invalid odd input",
			input: FunctionInput[int]{
				Args: 3,
			},
			expectedErr: "odd value: 3",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()
			boundContract := newTestContract(address)

			read := NewRead(ReadParams[int, string, *testContract]{
				Name:         "test-read",
				Version:      semver.MustParse("1.0.0"),
				Description:  "Test read operation",
				ContractType: testContractType,
				Contract:     boundContract,
				CallContract: func(contract *testContract, opts *bind.CallOpts, input int) (string, error) {
					return contract.Read(opts, input)
				},
			})

			lggr, err := logger.New()
			require.NoError(t, err, "Failed to create logger")

			bundle := operations.NewBundle(
				func() context.Context { return context.Background() },
				lggr,
				operations.NewMemoryReporter(),
			)

			chain := evm.Chain{
				Selector: validChainSel,
			}

			report, err := operations.ExecuteOperation(bundle, read, chain, test.input)
			if test.expectedErr != "" {
				require.Error(t, err, "Expected ExecuteOperation error but got none")
				require.ErrorContains(t, err, test.expectedErr)
			} else {
				require.NoError(t, err, "Unexpected ExecuteOperation error")
				require.Equal(t, "even", report.Output)
			}
		})
	}
}
