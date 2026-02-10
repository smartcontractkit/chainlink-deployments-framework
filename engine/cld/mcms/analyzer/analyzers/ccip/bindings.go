package ccip

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const erc20ABIJSON = `[
	{
		"inputs": [],
		"name": "symbol",
		"outputs": [{"name": "", "type": "string"}],
		"stateMutability": "view",
		"type": "function"
	}
]`

var erc20ABI = mustParseABI(erc20ABIJSON)

func mustParseABI(abiJSON string) *abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		panic("failed to parse ABI: " + err.Error())
	}

	return &parsed
}

type erc20Caller struct {
	contract *bind.BoundContract
}

func newERC20Caller(address common.Address, backend bind.ContractCaller) *erc20Caller {
	contract := bind.NewBoundContract(address, *erc20ABI, backend, nil, nil)

	return &erc20Caller{contract: contract}
}

func (c *erc20Caller) symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}

	err := c.contract.Call(opts, &out, "symbol")
	if err != nil {
		return "", err
	}

	if len(out) == 0 {
		return "", errors.New("symbol returned no data")
	}

	s, ok := out[0].(string)
	if !ok {
		return "", fmt.Errorf("symbol: expected string, got %T", out[0])
	}

	return s, nil
}
