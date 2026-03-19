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
	"sync"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const l2scanRateLimit = 1

var l2scanRateLimiter = struct {
	ticker *time.Ticker
	once   sync.Once
}{}

type l2scanAPIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

type l2scanTxnInfo struct {
	Input string `json:"input"`
}

type l2scanTransactionListResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  []l2scanTxnInfo `json:"result"`
}

func newL2ScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnL2Scan(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the L2Scan API", cfg.Chain.EvmChainID)
	}
	apiURL := cfg.Network.BlockExplorer.URL
	if apiURL == "" {
		return nil, fmt.Errorf("l2scan API URL not configured for chain %s", cfg.Chain.Name)
	}
	apiKey := cfg.Network.BlockExplorer.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("l2scan API key not configured for chain %s", cfg.Chain.Name)
	}

	l2scanRateLimiter.once.Do(func() {
		l2scanRateLimiter.ticker = time.NewTicker(time.Second / l2scanRateLimit)
	})

	return &l2scanVerifier{
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

type l2scanVerifier struct {
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

func (v *l2scanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *l2scanVerifier) waitRateLimit(ctx context.Context) error {
	select {
	case <-l2scanRateLimiter.ticker.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (v *l2scanVerifier) IsVerified(ctx context.Context) (bool, error) {
	if err := v.waitRateLimit(ctx); err != nil {
		return false, err
	}

	params := url.Values{}
	params.Set("module", "contract")
	params.Set("action", "getabi")
	params.Set("address", v.address)
	params.Set("api_key", v.apiKey)

	abiURL := v.apiURL + "?" + params.Encode()
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}

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

	var apiResp l2scanAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp.Status == "1" && apiResp.Result != "", nil
}

func (v *l2scanVerifier) Verify(ctx context.Context) error {
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

	formData := url.Values{}
	formData.Set("module", "contract")
	formData.Set("action", "verifysourcecode")
	formData.Set("api_key", v.apiKey)
	formData.Set("contractaddress", v.address)
	formData.Set("sourceCode", sourceCode)
	formData.Set("codeformat", "solidity-standard-json-input")
	formData.Set("contractname", v.metadata.Name)
	formData.Set("compilerversion", v.metadata.Version)
	formData.Set("constructorArguements", constructorArgs)

	resp, err := sendL2ScanFormRequest[l2scanAPIResponse](ctx, v.httpClient, v.apiURL, formData)
	if err != nil {
		return fmt.Errorf("failed to verify contract: %w", err)
	}
	if resp.Status != "1" {
		return fmt.Errorf("verification submission failed: %s - %s", resp.Message, resp.Result)
	}

	v.lggr.Infof("Verification submitted successfully")

	verified, err = v.IsVerified(ctx)
	if err != nil {
		return fmt.Errorf("failed to check verification status after submission: %w", err)
	}
	if verified {
		return nil
	}

	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}
	for range 12 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollDur):
			verified, err = v.IsVerified(ctx)
			if err != nil {
				return err
			}
			if verified {
				return nil
			}
		}
	}

	return errors.New("verification timed out")
}

func (v *l2scanVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	if err := v.waitRateLimit(ctx); err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "txlist")
	params.Set("address", v.address)
	params.Set("page", "1")
	params.Set("offset", "1")
	params.Set("sort", "asc")
	params.Set("api_key", v.apiKey)

	txnURL := v.apiURL + "?" + params.Encode()
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}

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

	var txnResp l2scanTransactionListResponse
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
		return "", nil
	}

	return txInput[len(bytecode):], nil
}

func sendL2ScanFormRequest[T any](ctx context.Context, client *http.Client, apiURL string, formData url.Values) (T, error) {
	var empty T
	if apiURL == "" {
		return empty, errors.New("l2scan API URL cannot be empty")
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
