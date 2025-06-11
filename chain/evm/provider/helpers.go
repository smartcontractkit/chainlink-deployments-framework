package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ContractCaller is an interface that defines the CallContract method. This is copied from the
// go-ethereum package method to limit the scope of dependencies provided to the functions.
type ContractCaller interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// getErrorReasonFromTx retrieves the error reason from a transaction by simulating the call
// using the CallContract method. If the transaction reverts, it attempts to extract the
// error reason from the returned error.
func getErrorReasonFromTx(
	ctx context.Context,
	caller ContractCaller,
	from common.Address,
	tx *types.Transaction,
	receipt *types.Receipt,
) (string, error) {
	call := ethereum.CallMsg{
		From:     from,
		To:       tx.To(),
		Data:     tx.Data(),
		Value:    tx.Value(),
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice(),
	}

	if _, err := caller.CallContract(ctx, call, receipt.BlockNumber); err != nil {
		reason, perr := getJSONErrorData(err)

		// If the reason exists and we had no issues parsing it, we return it
		if perr == nil {
			return reason, nil
		}

		// If we get no information from parsing the error, we return the original error from
		// CallContract
		if reason == "" {
			return err.Error(), nil
		}
	}

	return "", fmt.Errorf("tx %s reverted with no reason", tx.Hash().Hex())
}

// getJSONErrorData extracts the error data from a JSON Error.
func getJSONErrorData(err error) (string, error) {
	if err == nil {
		return "", errors.New("cannot parse nil error")
	}

	// Define a custom interface that matches the structure of the JSON error because it is a
	// private type in go-ethereum.
	//
	// https://github.com/ethereum/go-ethereum/blob/0983cd789ee1905aedaed96f72793e5af8466f34/rpc/json.go#L140
	type jsonError interface {
		Error() string
		ErrorCode() int
		ErrorData() any
	}

	var jerr jsonError
	ok := errors.As(err, &jerr)
	if !ok {
		return "", fmt.Errorf("error must be of type jsonError: %w", err)
	}

	data := fmt.Sprintf("%s", jerr.ErrorData())
	if data == "" && strings.Contains(jerr.Error(), "missing trie node") {
		return "", errors.New("missing trie node, likely due to not using an archive node")
	}

	return data, nil
}
