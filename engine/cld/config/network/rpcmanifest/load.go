package rpcmanifest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	chains map[uint64]ChainEntry
}

// Load loads an RPC manifest from an HTTP(S) URL, file:// URI, or local filesystem path. Entries
// with an invalid or duplicate chain selector are skipped.
func Load(ctx context.Context, source string, opts ...LoadOption) (*Manifest, error) {
	cfg := loadOptions{lggr: logger.Nop()}
	for _, opt := range opts {
		opt(&cfg)
	}

	data, err := readSource(ctx, source)
	if err != nil {
		return nil, err
	}

	var entries []ChainEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rpc manifest: %w", err)
	}

	chains := make(map[uint64]ChainEntry, len(entries))
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

	return &Manifest{chains: chains}, nil
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

// Chain returns the manifest entry for a chain selector, if present.
func (m *Manifest) Chain(selector uint64) (ChainEntry, bool) {
	entry, ok := m.chains[selector]

	return entry, ok
}

func readSource(ctx context.Context, source string) ([]byte, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, errors.New("rpc manifest source is empty")
	}

	if strings.HasPrefix(source, "file://") {
		path, err := urlPathFromFileURI(source)
		if err != nil {
			return nil, err
		}

		return readFileBounded(path)
	}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return fetchHTTP(ctx, source)
	}

	path, err := expandPath(source)
	if err != nil {
		return nil, err
	}

	return readFileBounded(path)
}

func readFileBounded(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open rpc manifest file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read rpc manifest file: %w", err)
	}
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("rpc manifest file %s exceeds max size of %d bytes", path, maxManifestBytes)
	}

	return data, nil
}

// expandPath expands a leading "~/" to the user's home directory.
func expandPath(path string) (string, error) {
	rest, ok := strings.CutPrefix(path, "~/")
	if !ok {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve home directory: %w", err)
	}

	return filepath.Join(home, rest), nil
}

func urlPathFromFileURI(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("failed to parse file URI: %w", err)
	}
	if u.Host != "" && u.Host != "localhost" {
		return "", fmt.Errorf("file URI must not specify a host: %q", raw)
	}
	if u.Path == "" {
		return "", errors.New("file URI has empty path")
	}

	return filepath.FromSlash(u.Path), nil
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
