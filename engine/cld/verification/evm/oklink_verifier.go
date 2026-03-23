package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	oklinkBaseURL         = "https://www.oklink.com/api/v5"
	oklinkRateLimit       = 2
	oklinkBurst           = 1
	oklinkMaxRetries      = 5
	oklinkMaxPollAttempts = 60
	oklinkStatusOK        = "0"
	oklinkMessageOK       = "Success"
)

var oklinkRateLimiter = struct {
	limiter *rate.Limiter
	once    sync.Once
}{}

type okLinkResponse[R any] struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []R    `json:"data"`
}

type oklinkVerifyContractInfo struct {
	SourceCode      string `json:"sourceCode"`
	ContractName    string `json:"contractName"`
	CompilerVersion string `json:"compilerVersion"`
}

func newOkLinkVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	chainShortName, ok := GetOkLinkShortName(cfg.Chain.EvmChainID)
	if !ok {
		return nil, fmt.Errorf("chain ID %d is not supported by the OKLink API", cfg.Chain.EvmChainID)
	}
	apiKey := cfg.Network.BlockExplorer.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("OKLink API key not configured for chain %s", cfg.Chain.Name)
	}

	return &oklinkVerifier{
		chainShortName: chainShortName,
		apiKey:         apiKey,
		address:        cfg.Address,
		metadata:       cfg.Metadata,
		contractType:   cfg.ContractType,
		version:        cfg.Version,
		pollInterval:   cfg.PollInterval,
		lggr:           cfg.Logger,
		httpClient:     cfg.HTTPClient,
	}, nil
}

type oklinkVerifier struct {
	chainShortName string
	apiKey         string
	address        string
	metadata       SolidityContractMetadata
	contractType   string
	version        string
	pollInterval   time.Duration
	lggr           logger.Logger
	httpClient     *http.Client
}

func (v *oklinkVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chainShortName)
}

func (v *oklinkVerifier) IsVerified(ctx context.Context) (bool, error) {
	resp, err := sendOkLinkRequest[oklinkVerifyContractInfo](ctx, v.httpClient, "GET", "/explorer/contract/verify-contract-info", v.apiKey, map[string]string{
		"contractAddress": v.address,
		"chainShortName":  v.chainShortName,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check verification status: %w", err)
	}
	if resp.Code != oklinkStatusOK {
		return false, fmt.Errorf("API returned non-success code: %s, msg: %s", resp.Code, resp.Msg)
	}

	return len(resp.Data) > 0, nil
}

func (v *oklinkVerifier) Verify(ctx context.Context) error {
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

	evmVersion := "default"
	if evmRaw, ok := v.metadata.Settings["evmVersion"]; ok {
		if evmStr, ok := evmRaw.(string); ok && evmStr != "" {
			evmVersion = evmStr
		}
	}

	resp, err := sendOkLinkRequest[string](ctx, v.httpClient, "POST", "/explorer/contract/verify-source-code", v.apiKey, map[string]string{
		"chainShortName":  v.chainShortName,
		"contractAddress": v.address,
		"contractName":    v.metadata.Name,
		"sourceCode":      sourceCode,
		"codeFormat":      "solidity-standard-json-input",
		"compilerVersion": v.metadata.Version,
		"evmVersion":      evmVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to verify contract: %w", err)
	}
	if resp.Code != oklinkStatusOK || len(resp.Data) == 0 {
		return fmt.Errorf("oklink error - code=%s msg=%s", resp.Code, resp.Msg)
	}

	guid := resp.Data[0]
	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}
	for attempts := range oklinkMaxPollAttempts {
		statusResp, err := sendOkLinkRequest[string](ctx, v.httpClient, "POST", "/explorer/contract/check-verify-result", v.apiKey, map[string]string{
			"chainShortName": v.chainShortName,
			"guid":           guid,
		})
		if err != nil {
			return fmt.Errorf("failed to check verification status: %w", err)
		}
		if statusResp.Code == oklinkStatusOK && len(statusResp.Data) > 0 && statusResp.Data[0] == "Success" {
			return nil
		}
		if statusResp.Code == oklinkStatusOK && len(statusResp.Data) > 0 && statusResp.Data[0] == "Fail" {
			return errors.New("verification failed")
		}
		v.lggr.Infof("Verification status - checking again in %s (attempt %d/%d)", pollDur, attempts+1, oklinkMaxPollAttempts)
		select {
		case <-time.After(pollDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return errors.New("verification status polling exceeded maximum attempts")
}

func sendOkLinkRequest[R any](ctx context.Context, client *http.Client, method, endpoint, apiKey string, extraParams map[string]string) (okLinkResponse[R], error) {
	oklinkRateLimiter.once.Do(func() {
		oklinkRateLimiter.limiter = rate.NewLimiter(rate.Limit(oklinkRateLimit), oklinkBurst)
	})
	if err := oklinkRateLimiter.limiter.Wait(ctx); err != nil {
		return okLinkResponse[R]{}, fmt.Errorf("rate limiter error: %w", err)
	}

	if client == nil {
		client = http.DefaultClient
	}

	fullURL := oklinkBaseURL + endpoint
	var lastErr error
	for attempt := range oklinkMaxRetries {
		var req *http.Request
		var err error
		if method == http.MethodGet {
			query := url.Values{}
			for k, v := range extraParams {
				query.Set(k, v)
			}
			req, err = http.NewRequestWithContext(ctx, method, fullURL+"?"+query.Encode(), nil)
		} else {
			payload := make(map[string]string)
			for k, v := range extraParams {
				payload[k] = v
			}
			var jsonBody []byte
			jsonBody, err = json.Marshal(payload)
			if err != nil {
				return okLinkResponse[R]{}, fmt.Errorf("failed to marshal payload: %w", err)
			}
			req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(jsonBody))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
			}
		}
		if err != nil {
			return okLinkResponse[R]{}, fmt.Errorf("failed to create request: %w", err)
		}
		if apiKey != "" {
			req.Header.Set("Ok-Access-Key", apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			return okLinkResponse[R]{}, fmt.Errorf("HTTP request failed: %w", err)
		}
		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return okLinkResponse[R]{}, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := time.Duration(1<<attempt) * time.Second
			lastErr = fmt.Errorf("HTTP 429: %s", string(respBytes))
			select {
			case <-ctx.Done():
				return okLinkResponse[R]{}, fmt.Errorf("context canceled during backoff: %w", ctx.Err())
			case <-time.After(wait):
			}

			continue
		}
		if resp.StatusCode != http.StatusOK {
			return okLinkResponse[R]{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
		}

		var parsed okLinkResponse[R]
		if err := json.Unmarshal(respBytes, &parsed); err != nil {
			return okLinkResponse[R]{}, fmt.Errorf("failed to decode JSON: %w", err)
		}

		return parsed, nil
	}

	return okLinkResponse[R]{}, fmt.Errorf("exceeded max retries: %w", lastErr)
}
