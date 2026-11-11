package layout

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	"github.com/smartcontractkit/ccip-owner-contracts/gethwrappers"

	"github.com/smartcontractkit/chainlink-testing-framework/framework/evm_storage"
)

//go:embed mcms_layout.json
var MCMSLayout string

// ChangeAddressSlot replaces address slot with a new public key (address)
func ChangeAddressSlot(lggr logger.Logger, layoutData string, url string, layoutField string, contractAddr, address string) error {
	return ChangeAddressSlotContext(context.Background(), lggr, layoutData, url, layoutField, contractAddr, address)
}

// ChangeAddressSlotContext replaces an address slot on the fork while honoring
// the caller's cancellation and deadline.
func ChangeAddressSlotContext(ctx context.Context, lggr logger.Logger, layoutData string, url string, layoutField string, contractAddr, address string) error {
	lggr.Infow("Changing address slot", "URL", url, "ContractAddr", contractAddr, "Address", address)

	var layout evm_storage.StorageLayout
	err := json.Unmarshal([]byte(layoutData), &layout)
	if err != nil {
		return fmt.Errorf("failed to unmarshal storage layout: %w", err)
	}

	slot := layout.MustSlot(layoutField)
	data := evm_storage.MustEncodeStorageSlot("address", common.HexToAddress(address))
	lggr.Infow("Setting data to slot", "Slot", slot, "Data", data)
	client, err := gethrpc.DialContext(ctx, url)
	if err != nil {
		return fmt.Errorf("connecting for storage mutation: %w", err)
	}
	defer client.Close()
	var result any
	if err := client.CallContext(ctx, &result, "anvil_setStorageAt", contractAddr, slot, data); err != nil {
		return fmt.Errorf("could not set storage slot: %w", err)
	}

	return nil
}

// SetMCMSigner installs the test signer configuration used by legacy fork
// execution. The original owner is restored; the configuration stays installed.
func SetMCMSigner(ctx context.Context, lggr logger.Logger, layoutData string, privateKeyHex, newOwnerAddr, signerAddr, rpcURL string, cID string, mcmsAddr string) error {
	return changeMCMSConfig(ctx, lggr, layoutData, privateKeyHex, newOwnerAddr, rpcURL, cID, mcmsAddr, testSignerConfig(signerAddr), nil)
}

// SetTemporaryMCMSigner installs a test signer and returns a callback restoring
// the exact original configuration without clearing the accepted root. Call it
// after setRoot and before execute: execute authenticates merkle proofs against
// the stored root and does not consult the current signer configuration.
func SetTemporaryMCMSigner(ctx context.Context, lggr logger.Logger, layoutData string, privateKeyHex, newOwnerAddr, signerAddr, rpcURL string, cID string, mcmsAddr string) (func(context.Context) error, error) {
	var original gethwrappers.ManyChainMultiSigConfig
	if err := changeMCMSConfig(ctx, lggr, layoutData, privateKeyHex, newOwnerAddr, rpcURL, cID, mcmsAddr, testSignerConfig(signerAddr), &original); err != nil {
		return nil, err
	}

	return func(restoreCtx context.Context) error {
		return changeMCMSConfig(restoreCtx, lggr, layoutData, privateKeyHex, newOwnerAddr, rpcURL, cID, mcmsAddr, original, nil)
	}, nil
}

func testSignerConfig(address string) gethwrappers.ManyChainMultiSigConfig {
	return gethwrappers.ManyChainMultiSigConfig{
		Signers:      []gethwrappers.ManyChainMultiSigSigner{{Addr: common.HexToAddress(address)}},
		GroupQuorums: [32]uint8{1},
	}
}

// changeMCMSConfig temporarily owns the fork contract solely to update config.
// clearRoot=false is essential: restoration follows root authentication.
func changeMCMSConfig(ctx context.Context, lggr logger.Logger, layoutData, privateKeyHex, newOwnerAddr, rpcURL, cID, mcmsAddr string, desired gethwrappers.ManyChainMultiSigConfig, original *gethwrappers.ManyChainMultiSigConfig) (err error) {
	mcmAddress := common.HexToAddress(mcmsAddr)

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
	}
	defer client.Close()
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	chainID, ok := new(big.Int).SetString(cID, 10)
	if !ok || chainID.Sign() <= 0 {
		return fmt.Errorf("invalid chain ID: %s", cID)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}
	auth.Context = ctx

	contract, err := gethwrappers.NewManyChainMultiSig(mcmAddress, client)
	if err != nil {
		return fmt.Errorf("failed to create contract wrapper from address %s: %w", mcmAddress, err)
	}

	origOwnerAddr, err := contract.Owner(&bind.CallOpts{Context: ctx})
	if err != nil {
		return fmt.Errorf("failed to get mcm owner: %w", err)
	}
	lggr.Infow("mcm original owner", "mcm address", mcmAddress, "owner", origOwnerAddr)

	err = ChangeAddressSlotContext(ctx, lggr, layoutData, rpcURL, "_owner", mcmsAddr, newOwnerAddr)
	if err != nil {
		return fmt.Errorf("could not change address slot: %w", err)
	}
	lggr.Infow("changed mcm owner", "mcm address", mcmAddress, "new owner", newOwnerAddr)

	defer func() {
		// Only cleanup of this temporary owner slot may outlive cancellation.
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		cerr := ChangeAddressSlotContext(cleanupCtx, lggr, layoutData, rpcURL, "_owner", mcmsAddr, origOwnerAddr.Hex())
		if cerr != nil {
			lggr.Errorw("failed to restore the mcm owner", "mcm address", mcmAddress, "orig owner", origOwnerAddr.Hex())
			err = errors.Join(err, fmt.Errorf("restoring MCMS owner: %w", cerr))
		} else {
			lggr.Infow("restored mcm owner", "mcm address", mcmAddress, "orig owner", origOwnerAddr.Hex())
		}
	}()

	signers := make([]common.Address, len(desired.Signers))
	groups := make([]uint8, len(desired.Signers))
	for i, signer := range desired.Signers {
		signers[i], groups[i] = signer.Addr, signer.Group
	}

	cfg, err := contract.GetConfig(&bind.CallOpts{Context: ctx, From: common.HexToAddress(newOwnerAddr)})
	if err != nil {
		return fmt.Errorf("failed to get MCMS config: %w", err)
	}
	if original != nil {
		*original = cfg
	}
	lggr.Infof("Current signers: %+v", cfg.Signers)

	tx, err := contract.SetConfig(auth, signers, groups, desired.GroupQuorums, desired.GroupParents, false)
	if err != nil {
		return fmt.Errorf("failed to set MCMS config: %w", err)
	}
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return fmt.Errorf("failed to confirm MCMS config transaction: %w", err)
	}
	if receipt.Status != gethtypes.ReceiptStatusSuccessful {
		return fmt.Errorf("MCMS config transaction %s reverted", tx.Hash())
	}
	cfg, err = contract.GetConfig(&bind.CallOpts{Context: ctx, From: common.HexToAddress(newOwnerAddr)})
	if err != nil {
		return fmt.Errorf("failed to get MCMS config: %w", err)
	}
	lggr.Infof("New signers: %+v", cfg.Signers)

	return nil
}
