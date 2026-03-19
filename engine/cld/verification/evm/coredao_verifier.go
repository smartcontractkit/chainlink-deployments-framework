package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type coreDAOAPIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type coreDAOTxnInfo struct {
	Input string `json:"input"`
}

type coreDAOTransactionListResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Result  []coreDAOTxnInfo `json:"result"`
}

type coreDAOVerifyRequest struct {
	Action               string `json:"action"`
	Address              string `json:"address"`
	APIKey               string `json:"apikey"`
	CodeFormat           string `json:"codeformat"`
	CompilerVersion      string `json:"compilerversion"`
	ConstructorArguments string `json:"constructorArguements"`
	ContractAddress      string `json:"contractaddress"`
	EVMVersion           string `json:"evmversion,omitempty"`
	LicenseType          int    `json:"licenseType"`
	Module               string `json:"module"`
	OptimizationUsed     string `json:"optimizationUsed"`
	Runs                 int    `json:"runs"`
	SourceCode           string `json:"sourceCode"`
}

func newCoreDAOVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnCoreDAO(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the CoreDAO API", cfg.Chain.EvmChainID)
	}
	apiURL := cfg.Network.BlockExplorer.URL
	if apiURL == "" {
		return nil, fmt.Errorf("coredao API URL not configured for chain %s", cfg.Chain.Name)
	}
	apiKey := cfg.Network.BlockExplorer.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("coredao API key not configured for chain %s", cfg.Chain.Name)
	}

	return &coredaoVerifier{
		chain:        cfg.Chain,
		apiURL:       strings.TrimSuffix(apiURL, "/"),
		apiKey:       apiKey,
		address:      cfg.Address,
		metadata:     cfg.Metadata,
		contractType: cfg.ContractType,
		version:      cfg.Version,
		pollInterval: cfg.PollInterval,
		lggr:         cfg.Logger,
		httpClient:   cfg.HTTPClient,
	}, nil
}

type coredaoVerifier struct {
	chain        chainsel.Chain
	apiURL       string
	apiKey       string
	address      string
	metadata     SolidityContractMetadata
	contractType string
	version      string
	pollInterval time.Duration
	lggr         logger.Logger
	httpClient   *http.Client
}

func (v *coredaoVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *coredaoVerifier) IsVerified(ctx context.Context) (bool, error) {
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	abiURL := fmt.Sprintf("%s/contracts/abi_of_verified_contract/%s?apikey=%s", v.apiURL, v.address, v.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, abiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var apiResp coreDAOAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp.Status == "1" && apiResp.Result != "", nil
}

func (v *coredaoVerifier) Verify(ctx context.Context) error {
	verified, err := v.IsVerified(ctx)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if verified {
		v.lggr.Infof("%s is already verified", v)

		return nil
	}

	sourceCode, err := v.metadata.SourceCode()
	if err != nil {
		return fmt.Errorf("failed to get source code: %w", err)
	}

	constructorArgs, err := v.getConstructorArgs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get constructor args: %w", err)
	}

	verifyRequest := coreDAOVerifyRequest{
		Action:               "verifysourcecode",
		Address:              v.address,
		APIKey:               v.apiKey,
		CodeFormat:           "solidity-standard-json-input",
		CompilerVersion:      v.metadata.Version,
		ConstructorArguments: constructorArgs,
		ContractAddress:      v.address,
		LicenseType:          1,
		Module:               "contract",
		OptimizationUsed:     "1",
		Runs:                 200,
		SourceCode:           sourceCode,
	}

	verifyURL := fmt.Sprintf("%s/contracts/verify_source_code?apikey=%s", v.apiURL, v.apiKey)
	resp, err := sendCoreDAOPOSTRequest[coreDAOAPIResponse](ctx, v.httpClient, verifyURL, verifyRequest)
	if err != nil {
		return fmt.Errorf("failed to verify contract: %w", err)
	}
	if resp.Status != "1" {
		return fmt.Errorf("verification submission failed: %s - %s", resp.Message, resp.Result)
	}

	guid := resp.Result
	v.lggr.Infof("Verification submitted successfully. GUID: %s", guid)

	return v.pollVerificationStatus(ctx, guid)
}

func (v *coredaoVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	txnURL := fmt.Sprintf("%s/accounts/list_of_txs_by_address/%s?apikey=%s", v.apiURL, v.address, v.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, txnURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var txnResp coreDAOTransactionListResponse
	if err := json.Unmarshal(body, &txnResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if txnResp.Status != "1" {
		return "", fmt.Errorf("API call failed: %s", txnResp.Message)
	}
	if len(txnResp.Result) == 0 {
		return "", errors.New("no transactions found")
	}

	tx := txnResp.Result[0]
	bytecode := strings.TrimPrefix(v.metadata.Bytecode, "0x")
	txInput := strings.TrimPrefix(tx.Input, "0x")
	if !strings.HasPrefix(txInput, bytecode) {
		return "", errors.New("contract creation tx input does not contain contract bytecode")
	}

	return txInput[len(bytecode):], nil
}

func (v *coredaoVerifier) pollVerificationStatus(ctx context.Context, guid string) error {
	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}
	timeout := time.After(time.Minute)
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return errors.New("verification timeout exceeded")
		case <-time.After(pollDur):
			statusURL := fmt.Sprintf("%s/contracts/check_proxy_contract_verification_submission_status_using_cURL?apikey=%s&guid=%s", v.apiURL, v.apiKey, guid)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
			if err != nil {
				v.lggr.Warnf("Failed to create status check request: %v", err)

				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				v.lggr.Warnf("Failed to check verification status: %v", err)

				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				v.lggr.Warnf("Failed to read status response: %v", err)

				continue
			}
			if resp.StatusCode != http.StatusOK {
				v.lggr.Warnf("Status check HTTP error - status=%d", resp.StatusCode)

				continue
			}

			var statusResp coreDAOAPIResponse
			if err := json.Unmarshal(body, &statusResp); err != nil {
				v.lggr.Warnf("Failed to decode status response: %v", err)

				continue
			}

			switch statusResp.Status {
			case "1":
				return nil
			case "0":
				v.lggr.Infof("Verification pending: %s", statusResp.Message)

				continue
			default:
				return fmt.Errorf("contract verification failed: %s - %s", statusResp.Message, statusResp.Result)
			}
		}
	}
}

func sendCoreDAOPOSTRequest[T any](ctx context.Context, client *http.Client, apiURL string, requestData any) (T, error) {
	var empty T
	if apiURL == "" {
		return empty, errors.New("coredao API URL cannot be empty")
	}
	if client == nil {
		client = http.DefaultClient
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return empty, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return empty, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return empty, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var apiResp T
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return empty, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp, nil
}
