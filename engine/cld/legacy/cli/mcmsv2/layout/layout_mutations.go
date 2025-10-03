package layout

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/evm_storage"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/rpc"
)

//go:embed mcms_layout.json
var MCMSLayout string

// ChangeAddressSlot replaces address slot with a new public key (address)
func ChangeAddressSlot(lggr logger.Logger, layoutData string, url string, layoutField string, contractAddr, address string) error {
	lggr.Infow("Changing address slot", "URL", url, "ContractAddr", contractAddr, "Address", address)

	var layout evm_storage.StorageLayout
	err := json.Unmarshal([]byte(layoutData), &layout)
	if err != nil {
		return fmt.Errorf("failed to unmarshal storage layout: %w", err)
	}

	slot := layout.MustSlot(layoutField)
	data := evm_storage.MustEncodeStorageSlot("address", common.HexToAddress(address))
	lggr.Infow("Setting data to slot", "Slot", slot, "Data", data)
	r := rpc.New(url, nil)
	err = r.AnvilSetStorageAt([]interface{}{contractAddr, slot, data})
	if err != nil {
		return fmt.Errorf("could not set storage slot: %w", err)
	}

	return nil
}

// SetMCMSigner is using gethwrappers.NewManyChainMultiSig to change the owner of MCM contract
// and set test signer address so we can run MCM proposals with test signatures only
func SetMCMSigner(ctx context.Context, lggr logger.Logger, layoutData string, privateKeyHex, newOwnerAddr, signerAddr, rpcURL string, cID string, mcmsAddr string) error {
	lggr.Infow("Setting MCMS signer", "RPCURL", rpcURL, "MCMSContractAddress", mcmsAddr, "NewOwnerAddress", newOwnerAddr, "SignerAddress", signerAddr, "ChainID", cID)
	mcmAddress := common.HexToAddress(mcmsAddr)

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	chainID, ok := new(big.Int).SetString(cID, 10)
	if !ok {
		return fmt.Errorf("invalid chain ID: %s", cID)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	contract, err := gethwrappers.NewManyChainMultiSig(mcmAddress, client)
	if err != nil {
		return fmt.Errorf("failed to create contract wrapper from address %s: %w", mcmAddress, err)
	}

	origOwnerAddr, err := contract.Owner(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("failed to get mcm owner: %w", err)
	}
	lggr.Infow("mcm original owner", "mcm address", mcmAddress, "owner", origOwnerAddr)

	err = ChangeAddressSlot(lggr, layoutData, rpcURL, "_owner", mcmsAddr, newOwnerAddr)
	if err != nil {
		return fmt.Errorf("could not change address slot: %w", err)
	}
	lggr.Infow("changed mcm owner", "mcm address", mcmAddress, "new owner", newOwnerAddr)

	defer func() {
		cerr := ChangeAddressSlot(lggr, layoutData, rpcURL, "_owner", mcmsAddr, origOwnerAddr.Hex())
		if cerr != nil {
			lggr.Errorw("failed to restore the mcm owner", "mcm address", mcmAddress, "orig owner", origOwnerAddr.Hex())
		} else {
			lggr.Infow("restored mcm owner", "mcm address", mcmAddress, "orig owner", origOwnerAddr.Hex())
		}
	}()

	singleSigners := []common.Address{
		common.HexToAddress(signerAddr),
	}
	signerGroups := []uint8{0}

	var groupQuorums [32]uint8
	var groupParents [32]uint8
	groupQuorums[0] = 1
	groupParents[0] = 0

	cfg, err := contract.GetConfig(&bind.CallOpts{Context: context.Background(), From: common.HexToAddress(newOwnerAddr)})
	if err != nil {
		return fmt.Errorf("failed to get MCMS config: %w", err)
	}
	lggr.Infof("Current signers: %+v", cfg.Signers)

	tx, err := contract.SetConfig(auth, singleSigners, signerGroups, groupQuorums, groupParents, false)
	if err != nil {
		return fmt.Errorf("failed to set MCMS config: %w", err)
	}
	_, err = bind.WaitMined(ctx, client, tx)
	if err != nil {
		return fmt.Errorf("failed to confirm MCMS config transaction: %w", err)
	}
	cfg, err = contract.GetConfig(&bind.CallOpts{Context: context.Background(), From: common.HexToAddress(newOwnerAddr)})
	if err != nil {
		return fmt.Errorf("failed to get MCMS config: %w", err)
	}
	lggr.Infof("New signers: %+v", cfg.Signers)

	return nil
}
