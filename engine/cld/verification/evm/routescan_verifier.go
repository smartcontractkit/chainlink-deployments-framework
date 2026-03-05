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
	"sync"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const routescanURL = "https://api.routescan.io/v2"
const routescanRateLimit = 2

// routescanChainIDs - see https://routescan.notion.site
var routescanChainIDs = map[string]map[uint64]string{
	"testnet": {
		21000001: "21000001", 3636: "3636", 80069: "80069", 9746: "9746_5", 43113: "43113",
	},
	"mainnet": {
		21000000: "21000000", 3637: "3637", 80094: "80094", 9745: "9745", 43114: "43114",
	},
}

func IsChainSupportedOnRouteScan(chainID uint64) (networkType string, ok bool) {
	for nt, ids := range routescanChainIDs {
		if _, found := ids[chainID]; found {
			return nt, true
		}
	}

	return "", false
}

var routescanRateLimiter = struct {
	ticker *time.Ticker
	once   sync.Once
}{}

type routescanTxInfo struct {
	Input string `json:"input"`
}

type routeScanAPIResponse[R any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  R      `json:"result"`
}

func newRouteScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	networkType, ok := IsChainSupportedOnRouteScan(cfg.Chain.EvmChainID)
	if !ok {
		return nil, fmt.Errorf("chain ID %d is not supported by the Routescan API", cfg.Chain.EvmChainID)
	}

	return &routescanVerifier{
		chain:        cfg.Chain,
		networkType:  networkType,
		apiKey:       cfg.Network.BlockExplorer.APIKey,
		address:      cfg.Address,
		metadata:     cfg.Metadata,
		contractType: cfg.ContractType,
		version:      cfg.Version,
		pollInterval: cfg.PollInterval,
		lggr:         cfg.Logger,
		httpClient:   cfg.HTTPClient,
	}, nil
}

type routescanVerifier struct {
	chain        chainsel.Chain
	networkType  string
	apiKey       string
	address      string
	metadata     SolidityContractMetadata
	contractType string
	version      string
	pollInterval time.Duration
	lggr         logger.Logger
	httpClient   *http.Client
}

func (v *routescanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *routescanVerifier) IsVerified(ctx context.Context) (bool, error) {
	resp, err := sendRoutescanRequest[string](ctx, v.httpClient, v.chain.EvmChainID, v.networkType, "GET", "contract", "getabi", v.apiKey, map[string]string{
		"address": v.address,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check verification status: %w", err)
	}
	if resp.Status != statusOK || !strings.EqualFold(resp.Message, messageOK) {
		return false, fmt.Errorf("routescan API error while checking verification status: status=%q message=%q", resp.Status, resp.Message)
	}
	var js interface{}
	if err := json.Unmarshal([]byte(resp.Result), &js); err != nil {
		if strings.Contains(strings.ToLower(resp.Result), "contract source code not verified") {
			return false, nil
		}

		return false, fmt.Errorf("failed to parse ABI JSON from routescan response: %w", err)
	}

	return true, nil
}

func (v *routescanVerifier) Verify(ctx context.Context) error {
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

	resp, err := sendRoutescanRequest[string](ctx, v.httpClient, v.chain.EvmChainID, v.networkType, "POST", "contract", "verifysourcecode", v.apiKey, map[string]string{
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
		return fmt.Errorf("routescan error - status=%s message=%s", resp.Status, resp.Message)
	}
	v.lggr.Infof("Verification request submitted for %s", v.String())

	guid := resp.Result
	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}
	for range maxVerificationPollAttempts {
		statusResp, err := sendRoutescanRequest[string](ctx, v.httpClient, v.chain.EvmChainID, v.networkType, "GET", "contract", "checkverifystatus", v.apiKey, map[string]string{
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

func (v *routescanVerifier) getConstructorArgs(ctx context.Context) (string, error) {
	resp, err := sendRoutescanRequest[[]routescanTxInfo](ctx, v.httpClient, v.chain.EvmChainID, v.networkType, "GET", "account", "txlist", v.apiKey, map[string]string{
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

func sendRoutescanRequest[R any](ctx context.Context, client *http.Client, chainID uint64, networkType string, method, module, action, key string, extraParams map[string]string) (routeScanAPIResponse[R], error) {
	routescanRateLimiter.once.Do(func() {
		routescanRateLimiter.ticker = time.NewTicker(time.Second / routescanRateLimit)
	})
	select {
	case <-routescanRateLimiter.ticker.C:
	case <-ctx.Done():
		return routeScanAPIResponse[R]{}, ctx.Err()
	}

	params := url.Values{}
	params.Add("module", module)
	params.Add("action", action)
	params.Add("apikey", key)
	for k, val := range extraParams {
		params.Add(k, val)
	}

	chainIDStr, ok := routescanChainIDs[networkType][chainID]
	if !ok {
		chainIDStr = strconv.FormatUint(chainID, 10)
	}

	var httpReq *http.Request
	var err error
	if method == "GET" {
		requestURL := routescanURL + fmt.Sprintf("/network/%s/evm/%s/etherscan/api?%s", networkType, chainIDStr, params.Encode())
		httpReq, err = http.NewRequestWithContext(ctx, method, requestURL, nil)
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, routescanURL+fmt.Sprintf("/network/%s/evm/%s/etherscan/api?", networkType, chainIDStr), strings.NewReader(params.Encode()))
		if err == nil {
			httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	if err != nil {
		return routeScanAPIResponse[R]{}, fmt.Errorf("failed to create request: %w", err)
	}

	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return routeScanAPIResponse[R]{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return routeScanAPIResponse[R]{}, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return routeScanAPIResponse[R]{}, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}

	var apiResp routeScanAPIResponse[R]
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return routeScanAPIResponse[R]{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp, nil
}
