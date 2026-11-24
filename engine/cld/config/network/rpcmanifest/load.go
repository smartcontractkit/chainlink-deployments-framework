package rpcmanifest

import (
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

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	manifestHTTPTimeout = 30 * time.Second
	maxManifestBytes    = 10 << 20 // 10 MiB
)

// LoadOption configures how an RPC manifest is loaded.
type LoadOption func(*loadOptions)

type loadOptions struct {
	lggr logger.Logger
}

// WithLogger sets the logger used to warn about skipped manifest entries.
func WithLogger(lggr logger.Logger) LoadOption {
	return func(opts *loadOptions) {
		opts.lggr = lggr
	}
}

// Manifest is an RPC proxy manifest indexed by chain selector.
type Manifest struct {
	chains map[uint64]chainEntry
	// baseURL is the origin of the manifest URL, which also serves the per-provider RPC paths.
	baseURL string
}

// Load fetches an RPC manifest from an HTTP(S) URL. Entries with an invalid or duplicate chain
// selector are skipped.
func Load(ctx context.Context, manifestURL string, opts ...LoadOption) (*Manifest, error) {
	cfg := loadOptions{lggr: logger.Nop()}
	for _, opt := range opts {
		opt(&cfg)
	}

	// url.Parse lowercases the scheme. The URL is left out of errors as it may carry credentials.
	u, err := url.Parse(strings.TrimSpace(manifestURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse rpc manifest url: %w", redactURLError(err))
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("rpc manifest url must be an http(s) url with a host")
	}

	data, err := fetchHTTP(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var entries []chainEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rpc manifest: %w", err)
	}

	chains := make(map[uint64]chainEntry, len(entries))
	for _, entry := range entries {
		selector, err := parseChainSelector(entry.ChainSelector)
		if err != nil {
			cfg.lggr.Warnw("skipping rpc manifest chain entry with invalid chain selector",
				"chain_name", entry.Names.Canonical,
				"chain_selector", entry.ChainSelector,
				"error", err,
			)

			continue
		}
		if _, exists := chains[selector]; exists {
			cfg.lggr.Warnw("skipping duplicate rpc manifest chain entry",
				"chain_name", entry.Names.Canonical,
				"chain_selector", selector,
			)

			continue
		}
		chains[selector] = entry
	}

	baseURL := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()

	return &Manifest{chains: chains, baseURL: baseURL}, nil
}

func parseChainSelector(raw string) (uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, errors.New("empty chain selector")
	}

	selector, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse chain selector %q: %w", raw, err)
	}

	return selector, nil
}

func fetchHTTP(ctx context.Context, manifestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc manifest request: %w", redactURLError(err))
	}

	client := &http.Client{Timeout: manifestHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rpc manifest: %w", redactURLError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch rpc manifest: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read rpc manifest body: %w", err)
	}
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("rpc manifest response exceeds max size of %d bytes", maxManifestBytes)
	}

	return data, nil
}

// redactURLError strips the request URL from a *url.Error: the manifest URL may carry credentials
// (userinfo or query tokens) that must not reach logs.
func redactURLError(err error) error {
	var uerr *url.Error
	if errors.As(err, &uerr) {
		return uerr.Err
	}

	return err
}
