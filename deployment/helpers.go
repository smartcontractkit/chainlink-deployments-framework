package deployment

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
)

// SimTransactOpts is useful to generate just the calldata for a given gethwrapper method.
func SimTransactOpts() *bind.TransactOpts {
	return &bind.TransactOpts{Signer: func(address common.Address, transaction *types.Transaction) (*types.Transaction, error) {
		return transaction, nil
	}, From: common.HexToAddress("0x0"), NoSend: true, GasLimit: 1_000_000}
}

// todo: remove when Chainlink is migrated
var ChainInfo = utils.ChainInfo

func parseErrorFromABI(errorString string, contractABI string) (string, error) {
	errorString = strings.TrimPrefix(errorString, "Reverted ")
	errorString = strings.TrimPrefix(errorString, "0x")

	data, err := hex.DecodeString(errorString)
	if err != nil {
		return "", fmt.Errorf("error decoding error string: %w", err)
	}

	v, err := abi.UnpackRevert(data)
	if err == nil {
		return fmt.Sprintf("error - `%s`", v), nil
	}

	parsedAbi, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return "", fmt.Errorf("error loading ABI: %w", err)
	}

	for errorName, abiError := range parsedAbi.Errors {
		if bytes.Equal(data[:4], abiError.ID.Bytes()[:4]) {
			// Found a matching error
			v, err3 := abiError.Unpack(data)
			if err3 != nil {
				return "", fmt.Errorf("error unpacking data: %w", err3)
			}

			return fmt.Sprintf("error -`%v` args %v", errorName, v), nil
		}
	}

	return "", errors.New("error not found in ABI")
}

// DecodeErr decodes an error from a contract call using the contract's ABI.
// If the error is not decodable, it returns the original error.
func DecodeErr(encodedABI string, err error) error {
	if err == nil {
		return nil
	}
	//revive:disable
	var d rpc.DataError
	ok := errors.As(err, &d)
	if ok {
		encErr, ok := d.ErrorData().(string)
		if !ok {
			return fmt.Errorf("error without error data: %s", d.Error())
		}
		errStr, parseErr := parseErrorFromABI(encErr, encodedABI)
		if parseErr != nil {
			return fmt.Errorf("failed to decode error '%s' with abi: %w", encErr, parseErr)
		}

		return fmt.Errorf("contract error: %s", errStr)
	}

	return fmt.Errorf("cannot decode error with abi: %w", err)
}

// ContractDeploy represents the result of an EVM contract deployment
// via an abigen Go binding. It contains all the return values
// as they are useful in different ways.
type ContractDeploy[C any] struct {
	Address  common.Address     // We leave this incase a Go binding doesn't have Address()
	Contract C                  // Expected to be a Go binding
	Tx       *types.Transaction // Incase the caller needs for example tx hash info for
	Tv       TypeAndVersion
	Err      error
}

// DeployContract deploys an EVM contract and
// records the address in the provided address book
// if the deployment was confirmed onchain.
// Deploying and saving the address is a very common pattern
// so this helps to reduce boilerplate.
// It returns an error if the deployment failed, the tx was not
// confirmed or the address could not be saved.
func DeployContract[C any](
	lggr logger.Logger,
	chain Chain,
	addressBook AddressBook,
	deploy func(chain Chain) ContractDeploy[C],
) (*ContractDeploy[C], error) {
	contractDeploy := deploy(chain)
	if contractDeploy.Err != nil {
		lggr.Errorw("Failed to deploy contract", "chain", chain.String(), "err", contractDeploy.Err)
		return nil, contractDeploy.Err
	}
	var err error
	if !chain.IsZkSyncVM {
		_, err = chain.Confirm(contractDeploy.Tx)
		if err != nil {
			lggr.Errorw("Failed to confirm deployment", "chain", chain.String(), "Contract", contractDeploy.Tv.String(), "err", err)
			return nil, err
		}
	}
	lggr.Infow("Deployed contract", "Contract", contractDeploy.Tv.String(), "addr", contractDeploy.Address, "chain", chain.String())
	err = addressBook.Save(chain.Selector, contractDeploy.Address.String(), contractDeploy.Tv)
	if err != nil {
		lggr.Errorw("Failed to save contract address", "Contract", contractDeploy.Tv.String(), "addr", contractDeploy.Address, "chain", chain.String(), "err", err)
		return nil, err
	}

	return &contractDeploy, nil
}

// IsValidChainSelector checks if the chain selector is valid.
func IsValidChainSelector(cs uint64) error {
	if cs == 0 {
		return errors.New("chain selector must be set")
	}
	_, err := chain_selectors.GetSelectorFamily(cs)
	if err != nil {
		return err
	}

	return nil
}
