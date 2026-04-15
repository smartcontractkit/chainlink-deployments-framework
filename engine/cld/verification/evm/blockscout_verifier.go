package evm

import (
	"bytes"
	"context"
	"encoding/json"
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

func newBlockscoutVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	apiURL := strings.TrimSpace(cfg.Network.BlockExplorer.URL)
	if apiURL == "" {
		return nil, fmt.Errorf("blockscout API URL not configured for chain %s", cfg.Chain.Name)
	}

	return &blockscoutVerifier{
		chain:        cfg.Chain,
		apiURL:       apiURL,
		address:      cfg.Address,
		metadata:     cfg.Metadata,
		contractType: cfg.ContractType,
		version:      cfg.Version,
		pollInterval: cfg.PollInterval,
		lggr:         cfg.Logger,
		httpClient:   cfg.HTTPClient,
	}, nil
}

type blockscoutVerifier struct {
	chain        chainsel.Chain
	apiURL       string
	address      string
	metadata     SolidityContractMetadata
	contractType string
	version      string
	pollInterval time.Duration
	lggr         logger.Logger
	httpClient   *http.Client
}

func (v *blockscoutVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

// apiBase returns the base URL for Blockscout API calls. If v.apiURL has no path or path is "/",
// appends "/api"; otherwise preserves the configured path to avoid clobbering custom API paths.
func (v *blockscoutVerifier) apiBase() (*url.URL, error) {
	u, err := url.Parse(v.apiURL)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" || path == "/" {
		u.Path = "/api"
	}

	return u, nil
}

func (v *blockscoutVerifier) IsVerified(ctx context.Context) (bool, error) {
	u, err := v.apiBase()
	if err != nil {
		return false, fmt.Errorf("failed to parse API URL: %w", err)
	}
	q := u.Query()
	q.Set("module", "contract")
	q.Set("action", "getabi")
	q.Set("address", v.address)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return false, fmt.Errorf("blockscout IsVerified: unexpected status code %d: %s", resp.StatusCode, string(limitedBody))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	var result struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}
	if result.Status == "1" && result.Result != "" {
		v.lggr.Infof("Contract %s is verified on Blockscout", v.address)
		return true, nil
	}

	return false, nil
}

