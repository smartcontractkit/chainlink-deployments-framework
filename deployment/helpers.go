package deployment

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	pkgErrors "github.com/pkg/errors"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

func ChainInfo(cs uint64) (chain_selectors.ChainDetails, error) {
	id, err := chain_selectors.GetChainIDFromSelector(cs)
	if err != nil {
		return chain_selectors.ChainDetails{}, err
	}
	family, err := chain_selectors.GetSelectorFamily(cs)
	if err != nil {
		return chain_selectors.ChainDetails{}, err
	}
	info, err := chain_selectors.GetChainDetailsByChainIDAndFamily(id, family)
	if err != nil {
		return chain_selectors.ChainDetails{}, err
	}
	return info, nil
}

func parseErrorFromABI(errorString string, contractABI string) (string, error) {
	errorString = strings.TrimPrefix(errorString, "Reverted ")
	errorString = strings.TrimPrefix(errorString, "0x")

	data, err := hex.DecodeString(errorString)
	if err != nil {
		return "", pkgErrors.Wrap(err, "error decoding error string")
	}

	v, err := abi.UnpackRevert(data)
	if err == nil {
		return fmt.Sprintf("error - `%s`", v), nil
	}

	parsedAbi, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return "", pkgErrors.Wrap(err, "error loading ABI")
	}

	for errorName, abiError := range parsedAbi.Errors {
		if bytes.Equal(data[:4], abiError.ID.Bytes()[:4]) {
			// Found a matching error
			v, err3 := abiError.Unpack(data)
			if err3 != nil {
				return "", pkgErrors.Wrap(err3, "error unpacking data")
			}
			return fmt.Sprintf("error -`%v` args %v", errorName, v), nil
		}
	}
	return "", pkgErrors.New("error not found in ABI")
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
	_, err := chain.Confirm(contractDeploy.Tx)
	if err != nil {
		lggr.Errorw("Failed to confirm deployment", "chain", chain.String(), "Contract", contractDeploy.Tv.String(), "err", err)
		return nil, err
	}
	lggr.Infow("Deployed contract", "Contract", contractDeploy.Tv.String(), "addr", contractDeploy.Address, "chain", chain.String())
	err = addressBook.Save(chain.Selector, contractDeploy.Address.String(), contractDeploy.Tv)
	if err != nil {
		lggr.Errorw("Failed to save contract address", "Contract", contractDeploy.Tv.String(), "addr", contractDeploy.Address, "chain", chain.String(), "err", err)
		return nil, err
	}
	return &contractDeploy, nil
}
