package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type btrScanSourceCodeResponse struct {
	Status  string                    `json:"status"`
	Message string                    `json:"message"`
	Result  []btrScanSourceCodeResult `json:"result"`
}

type btrScanSourceCodeResult struct {
	SourceCode string `json:"SourceCode"`
}

type btrScanAPIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type btrScanTxnInfo struct {
	Input string `json:"input"`
}

type btrScanTransactionListResponse struct {
	Status  json.Number      `json:"status"`
	Message string           `json:"message"`
	Result  []btrScanTxnInfo `json:"result"`
}

type btrScanVerificationStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func newBtrScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnBtrScan(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the BtrScan API", cfg.Chain.EvmChainID)
	}
	apiURL := cfg.Network.BlockExplorer.URL
	if apiURL == "" {
		return nil, fmt.Errorf("btrscan API URL not configured for chain %s", cfg.Chain.Name)
	}
	apiKey := cfg.Network.BlockExplorer.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("btrscan API key not configured for chain %s", cfg.Chain.Name)
	}

	return &btrscanVerifier{
		chain:        cfg.Chain,
		apiURL:       apiURL,
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

type btrscanVerifier struct {
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

func (v *btrscanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *btrscanVerifier) IsVerified(ctx context.Context) (bool, error) {
	params := url.Values{}
	params.Set("apikey", v.apiKey)
	params.Set("module", "contract")
	params.Set("action", "getsourcecode")
	params.Set("address", v.address)

	resp, err := sendBtrScanGETRequest[btrScanSourceCodeResponse](ctx, v.httpClient, v.apiURL, params)
	if err != nil {
		return false, fmt.Errorf("failed to check verification status: %w", err)
	}
	if resp.Status == "1" && len(resp.Result) > 0 {
		return resp.Result[0].SourceCode != "", nil
	}

	return false, nil
}

func (v *btrscanVerifier) Verify(ctx context.Context) error {
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

	contractName := v.metadata.Name
	if idx := strings.LastIndex(contractName, ":"); idx >= 0 && idx+1 < len(contractName) {
		contractName = contractName[idx+1:]
	}

	formData := url.Values{}
	formData.Set("apikey", v.apiKey)
	formData.Set("module", "contract")
	formData.Set("action", "verifysourcecode")
	formData.Set("contractaddress", v.address)
	formData.Set("sourceCode", sourceCode)
	formData.Set("codeformat", "solidity-standard-json-input")
	formData.Set("contractname", contractName)
	formData.Set("compilerversion", v.metadata.Version)
	formData.Set("optimizationUsed", "1")
	formData.Set("runs", "200")
	formData.Set("licenseType", "1")
	if constructorArgs != "" {
		formData.Set("constructorArguements", constructorArgs)
	}

	resp, err := sendBtrScanPOSTRequest[btrScanAPIResponse](ctx, v.httpClient, v.apiURL, formData)
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

func (v *btrscanVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Set("apikey", v.apiKey)
	params.Set("module", "account")
	params.Set("action", "txlist")
	params.Set("address", v.address)
	params.Set("page", "1")
	params.Set("offset", "1")
	params.Set("sort", "asc")

	resp, err := sendBtrScanGETRequest[btrScanTransactionListResponse](ctx, v.httpClient, v.apiURL, params)
	if err != nil {
		return "", fmt.Errorf("failed to get contract creation info: %w", err)
	}
	if resp.Status.String() != "1" {
		return "", fmt.Errorf("API call failed: %s", resp.Message)
	}
	if len(resp.Result) == 0 {
		return "", errors.New("no transactions found")
	}

	tx := resp.Result[0]
	bytecode := strings.TrimPrefix(v.metadata.Bytecode, "0x")
	txInput := strings.TrimPrefix(tx.Input, "0x")
	if !strings.HasPrefix(txInput, bytecode) {
		return "", nil
	}

	return txInput[len(bytecode):], nil
}

func (v *btrscanVerifier) pollVerificationStatus(ctx context.Context, guid string) error {
	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}
	timeout := time.After(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return errors.New("verification timeout exceeded")
		case <-time.After(pollDur):
			params := url.Values{}
			params.Set("apikey", v.apiKey)
			params.Set("module", "contract")
			params.Set("action", "checkverifystatus")
			params.Set("guid", guid)

			resp, err := sendBtrScanGETRequest[btrScanVerificationStatusResponse](ctx, v.httpClient, v.apiURL, params)
			if err != nil {
				v.lggr.Warnf("Failed to check verification status: %v", err)

				continue
			}
			switch resp.Status {
			case "1":
				return nil
			case "2":
				return fmt.Errorf("contract verification failed: %s - %s", resp.Message, resp.Result)
			default:
				v.lggr.Infof("Verification pending: %s", resp.Message)
			}
		}
	}
}

func sendBtrScanGETRequest[T any](ctx context.Context, client *http.Client, apiURL string, params url.Values) (T, error) {
	var empty T
	if apiURL == "" {
		return empty, errors.New("btrscan API URL cannot be empty")
	}
	if client == nil {
		client = http.DefaultClient
	}

	fullURL := apiURL + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return empty, fmt.Errorf("failed to create request: %w", err)
	}

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

func sendBtrScanPOSTRequest[T any](ctx context.Context, client *http.Client, apiURL string, formData url.Values) (T, error) {
	var empty T
	if apiURL == "" {
		return empty, errors.New("btrscan API URL cannot be empty")
	}
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return empty, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