func (v *blockscoutVerifier) Verify(ctx context.Context) error {
	verified, err := v.IsVerified(ctx)
	if err != nil {
		return err
	}
	if verified {
		v.lggr.Infof("%s is already verified", v.String())
		return nil
	}

	constructorArgs, err := v.getConstructorArgs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get constructor args: %w", err)
	}

	sourceCode, err := v.metadata.SourceCode()
	if err != nil {
		return fmt.Errorf("failed to get source code: %w", err)
	}
	contractName := v.metadata.Name
	if contractName == "" {
		contractName = v.contractType
	}

	// Use Etherscan-compatible verifysourcecode with standard JSON input. The legacy
	// action=verify JSON body treats contractSourceCode as a single .sol file, so
	// Standard-JSON from SourceCode() was compiled as Solidity and failed with a parse error.
	u, err := v.apiBase()
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	q := u.Query()
	q.Set("module", "contract")
	q.Set("action", "verifysourcecode")
	u.RawQuery = q.Encode()
	verifyURL := u.String()

	form := url.Values{}
	form.Set("contractaddress", v.address)
	form.Set("sourceCode", sourceCode)
	form.Set("codeformat", "solidity-standard-json-input")
	form.Set("contractname", contractName)
	form.Set("compilerversion", v.metadata.Version)
	form.Set("constructorArguments", constructorArgs)

	v.lggr.Infof("Blockscout verifysourcecode submit: address=%s contractname=%q compilerversion=%q codeformat=solidity-standard-json-input constructorArgs_hex_len=%d metadata_name=%q",
		v.address, contractName, v.metadata.Version, len(constructorArgs), v.metadata.Name)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read verify response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var submitResp etherscanAPIResponse[string]
	if err := json.Unmarshal(body, &submitResp); err != nil {
		snippet := string(body)
		if len(snippet) > 512 {
			snippet = snippet[:512] + "..."
		}

		return fmt.Errorf("decode verifysourcecode response: %w (body=%s)", err, snippet)
	}
	if submitResp.Status != statusOK || !strings.EqualFold(submitResp.Message, messageOK) {
		msg := submitResp.Message
		if msg == "" {
			msg = submitResp.Result
		}
		if msg == "" {
			msg = string(bytes.TrimSpace(body))
		}

		return fmt.Errorf("blockscout verifysourcecode rejected: status=%s message=%s: %s", submitResp.Status, submitResp.Message, msg)
	}

	guid := submitResp.Result
	// Async verification: poll with guid (same flow as Etherscan-compatible APIs).
	pollDur := v.effectivePollInterval()
	for range maxVerificationPollAttempts {
		statusResp, rawCheckBody, err := v.blockscoutCheckVerifyStatus(ctx, guid)
		if err != nil {
			return fmt.Errorf("check verification status: %w", err)
		}
		resultLower := strings.ToLower(statusResp.Result)
		if statusResp.Status == statusOK && strings.Contains(resultLower, "pass") {
			v.lggr.Infof("Verification submitted successfully for contract %s", v.address)
			return nil
		}
		if strings.Contains(resultLower, "fail") {
			msg := strings.TrimSpace(statusResp.Message)
			// Many Blockscout instances return status=1 and message=OK even when result begins with
			// "Fail - …"; compiler details are often not in this JSON. Log full body for debugging.
			v.lggr.Warnf("Blockscout checkverifystatus (failure): full JSON: %s", truncateForLog(string(rawCheckBody), 6000))
			hint := ""
			if statusResp.Status == statusOK && strings.EqualFold(msg, messageOK) {
				hint = " (this explorer uses status=1/message=OK even on verification failure; details are usually not in the API — check explorer UI or contractname/metadata vs on-chain bytecode)"
			}

			return fmt.Errorf("blockscout verification failed: result=%q message=%q api_status=%s%s; submitted contractname=%q compilerversion=%q constructorArgs_hex_len=%d; raw_checkverifystatus=%s",
				statusResp.Result, msg, statusResp.Status, hint,
				contractName, v.metadata.Version, len(constructorArgs), truncateForLog(string(rawCheckBody), 2000))
		}
		v.lggr.Infof("Verification status — %s, checking again in %s", statusResp.Result, pollDur)
		select {
		case <-time.After(pollDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("blockscout verification timed out after %d attempts", maxVerificationPollAttempts)
}

func (v *blockscoutVerifier) effectivePollInterval() time.Duration {
	d := v.pollInterval
	if d <= 0 {
		d = 5 * time.Second
	}

	return d
}

// blockscoutCheckVerifyStatus calls module=contract&action=checkverifystatus (Etherscan-compatible).
// rawBody is the exact JSON bytes (for logging when the parsed fields are unhelpful).
func (v *blockscoutVerifier) blockscoutCheckVerifyStatus(ctx context.Context, guid string) (etherscanAPIResponse[string], []byte, error) {
	u, err := v.apiBase()
	if err != nil {
		return etherscanAPIResponse[string]{}, nil, err
	}
	q := u.Query()
	q.Set("module", "contract")
	q.Set("action", "checkverifystatus")
	q.Set("guid", guid)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return etherscanAPIResponse[string]{}, nil, err
	}
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return etherscanAPIResponse[string]{}, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return etherscanAPIResponse[string]{}, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return etherscanAPIResponse[string]{}, body, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}
	var apiResp etherscanAPIResponse[string]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return etherscanAPIResponse[string]{}, body, fmt.Errorf("decode checkverifystatus: %w", err)
	}

	return apiResp, body, nil
}

func truncateForLog(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "…(truncated)"
}

// getConstructorArgs mirrors Etherscan logic: extract constructor suffix from the contract-creation tx.
func (v *blockscoutVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	u, err := v.apiBase()
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("module", "account")
	q.Set("action", "txlist")
	q.Set("address", v.address)
	q.Set("page", "1")
	q.Set("offset", "1")
	q.Set("sort", "asc")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var apiResp etherscanAPIResponse[[]transactionInfo]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("decode txlist: %w", err)
	}
	if apiResp.Status != statusOK {
		return "", fmt.Errorf("txlist API error: status=%s message=%s", apiResp.Status, apiResp.Message)
	}
	if len(apiResp.Result) != 1 {
		return "", fmt.Errorf("expected 1 contract creation tx, got %d", len(apiResp.Result))
	}
	tx := apiResp.Result[0]
	bytecode := strings.TrimPrefix(v.metadata.Bytecode, "0x")
	txInput := strings.TrimPrefix(tx.Input, "0x")
	if !strings.HasPrefix(txInput, bytecode) {
		return "", nil
	}

	return txInput[len(bytecode):], nil
}
