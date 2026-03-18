package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const sourcifyServerURL = "https://sourcify.dev/server"

func newSourcifyVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnSourcify(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by Sourcify", cfg.Chain.EvmChainID)
	}

	baseURL := sourcifyServerURL
	if cfg.Network.BlockExplorer.URL != "" {
		baseURL = strings.TrimSuffix(cfg.Network.BlockExplorer.URL, "/")
	}

	return &sourcifyVerifier{
		chain:        cfg.Chain,
		baseURL:      baseURL,
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
	baseURL      string
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

func (v *sourcifyVerifier) client() *http.Client {
	if v.httpClient != nil {
		return v.httpClient
	}
	return http.DefaultClient
}

// IsVerified checks whether the contract is already verified on Sourcify via
// GET /v2/contract/{chainId}/{address}.
// A 200 with a non-null "match" field means verified; 404 means not verified.
func (v *sourcifyVerifier) IsVerified(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/v2/contract/%d/%s", v.baseURL, v.chain.EvmChainID, v.address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := v.client().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return false, fmt.Errorf("sourcify IsVerified: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var result sourcifyContractResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to decode sourcify response: %w", err)
	}

	if result.Match != nil {
		v.lggr.Infof("Contract %s is already verified on Sourcify (match=%s)", v.address, *result.Match)
		return true, nil
	}

	return false, nil
}

// Verify submits the contract for verification on Sourcify and polls until completion.
func (v *sourcifyVerifier) Verify(ctx context.Context) error {
	verified, err := v.IsVerified(ctx)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if verified {
		v.lggr.Infof("%s is already verified", v.String())
		return nil
	}

	verificationID, err := v.submitVerification(ctx)
	if err != nil {
		return err
	}

	v.lggr.Infof("Verification submitted for %s, verificationId=%s", v.String(), verificationID)

	return v.pollVerification(ctx, verificationID)
}

func (v *sourcifyVerifier) submitVerification(ctx context.Context) (string, error) {
	stdJsonInput := map[string]any{
		"language": v.metadata.Language,
		"settings": v.metadata.Settings,
		"sources":  v.metadata.Sources,
	}

	contractID := v.buildContractIdentifier()

	reqBody := sourcifyVerifyRequest{
		StdJsonInput:       stdJsonInput,
		CompilerVersion:    v.metadata.Version,
		ContractIdentifier: contractID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal verify request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/verify/%d/%s", v.baseURL, v.chain.EvmChainID, v.address)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read verify response: %w", err)
	}

	if resp.StatusCode == http.StatusConflict {
		v.lggr.Infof("%s is already verified on Sourcify (409 conflict)", v.String())
		return "", nil
	}

	if resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("sourcify verify: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var verifyResp sourcifyVerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return "", fmt.Errorf("failed to decode verify response: %w", err)
	}

	return verifyResp.VerificationID, nil
}

func (v *sourcifyVerifier) pollVerification(ctx context.Context, verificationID string) error {
	if verificationID == "" {
		return nil
	}

	pollDur := v.pollInterval
	if pollDur <= 0 {
		pollDur = 5 * time.Second
	}

	url := fmt.Sprintf("%s/v2/verify/%s", v.baseURL, verificationID)

	for range maxVerificationPollAttempts {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := v.client().Do(req)
		if err != nil {
			return fmt.Errorf("failed to poll verification status: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read poll response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("sourcify poll: unexpected status %d: %s", resp.StatusCode, string(body))
		}

		var status sourcifyJobStatus
		if err := json.Unmarshal(body, &status); err != nil {
			return fmt.Errorf("failed to decode poll response: %w", err)
		}

		if !status.IsJobCompleted {
			v.lggr.Infof("Verification in progress for %s, checking again in %s", v.String(), pollDur)
			select {
			case <-time.After(pollDur):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		if status.Error != nil {
			return fmt.Errorf("sourcify verification failed: %s - %s", status.Error.CustomCode, status.Error.Message)
		}

		if status.Contract != nil && status.Contract.Match != nil {
			v.lggr.Infof("Verification succeeded for %s (match=%s)", v.String(), *status.Contract.Match)
			return nil
		}

		return fmt.Errorf("sourcify verification completed but contract match is empty")
	}

	return fmt.Errorf("verification timed out after %d attempts", maxVerificationPollAttempts)
}

// buildContractIdentifier constructs "path/to/Contract.sol:ContractName" from metadata.
func (v *sourcifyVerifier) buildContractIdentifier() string {
	name := v.metadata.Name
	if name == "" {
		name = v.contractType
	}

	suffix := "/" + name + ".sol"
	for source := range v.metadata.Sources {
		if strings.HasSuffix(source, suffix) {
			return source + ":" + name
		}
	}

	// Fall back: use the first source key with an exact filename match, or just the first key.
	for source := range v.metadata.Sources {
		if strings.HasSuffix(source, name+".sol") {
			return source + ":" + name
		}
	}
	for source := range v.metadata.Sources {
		return source + ":" + name
	}

	return name
}

// Sourcify API types

type sourcifyVerifyRequest struct {
	StdJsonInput       map[string]any `json:"stdJsonInput"`
	CompilerVersion    string         `json:"compilerVersion"`
	ContractIdentifier string         `json:"contractIdentifier"`
}

type sourcifyVerifyResponse struct {
	VerificationID string `json:"verificationId"`
}

type sourcifyContractResponse struct {
	Match *string `json:"match"`
}

type sourcifyJobStatus struct {
	IsJobCompleted bool                     `json:"isJobCompleted"`
	VerificationID string                   `json:"verificationId"`
	Error          *sourcifyJobError        `json:"error"`
	Contract       *sourcifyContractResponse `json:"contract"`
}

type sourcifyJobError struct {
	CustomCode string `json:"customCode"`
	Message    string `json:"message"`
}
