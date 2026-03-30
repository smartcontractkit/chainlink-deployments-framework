package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const etherscanURL = "https://api.etherscan.io/v2/api"

type etherscanAPIResponse[R any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  R      `json:"result"`
}

type transactionInfo struct {
	Input string `json:"input"`
}

const statusOK = "1"
const messageOK = "OK"

// maxVerificationPollAttempts limits polling to avoid stalling CI when the API never returns pass/fail.
const maxVerificationPollAttempts = 12 // ~1 min at 5s poll interval

// NewEtherscanV2ContractVerifier creates a verifier that uses metadata from ContractInputsProvider.
// apiBaseURL is optional: when empty, the default Etherscan v2 multiplexer is used; otherwise it must
// be the explorer API base URL from domain network config (CLD).
func NewEtherscanV2ContractVerifier(
	chain chainsel.Chain,
	apiKey string,
	apiBaseURL string,
	address string,
	metadata SolidityContractMetadata,
	contractType string,
	version string,
	pollInterval time.Duration,
	lggr logger.Logger,
	httpClient *http.Client,
) (verification.Verifiable, error) {
	return &etherscanVerifier{
		chain:        chain,
		apiKey:       apiKey,
		apiBaseURL:   strings.TrimSpace(apiBaseURL),
		address:      address,
		metadata:     metadata,
		contractType: contractType,
		version:      version,
		pollInterval: pollInterval,
		lggr:         lggr,
		httpClient:   httpClient,
	}, nil
}

type etherscanVerifier struct {
	chain        chainsel.Chain
	apiKey       string
	apiBaseURL   string
	address      string
	metadata     SolidityContractMetadata
	contractType string
	version      string
	pollInterval time.Duration
	lggr         logger.Logger
	httpClient   *http.Client
}

func (v *etherscanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *etherscanVerifier) IsVerified(ctx context.Context) (bool, error) {
	resp, err := sendEtherscanRequestForVerifier[string](v, ctx, "GET", "contract", "getabi", map[string]string{
		"address": v.address,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check verification status: %w", err)
	}
	if resp.Status != statusOK || !strings.EqualFold(resp.Message, messageOK) {
		if strings.Contains(strings.ToLower(resp.Result), "contract source code not verified") {
			return false, nil
		}

		return false, fmt.Errorf("etherscan API error while checking verification status: status=%s message=%s result=%s", resp.Status, resp.Message, resp.Result)
	}
	var js interface{}
	if err := json.Unmarshal([]byte(resp.Result), &js); err != nil {
		if strings.Contains(strings.ToLower(resp.Result), "contract source code not verified") {
			return false, nil
		}

		return false, fmt.Errorf("failed to parse ABI JSON from etherscan response: %w", err)
	}

	return true, nil
}

func (v *etherscanVerifier) Verify(ctx context.Context) error {
	verified, err := v.IsVerified(ctx)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if verified {
		v.lggr.Infof("%s is already verified", v.String())
		return nil
	}

	constructorArgs, err := v.getConstructorArgs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get constructor args: %w", err)
	}
	v.lggr.Infof("Got constructor args for %s: %s", v.String(), constructorArgs)

	sourceCode, err := v.metadata.SourceCode()
	if err != nil {
		return fmt.Errorf("failed to get source code: %w", err)
	}

	resp, err := sendEtherscanRequestForVerifier[string](v, ctx, "POST", "contract", "verifysourcecode", map[string]string{
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
	if resp.Status != statusOK || resp.Message != messageOK {
		return fmt.Errorf("etherscan error - status=%s message=%s", resp.Status, resp.Message)
	}
	v.lggr.Infof("Verification request submitted for %s", v.String())

	guid := resp.Result
	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}
	for range maxVerificationPollAttempts {
		statusResp, err := sendEtherscanRequestForVerifier[string](v, ctx, "GET", "contract", "checkverifystatus", map[string]string{
			"guid": guid,
		})
		if err != nil {
			return fmt.Errorf("failed to check verification status: %w", err)
		}
		resultLower := strings.ToLower(statusResp.Result)
		if statusResp.Status == statusOK && strings.Contains(resultLower, "pass") {
			return nil
		}
		if strings.Contains(resultLower, "fail") {
			return fmt.Errorf("verification failed: %s", statusResp.Result)
		}
		v.lggr.Infof("Verification status - %s, checking again in %s", statusResp.Result, pollDur)
		select {
		case <-time.After(pollDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return fmt.Errorf("verification timed out after %d attempts", maxVerificationPollAttempts)
}

func (v *etherscanVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	resp, err := sendEtherscanRequestForVerifier[[]transactionInfo](v, ctx, "GET", "account", "txlist", map[string]string{
		"address": v.address,
		"page":    "1",
		"offset":  "1",
		"sort":    "asc",
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
		return "", nil
	}

	return txInput[len(bytecode):], nil
}

func (v *etherscanVerifier) apiBase() string {
	if v.apiBaseURL != "" {
		return v.apiBaseURL
	}

	return etherscanURL
}

// etherscanRequestURL builds the request URL. If the configured base already contains chainid=, it is not added again.
func (v *etherscanVerifier) etherscanRequestURL(method string, form url.Values) string {
	base := v.apiBase()
	chainID := v.chain.EvmChainID
	lower := strings.ToLower(base)
	full := base
	if !strings.Contains(lower, "chainid=") {
		sep := "?"
		if strings.Contains(base, "?") {
			sep = "&"
		}
		full = base + sep + "chainid=" + strconv.FormatUint(chainID, 10)
	}
	if method == http.MethodGet {
		if strings.Contains(full, "?") {
			return full + "&" + form.Encode()
		}

		return full + "?" + form.Encode()
	}

	return full
}

func sendEtherscanRequestForVerifier[R any](v *etherscanVerifier, ctx context.Context, method, module, action string, extraParams map[string]string) (etherscanAPIResponse[R], error) {
	form := url.Values{}
	form.Add("module", module)
	form.Add("action", action)
	form.Add("apikey", v.apiKey)
	for k, val := range extraParams {
		form.Add(k, val)
	}

	reqURL := v.etherscanRequestURL(method, form)
	var reqBody io.Reader
	if method != http.MethodGet {
		reqBody = strings.NewReader(form.Encode())
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return etherscanAPIResponse[R]{}, fmt.Errorf("failed to create request: %w", err)
	}
	if method != http.MethodGet {
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return etherscanAPIResponse[R]{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return etherscanAPIResponse[R]{}, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return etherscanAPIResponse[R]{}, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var apiResp etherscanAPIResponse[R]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return etherscanAPIResponse[R]{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp, nil
}
