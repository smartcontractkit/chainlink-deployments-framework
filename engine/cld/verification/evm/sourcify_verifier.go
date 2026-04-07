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

// sourcifyAPIResponse is the legacy v1 response used by IsVerified (GET /files/any/).
type sourcifyAPIResponse struct {
	Status string `json:"status"`
}

// sourcifyV2SubmitResponse is the response from POST /v2/verify/{chainId}/{address}.
type sourcifyV2SubmitResponse struct {
	VerificationID string `json:"verificationId"`
}

// sourcifyV2JobResponse is the response from GET /v2/verify/{verificationId}.
type sourcifyV2JobResponse struct {
	IsJobCompleted bool             `json:"isJobCompleted"`
	VerificationID string           `json:"verificationId"`
	Contract       *sourcifyV2Match `json:"contract,omitempty"`
	Error          *sourcifyV2Error `json:"error,omitempty"`
}

type sourcifyV2Match struct {
	Match         *string `json:"match"`
	CreationMatch *string `json:"creationMatch"`
	RuntimeMatch  *string `json:"runtimeMatch"`
	ChainID       string  `json:"chainId"`
	Address       string  `json:"address"`
}

type sourcifyV2Error struct {
	CustomCode string `json:"customCode"`
	Message    string `json:"message"`
}

func newSourcifyVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
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
	checkURL := fmt.Sprintf("%s/files/any/%d/%s", v.apiURL, v.chain.EvmChainID, v.address)
	resp, err := doSourcifyRequest[sourcifyAPIResponse](ctx, v.httpClient, http.MethodGet, checkURL, nil)
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

	contractIdentifier := v.metadata.Name
	if !strings.Contains(contractIdentifier, ":") {
		for sourcePath := range v.metadata.Sources {
			contractIdentifier = sourcePath + ":" + v.metadata.Name
			break
		}
	}

	requestData := map[string]any{
		"stdJsonInput": map[string]any{
			"language": v.metadata.Language,
			"sources":  v.metadata.Sources,
			"settings": v.metadata.Settings,
		},
		"compilerVersion":    v.metadata.Version,
		"contractIdentifier": contractIdentifier,
	}

	submitURL := fmt.Sprintf("%s/v2/verify/%d/%s", v.apiURL, v.chain.EvmChainID, v.address)
	submitResp, err := doSourcifyRequest[sourcifyV2SubmitResponse](ctx, v.httpClient, http.MethodPost, submitURL, requestData)
	if err != nil {
		return fmt.Errorf("failed to submit verification: %w", err)
	}

	if submitResp.VerificationID == "" {
		return errors.New("no verification ID returned from sourcify")
	}

	v.lggr.Infof("Verification submitted for %s (id: %s), polling for result...", v, submitResp.VerificationID)

	return v.pollVerificationJob(ctx, submitResp.VerificationID)
}

func (v *sourcifyVerifier) pollVerificationJob(ctx context.Context, verificationID string) error {
	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}

	pollURL := fmt.Sprintf("%s/v2/verify/%s", v.apiURL, verificationID)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollDur):
		}

		jobResp, err := doSourcifyRequest[sourcifyV2JobResponse](ctx, v.httpClient, http.MethodGet, pollURL, nil)
		if err != nil {
			return fmt.Errorf("failed to poll verification status: %w", err)
		}

		if !jobResp.IsJobCompleted {
			v.lggr.Infof("Verification in progress for %s...", v)
			continue
		}

		if jobResp.Error != nil {
			return fmt.Errorf("verification failed: [%s] %s", jobResp.Error.CustomCode, jobResp.Error.Message)
		}

		if jobResp.Contract != nil && jobResp.Contract.Match != nil {
			match := *jobResp.Contract.Match
			if match == "match" || match == "exact_match" {
				v.lggr.Infof("Verification succeeded for %s - %s", v, match)
				return nil
			}
		}

		return errors.New("verification completed but contract was not matched")
	}
}

// doSourcifyRequest sends an HTTP request and decodes the JSON response.
// It accepts any 2xx status code as success.
func doSourcifyRequest[T any](ctx context.Context, client *http.Client, method, reqURL string, body any) (T, error) {
	var empty T
	if client == nil {
		client = http.DefaultClient
	}

	var httpReq *http.Request
	var err error
	if body != nil {
		jsonData, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return empty, fmt.Errorf("failed to marshal JSON: %w", marshalErr)
		}
		httpReq, err = http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(jsonData))
		if err == nil {
			httpReq.Header.Set("Content-Type", "application/json")
		}
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, method, reqURL, nil)
	}
	if err != nil {
		return empty, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return empty, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return empty, fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var apiResp T
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return empty, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp, nil
}
