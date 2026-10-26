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

// LoadOption configures rpc manifest loading behavior.
type LoadOption func(*loadOptions)

type loadOptions struct {
	lggr logger.Logger
}

// WithLogger logs warnings for skipped or duplicate manifest entries. A nil logger is ignored,
// leaving the default no-op logger in place, so callers can't accidentally cause a nil-interface
// panic by passing one.
func WithLogger(lggr logger.Logger) LoadOption {
	return func(opts *loadOptions) {
		if lggr == nil {
			return
		}
		opts.lggr = lggr
	}
}

// Manifest maps chain selectors to RPC proxy manifest entries.
type Manifest struct {
	chains map[uint64]ChainEntry
}

// Load loads an RPC manifest from a URL, file:// URI, or local filesystem path.
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
		return nil, fmt.Errorf("unmarshal rpc manifest: %w", err)
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
		return 0, fmt.Errorf("parse chain selector %q: %w", raw, err)
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

// readFileBounded reads a local manifest file, rejecting anything larger than maxManifestBytes
// so a misconfigured source can't trigger an unbounded allocation.
func readFileBounded(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read rpc manifest file: %w", err)
	}
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("rpc manifest file %s exceeds max size of %d bytes", path, maxManifestBytes)
	}

	return data, nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}

		path = filepath.Join(home, path[2:])
	}

	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve manifest path: %w", err)
		}
		path = filepath.Join(cwd, path)
	}

	return path, nil
}

func urlPathFromFileURI(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse file URI: %w", err)
	}
	if u.Host != "" && u.Host != "localhost" {
		return "", fmt.Errorf("file URI must not specify a host: %q", raw)
	}
	if u.Path == "" {
		return "", errors.New("file URI has empty path")
	}

	path, err := url.PathUnescape(u.EscapedPath())
	if err != nil {
		return "", fmt.Errorf("unescape file URI path: %w", err)
	}

	return filepath.FromSlash(path), nil
}

func fetchHTTP(ctx context.Context, manifestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}

	client := &http.Client{Timeout: manifestHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch rpc manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("fetch rpc manifest from %s: status %d: %s", manifestURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read rpc manifest body: %w", err)
	}
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("rpc manifest response from %s exceeds max size of %d bytes", manifestURL, maxManifestBytes)
	}

	return data, nil
}
