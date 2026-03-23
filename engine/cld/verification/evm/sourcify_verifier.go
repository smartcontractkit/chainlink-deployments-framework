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
	"strconv"
	"strings"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type sourcifyAPIResponse struct {
	Status string `json:"status"`
}

type sourcifyVerificationResponse struct {
	Result []sourcifyAPIResponse `json:"result"`
}

func newSourcifyVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnSourcify(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the Sourcify API", cfg.Chain.EvmChainID)
	}
	apiURL := cfg.Network.BlockExplorer.URL
	if apiURL == "" {
		return nil, fmt.Errorf("sourcify API URL not configured for chain %s", cfg.Chain.Name)
	}

	return &sourcifyVerifier{
		chain:        cfg.Chain,
		apiURL:       strings.TrimSuffix(apiURL, "/"),
		address:      cfg.Address,
		metadata:     cfg.Metadata,
		contractType: cfg.ContractType,
		version:      cfg.Version,
		pollInterval: cfg.PollInterval,
		lggr:         cfg.Logger,
		httpClient:   cfg.HTTPClient,
	}, nil
}

type sourcifyVerifier struct {
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

func (v *sourcifyVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *sourcifyVerifier) IsVerified(ctx context.Context) (bool, error) {
	resp, err := sendSourcifyRequest[sourcifyAPIResponse](ctx, v.httpClient, v.chain.EvmChainID, "GET", "files/any", v.apiURL, map[string]string{
		"address": v.address,
	})
	if err != nil {
		if strings.Contains(err.Error(), "Files have not been found") {
			return false, nil
		}

		return false, fmt.Errorf("failed to check verification status: %w", err)
	}

	return resp.Status == "full" || resp.Status == "partial", nil
}

func (v *sourcifyVerifier) Verify(ctx context.Context) error {
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

	contractName := v.metadata.Name
	if idx := strings.LastIndex(contractName, ":"); idx >= 0 && idx+1 < len(contractName) {
		contractName = contractName[idx+1:]
	}

	requestData := map[string]any{
		"address":         v.address,
		"chain":           strconv.FormatUint(v.chain.EvmChainID, 10),
		"files":           map[string]string{"value": sourceCode},
		"compilerVersion": v.metadata.Version,
		"contractName":    contractName,
	}

	resp, err := sendSourcifyRequest[sourcifyVerificationResponse](ctx, v.httpClient, v.chain.EvmChainID, "POST", "/verify/solc-json", v.apiURL, requestData)
	if err != nil {
		return fmt.Errorf("failed to verify contract: %w", err)
	}
	if len(resp.Result) == 0 {
		return errors.New("invalid verification response")
	}
	if resp.Result[0].Status != "partial" && resp.Result[0].Status != "full" {
		return fmt.Errorf("unexpected verification status: %s", resp.Result[0].Status)
	}
	v.lggr.Infof("Verification status - %s", resp.Result[0].Status)

	return nil
}

func sendSourcifyRequest[T any](ctx context.Context, client *http.Client, chainID uint64, method, path, apiURL string, extraParams any) (T, error) {
	var empty T
	if apiURL == "" {
		return empty, errors.New("sourcify API URL cannot be empty")
	}
	if client == nil {
		client = http.DefaultClient
	}

	var httpReq *http.Request
	var err error
	if method == "GET" {
		baseURL, parseErr := url.Parse(apiURL)
		if parseErr != nil {
			return empty, fmt.Errorf("failed to parse base URL: %w", parseErr)
		}
		params := extraParams.(map[string]string)
		fullURL := baseURL.JoinPath(path, strconv.FormatUint(chainID, 10))
		for _, value := range params {
			fullURL = fullURL.JoinPath(value)
		}
		httpReq, err = http.NewRequestWithContext(ctx, method, fullURL.String(), nil)
	} else {
		jsonData, marshalErr := json.Marshal(extraParams)
		if marshalErr != nil {
			return empty, fmt.Errorf("failed to marshal JSON: %w", marshalErr)
		}
		httpReq, err = http.NewRequestWithContext(ctx, method, apiURL+path, bytes.NewReader(jsonData))
		if err == nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return empty, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return empty, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, fmt.Errorf("failed to read response body: %w", err)
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
