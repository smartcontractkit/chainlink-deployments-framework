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

const (
	socialscanURL                 = "https://api.socialscan.io/%s/v1/developer/api?"
	socialscanVerify              = "https://api.socialscan.io/%s/v1/explorer/command_api/contract?"
	socialscanMaxPollAttempts     = 60
	socialscanDefaultPollInterval = 5 * time.Second
)

type socialscanTransactionInfo struct {
	Input string `json:"input"`
}

type socialscanAPIResponse[R any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  R      `json:"result"`
}

func newSocialScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	chainName, ok := getSocialScanChainName(cfg.Chain.EvmChainID)
	if !ok {
		return nil, fmt.Errorf("chain ID %d is not supported by the SocialScan API", cfg.Chain.EvmChainID)
	}
	apiKey := cfg.Network.BlockExplorer.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("socialscan API key not configured for chain %s", cfg.Chain.Name)
	}

	return &socialscanVerifier{
		chain:        cfg.Chain,
		chainName:    chainName,
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

type socialscanVerifier struct {
	chain        chainsel.Chain
	chainName    string
	apiKey       string
	address      string
	metadata     SolidityContractMetadata
	contractType string
	version      string
	pollInterval time.Duration
	lggr         logger.Logger
	httpClient   *http.Client
}

func (v *socialscanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *socialscanVerifier) IsVerified(ctx context.Context) (bool, error) {
	resp, err := sendSocialscanRequest[string](ctx, v.httpClient, v.chainName, "GET", "contract", "getabi", v.apiKey, map[string]string{
		"address": v.address,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check verification status: %w", err)
	}
	if resp.Status != statusOK {
		return false, nil
	}

	var js interface{}

	return json.Unmarshal([]byte(resp.Result), &js) == nil, nil
}

func (v *socialscanVerifier) Verify(ctx context.Context) error {
	verified, err := v.IsVerified(ctx)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if verified {
		v.lggr.Infof("%s is already verified", v)

		return nil
	}

	constructorArgs, err := v.getConstructorArgs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get constructor args: %w", err)
	}
	v.lggr.Infof("Got constructor args for %s", v.String())

	sourceCode, err := v.metadata.SourceCode()
	if err != nil {
		return fmt.Errorf("failed to get source code: %w", err)
	}

	resp, err := sendSocialscanRequest[string](ctx, v.httpClient, v.chainName, "POST", "contract", "verifysourcecode", v.apiKey, map[string]string{
		"contractaddress":      v.address,
		"sourceCode":           sourceCode,
		"codeformat":           "solidity-standard-json-input",
		"contractname":         v.metadata.Name,
		"compilerversion":      v.metadata.Version,
		"constructorArguments": constructorArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to verify contract: %w", err)
	}
	if resp.Status != statusOK {
		return fmt.Errorf("socialscan error - status=%s message=%s", resp.Status, resp.Message)
	}
	v.lggr.Infof("Verification request submitted for %s", v.String())

	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = socialscanDefaultPollInterval
	}
	for attempts := range socialscanMaxPollAttempts {
		verified, err = v.IsVerified(ctx)
		if err != nil {
			return fmt.Errorf("failed to check verification status: %w", err)
		}
		if verified {
			return nil
		}
		v.lggr.Infof("Verification status - checking again in %s (attempt %d/%d)", pollDur, attempts+1, socialscanMaxPollAttempts)
		select {
		case <-time.After(pollDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("verification polling exceeded maximum attempts (%d)", socialscanMaxPollAttempts)
}

func (v *socialscanVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	resp, err := sendSocialscanRequest[[]socialscanTransactionInfo](ctx, v.httpClient, v.chainName, "GET", "account", "txlist", v.apiKey, map[string]string{
		"address":    v.address,
		"page":       "1",
		"offset":     "1",
		"sort":       "asc",
		"startblock": "0",
		"endblock":   "99999999",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get contract creation info: %w", err)
	}
	if len(resp.Result) != 1 {
		return "", fmt.Errorf("expected 1 result, got %d", len(resp.Result))
	}

	tx := resp.Result[0]
	bytecode := strings.TrimPrefix(v.metadata.Bytecode, "0x")
	txInput := strings.TrimPrefix(tx.Input, "0x")
	if !strings.HasPrefix(txInput, bytecode) {
		return "", errors.New("contract creation tx input does not contain contract bytecode")
	}

	return txInput[len(bytecode):], nil
}

func sendSocialscanRequest[R any](ctx context.Context, client *http.Client, chainName, method, module, action, apiKey string, extraParams map[string]string) (socialscanAPIResponse[R], error) {
	if client == nil {
		client = http.DefaultClient
	}

	var httpReq *http.Request
	var err error
	if method == "GET" {
		params := url.Values{}
		params.Add("module", module)
		params.Add("action", action)
		params.Add("apikey", apiKey)
		for key, value := range extraParams {
			params.Add(key, value)
		}
		requestURL := fmt.Sprintf(socialscanURL, chainName) + params.Encode()
		httpReq, err = http.NewRequestWithContext(ctx, method, requestURL, nil)
	} else {
		form := url.Values{}
		form.Add("module", module)
		form.Add("action", action)
		for key, value := range extraParams {
			form.Add(key, value)
		}
		params := url.Values{}
		params.Add("apikey", apiKey)
		requestURL := fmt.Sprintf(socialscanVerify, chainName) + params.Encode()
		httpReq, err = http.NewRequestWithContext(ctx, method, requestURL, strings.NewReader(form.Encode()))
		if err == nil {
			httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return socialscanAPIResponse[R]{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return socialscanAPIResponse[R]{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return socialscanAPIResponse[R]{}, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return socialscanAPIResponse[R]{}, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var apiResp socialscanAPIResponse[R]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return socialscanAPIResponse[R]{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp, nil
}
