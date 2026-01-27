package commands

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"

	fevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const (
	etherscanTimeout = 10 * time.Second
)

type APIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ContractCreationTxResponse struct {
	APIResponse
	Result json.RawMessage `json:"result"`
}

type ContractCreationResult struct {
	ContractAddress string `json:"contractAddress"`
	ContractCreator string `json:"contractCreator"`
	TxHash          string `json:"txHash"`
}

// GetContractCreationTx finds contract creation tx by querying getcontractcreation action.
// It is a short-cut provided by most Etherscan instances.
func GetContractCreationTx(ctx context.Context, endpoint string, addressStr string, apiKey string) (string, error) {
	url := fmt.Sprintf("%s?module=contract&action=getcontractcreation&contractaddresses=%s&apikey=%s", endpoint, addressStr, apiKey)

	ctx, cancel := context.WithTimeout(ctx, etherscanTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var data ContractCreationTxResponse
	if err = decoder.Decode(&data); err != nil {
		return "", err
	}

	// Happy path, return data shows success
	if data.Status == "1" {
		var result []ContractCreationResult
		if err = json.Unmarshal(data.Result, &result); err != nil {
			return "", err
		}

		if len(result) != 1 {
			return "", fmt.Errorf("invalid contract creation tx result: %v", data)
		}

		return result[0].TxHash, nil
	}

	var errMsg string
	if err = json.Unmarshal(data.Result, &errMsg); err != nil {
		return "", err
	}
	// If API call failed due to reasons other than action unsupported, there is no fallback.
	if !strings.Contains(errMsg, "invalid Action name") {
		return "", fmt.Errorf("failed to get contract creation tx: %s", errMsg)
	}

	return GetContractCreationTxFallback(ctx, endpoint, addressStr, apiKey)
}

type AddressTxListResponse struct {
	APIResponse
	Result []struct {
		BlockNumber string `json:"blockNumber"`
		TimeStamp   string `json:"timeStamp"`
		Hash        string `json:"hash"`
	} `json:"result"`
}

// GetContractCreationTxFallback finds contract creation tx by searching tx history.
// Some Etherscan instances may not support getcontractcreation action, we can fall back to parsing transaction history.
// Example offenders are Avax Fuji Etherscan and BSC Testnet Etherscan
func GetContractCreationTxFallback(ctx context.Context, endpoint string, addressStr string, apiKey string) (string, error) {
	url := fmt.Sprintf("%s?module=account&action=txlist&address=%s&apikey=%s&startblock=0&endblock=999999999&page=1&offset=1&sort=asc", endpoint, addressStr, apiKey)

	ctx, cancel := context.WithTimeout(ctx, etherscanTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var data AddressTxListResponse
	if err = decoder.Decode(&data); err != nil {
		return "", err
	}

	return data.Result[0].Hash, nil
}

type ContractABIResponse struct {
	APIResponse
	Result string `json:"result"`
}

func IsContractVerified(ctx context.Context, endpoint string, addressStr string, apiKey string) (bool, error) {
	url := fmt.Sprintf("%s?module=contract&action=getabi&address=%s&apikey=%s", endpoint, addressStr, apiKey)

	ctx, cancel := context.WithTimeout(ctx, etherscanTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var data ContractABIResponse
	if err = decoder.Decode(&data); err != nil {
		return false, err
	}

	return data.Status == "1", nil
}

func buildContractVerifyCmd(
	ctx context.Context,
	client *ethclient.Client,
	chainID string,
	optimizerRuns uint64,
	contractAddress,
	contractCreationTx,
	contractName,
	apiKey,
	compilerVersion string,
) ([]string, error) {
	constructorArgs, err := findConstructorArgs(ctx, client, contractAddress, contractCreationTx)
	if err != nil {
		return nil, fmt.Errorf("finding constructor args: %w", err)
	}

	cmd := []string{
		"forge", "contract verify", "--watch",
		"--compiler-version", compilerVersion,
		"--optimizer-runs", strconv.FormatUint(optimizerRuns, 10),
		"--chain-id", chainID,
		"--constructor-args", constructorArgs,
		"--etherscan-api-key", apiKey,
		contractAddress,
		contractName,
	}

	return cmd, nil
}

func findConstructorArgs(ctx context.Context, cl *ethclient.Client, contractAddress, contractCreationTx string) (string, error) {
	codeBytes, err := cl.CodeAt(ctx, common.HexToAddress(contractAddress), nil)
	if err != nil {
		return "", fmt.Errorf("get contract %s code: %w", contractAddress, err)
	}
	contractDataHex := hex.EncodeToString(codeBytes)

	tx, _, err := cl.TransactionByHash(ctx, common.HexToHash(contractCreationTx))
	if err != nil {
		return "", fmt.Errorf("transaction by hash %s: %w", contractCreationTx, err)
	}
	txData := hex.EncodeToString(tx.Data())

	// contract creation tx data = some init code bytes + bytes near the end of contract data + constructor args
	// we match based on the last BytecodeOffset length portion of contract bytecode
	matchIdx := strings.LastIndex(txData, contractDataHex[len(contractDataHex)-BytecodeOffset:])
	if matchIdx < 0 {
		return "", errors.New("failed to find end of contract bytecode in tx data")
	}

	argsIdx := matchIdx + BytecodeOffset

	return txData[argsIdx:], nil
}

func verifyContract(
	ctx context.Context,
	lggr logger.Logger,
	domain domain.Domain,
	chainselector uint64,
	environmentStr,
	contractDirectory,
	commit,
	contractAddress,
	compilerVersion string,
	optimizerRuns uint64,
	contractName string,
) error {
	networks, err := config.LoadNetworks(environmentStr, domain, lggr)
	if err != nil {
		return fmt.Errorf("failed to load networks from env %s: %w", environmentStr, err)
	}

	network, err := networks.NetworkBySelector(chainselector)
	if err != nil {
		return fmt.Errorf("network configuration not found with chain selector %d: %w",
			chainselector, err,
		)
	}

	chainID, err := network.ChainID()
	if err != nil {
		return fmt.Errorf("chain ID does not exist for network with chain selector %d: %w",
			chainselector, err,
		)
	}

	explorer := network.BlockExplorer

	lggr.Infof("Using chainID %s via %s", chainID, explorer.URL)

	ec, err := ethclient.Dial(network.RPCs[0].HTTPURL)
	if err != nil {
		return fmt.Errorf("failed to dial eth client: %w", err)
	}
	verified, err := IsContractVerified(ctx, explorer.URL, contractAddress, explorer.APIKey)
	if err != nil {
		return fmt.Errorf("failed to check if contract is verified: %w", err)
	}
	if verified {
		lggr.Infof("contract %s is already verified", contractAddress)
		return nil
	}
	contractCreationTx, err := GetContractCreationTx(ctx, explorer.URL, contractAddress, explorer.APIKey)
	if err != nil {
		return fmt.Errorf("failed to get contract creation tx: %w", err)
	}
	if contractCreationTx == "" {
		return errors.New("contract creation TX not found")
	}
	lggr.Infof("The contract was deployed at tx %s", contractCreationTx)
	cmdStrs, err := buildContractVerifyCmd(
		ctx,
		ec,
		chainID,
		optimizerRuns,
		contractAddress,
		contractCreationTx,
		contractName,
		explorer.APIKey,
		compilerVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to build contract verify cmd: %w", err)
	}

	lggr.Infof("Assembled verification command")
	var logCmd []string
	// Mask (redact) any sensitive API key value.
	for i := 0; i < len(cmdStrs); i++ {
		if cmdStrs[i] == "--etherscan-api-key" && i+1 < len(cmdStrs) {
			logCmd = append(logCmd, cmdStrs[i])
			logCmd = append(logCmd, "<REDACTED>")
			i++ // Skip the actual key value

			continue
		}
		logCmd = append(logCmd, cmdStrs[i])
	}
	lggr.Infof("Verification command assembled (API key redacted): %v", logCmd)

	// Keep this close to exec to avoid out of band checkouts changing it.
	if err = checkoutCommit(contractDirectory, commit); err != nil {
		return fmt.Errorf("failed to checkout commit: %w", err)
	}

	// Due to the bug in the Forge tool, to apply custom EtherscanAPI
	// keys and endpoints, need to modify foundry.toml in the contracts' dir.
	// https://github.com/foundry-rs/foundry/issues/7466
	closeFn, err := appendEtherscanInfoToFoundryToml(contractDirectory, chainID, network.BlockExplorer)
	if err != nil {
		return fmt.Errorf("failed to modify foundry.toml: %w", err)
	}

	defer func() {
		if err = closeFn(); err != nil {
			lggr.Errorf("failed to close appendEtherscanInfoToFoundryToml cmd %s", err)
		}
	}()

	forgeCmd := exec.CommandContext(ctx, cmdStrs[0], cmdStrs[1:]...) //nolint:gosec // This is passed on from the cobra command flags
	forgeCmd.Dir = contractDirectory

	lggr.Infof("Executing...")
	var outBuffer, errBuffer bytes.Buffer
	forgeCmd.Stdout = &outBuffer
	forgeCmd.Stderr = &errBuffer
	err = forgeCmd.Run()
	lggr.Infof(outBuffer.String())
	lggr.Infof(errBuffer.String())
	if err != nil {
		return fmt.Errorf("failed to run forge cmd %s. Are you sure the optimizer-runs, compiler version, commit hash and name are correct", err.Error())
	}

	return nil
}

func checkoutCommit(dirPath, commitHash string) error {
	cmd := exec.CommandContext(context.TODO(), "git", "-C", dirPath, "checkout", commitHash)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run() // Run the command and return any error encountered
}

// appendEtherscanInfoToFoundryToml Adds [etherscan] section to the foundry.toml in the dirPath and network with
// defined apiKey, apiEndpoint and ChainID. Currently, it's the only way to make Forge
// tool works with new/unknown chains.
func appendEtherscanInfoToFoundryToml(
	dirPath, chainID string, explorer cfgnet.BlockExplorer,
) (func() error, error) {
	foundryToml := filepath.Join(dirPath, "foundry.toml")
	foundryBackupToml := filepath.Join(dirPath, "foundry.backup.toml")

	// copy foundry.toml file, to the backup file, for recovery initial state after execution.
	cmd := exec.CommandContext(context.TODO(), "cp", foundryToml, foundryBackupToml)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to copy foundry.toml to foundry.backup.toml: %w", err)
	}

	file, err := os.OpenFile(dirPath+"/foundry.toml", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open foundry.toml: %w", err)
	}

	if _, err = file.WriteString("[etherscan]\n"); err != nil {
		return nil, fmt.Errorf("failed to write header: %w", err)
	}

	foundryConfigTpl := "chain = { key = \"%s\", url = \"%s\", chain = \"%s\" }"

	if _, err = fmt.Fprintf(
		file, foundryConfigTpl, explorer.APIKey, explorer.URL, chainID,
	); err != nil {
		return nil, fmt.Errorf("failed to write chain info: %w", err)
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close foundry.toml file: %w", err)
	}

	return func() error {
		// remove changed file
		cmdFileRm := exec.CommandContext(context.TODO(), "rm", foundryToml)
		cmdFileRm.Stdout = os.Stdout
		cmdFileRm.Stderr = os.Stderr
		if err := cmdFileRm.Run(); err != nil {
			return fmt.Errorf("failed to remove modified foundry.toml: %w", err)
		}

		// recover backup file
		cmdFileMv := exec.CommandContext(context.TODO(), "mv", foundryBackupToml, foundryToml)
		cmdFileMv.Stdout = os.Stdout
		cmdFileMv.Stderr = os.Stderr
		if err := cmdFileMv.Run(); err != nil {
			return fmt.Errorf("failed to rename foundry.backup.toml: %w", err)
		}

		return nil
	}, nil
}

func readContractsList(tomlPth string) (ContractsToVerify, error) {
	data, err := os.ReadFile(tomlPth)
	if err != nil {
		return ContractsToVerify{}, fmt.Errorf("failed to read file %w", err)
	}

	var contracts ContractsToVerify
	err = toml.Unmarshal(data, &contracts)
	if err != nil {
		return ContractsToVerify{}, fmt.Errorf("failed to unmarshal toml: %w", err)
	}

	return contracts, nil
}

// For example num = 11, denom = 10 would be a 10% increase.
func build1559Tx(
	ctx context.Context,
	lggr logger.Logger,
	chain fevm.Chain,
	amount *big.Int,
	toAddr common.Address,
	nonce uint64,
	multiplierNum *big.Int,
	multiplierDenom *big.Int,
) (*types.Transaction, error) {
	tipCap, err1 := chain.Client.SuggestGasTipCap(ctx)
	if err1 != nil {
		return nil, err1
	}

	latestHead, err1 := chain.Client.HeaderByNumber(ctx, nil)
	if err1 != nil {
		return nil, err1
	}

	// Tipcap must be raised by 10%
	tipCap = big.NewInt(0).Div(big.NewInt(0).Mul(multiplierNum, tipCap), multiplierDenom)
	baseFee := big.NewInt(0).Div(big.NewInt(0).Mul(multiplierNum, latestHead.BaseFee), multiplierDenom)
	gasLimit, err1 := chain.Client.EstimateGas(ctx, ethereum.CallMsg{
		From:  chain.DeployerKey.From,
		To:    &toAddr,
		Value: amount,
	})
	if err1 != nil {
		return nil, err1
	}
	tx := types.NewTx(
		&types.DynamicFeeTx{
			Nonce:     nonce,
			GasTipCap: tipCap,
			GasFeeCap: big.NewInt(0).Add(baseFee, tipCap),
			Gas:       gasLimit,
			To:        &toAddr,
			Value:     amount,
		},
	)
	lggr.Infof("Sending 1559 transaction %s "+
		"base fee %v"+
		"tip %v", tx.Hash(), baseFee, tipCap)

	return tx, nil
}

// indexOf returns the index of s in slice, or -1.
func indexOf(slice []*cobra.Command, s string) int {
	for i, cmd := range slice {
		if cmd.Use == s {
			return i
		}
	}

	return -1
}

func sendGas(ctx context.Context, lggr logger.Logger, chain fevm.Chain, amountStr, toStr string, use1559 bool) error {
	nonce, err := chain.Client.PendingNonceAt(ctx, chain.DeployerKey.From)
	if err != nil {
		return err
	}
	// TODO: 1559 for better inclusion.
	amount, success := big.NewInt(0).SetString(amountStr, 10)
	if !success {
		return errors.New("invalid amount")
	}
	b, err := chain.Client.BalanceAt(ctx, chain.DeployerKey.From, nil)
	if err != nil {
		return err
	}
	fmt.Println("Current deployer key balance", b)
	toAddr := common.HexToAddress(toStr)
	if toAddr == (common.Address{}) {
		return errors.New("invalid to address")
	}

	var tx *types.Transaction
	if use1559 {
		// Lets just always boost 10%.
		tx, err = build1559Tx(
			ctx, lggr, chain, amount, toAddr, nonce, big.NewInt(1), big.NewInt(2))
		if err != nil {
			return err
		}
	} else {
		gp, err1 := chain.Client.SuggestGasPrice(ctx)
		if err1 != nil {
			return err1
		}
		// Estimate here because some chains like arbitrum need it.
		gasLimit, err1 := chain.Client.EstimateGas(ctx, ethereum.CallMsg{
			From:     chain.DeployerKey.From,
			GasPrice: gp,
			To:       &toAddr,
			Value:    amount,
		})
		if err1 != nil {
			return err1
		}
		tx = types.NewTx(
			&types.LegacyTx{
				Nonce:    nonce,
				GasPrice: gp,
				Gas:      gasLimit,
				To:       &toAddr,
				Value:    amount,
				Data:     []byte{},
			},
		)
		lggr.Infof("Sending legacy transaction %s"+
			"gas price %v", tx.Hash(), gp.String())
	}
	signedTx, err := chain.DeployerKey.Signer(chain.DeployerKey.From, tx)
	if err != nil {
		return err
	}

	// Note this does its own balance checks.
	err = chain.Client.SendTransaction(ctx, signedTx)
	if err != nil {
		return err
	}
	lggr.Infof("Confirming transaction %s", signedTx.Hash().Hex())
	_, err = chain.Confirm(signedTx)

	return err
}
