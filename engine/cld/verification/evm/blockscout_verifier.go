package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type blockscoutVerifyRequest struct {
	AddressHash      string `json:"addressHash"`
	CompilerVersion  string `json:"compilerVersion"`
	ContractSource   string `json:"contractSourceCode"`
	Name             string `json:"name"`
	OptimizationUsed bool   `json:"optimization"`
}

func newBlockscoutVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnBlockscout(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the Blockscout API", cfg.Chain.EvmChainID)
	}
	apiURL := cfg.Network.BlockExplorer.URL
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
	lggr         logger.Logger
	httpClient   *http.Client
}

func (v *blockscoutVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.contractType, v.version, v.address, v.chain.Name)
}

func (v *blockscoutVerifier) IsVerified(ctx context.Context) (bool, error) {
	u, err := url.Parse(v.apiURL)
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
		return false, nil
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
	sourceCode, err := v.metadata.SourceCode()
	if err != nil {
		return fmt.Errorf("failed to get source code: %w", err)
	}
	verifyReq := blockscoutVerifyRequest{
		AddressHash:      v.address,
		CompilerVersion:  v.metadata.Version,
		ContractSource:   sourceCode,
		Name:             v.contractType,
		OptimizationUsed: true,
	}
	u, err := url.Parse(v.apiURL)
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	u.Path = "/api"
	q := u.Query()
	q.Set("module", "contract")
	q.Set("action", "verify")
	u.RawQuery = q.Encode()

	jsonData, err := json.Marshal(verifyReq)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http error - status=%d body=%s", resp.StatusCode, string(body))
	}
	v.lggr.Infof("Verification submitted successfully for contract %s", v.address)

	return nil
}
