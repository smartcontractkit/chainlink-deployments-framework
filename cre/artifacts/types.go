package artifacts

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

// EnvWorkflowDonFamily is the env name for CRE.don_family in CLD config; ApplyDeployDefaults takes cfg.CRE instead.
const EnvWorkflowDonFamily = "CRE_DON_FAMILY"

// WorkflowBundle describes workflow deploy inputs: pre-built WASM and config.
type WorkflowBundle struct {
	WorkflowName string       `json:"workflowName" yaml:"workflowName"`
	Binary       BinarySource `json:"binary" yaml:"binary"`
	Config       ConfigSource `json:"config" yaml:"config"`
	DonFamily    string       `json:"donFamily,omitempty" yaml:"donFamily,omitempty"`
	Attributes   string       `json:"attributes,omitempty" yaml:"attributes,omitempty"`
	Tag          string       `json:"tag,omitempty" yaml:"tag,omitempty"`
}

// Validate trims string fields and validates the workflow bundle.
func (w *WorkflowBundle) Validate() error {
	w.WorkflowName = strings.TrimSpace(w.WorkflowName)
	w.DonFamily = strings.TrimSpace(w.DonFamily)
	if w.WorkflowName == "" {
		return errors.New("cre: workflowName is required")
	}
	if err := w.Binary.Validate(); err != nil {
		return err
	}

	return w.Config.Validate()
}

// ApplyDeployDefaults sets DonFamily from cre when empty. No-op if w is nil.
func (w *WorkflowBundle) ApplyDeployDefaults(cre cfgenv.CREConfig) {
	if w == nil {
		return
	}
	w.DonFamily = strings.TrimSpace(w.DonFamily)
	if w.DonFamily != "" {
		return
	}
	if s := strings.TrimSpace(cre.DonFamily); s != "" {
		w.DonFamily = s
	}
}

// BinarySource is either a local path to an existing WASM file or an external reference.
// Create via [NewBinarySourceLocal], [NewBinarySourceURL], or [NewBinarySourceGitHubRelease],
// or unmarshal from JSON/YAML and call [BinarySource.Validate].
type BinarySource struct {
	ExternalRef *externalBinaryRef `json:"externalRef,omitempty" yaml:"externalRef,omitempty"`
	LocalPath   string             `json:"localPath,omitempty" yaml:"localPath,omitempty"`
}

// NewBinarySourceLocal returns a BinarySource for a local .wasm file at localPath.
func NewBinarySourceLocal(localPath string) BinarySource {
	return BinarySource{LocalPath: strings.TrimSpace(localPath)}
}

// NewBinarySourceURL returns a validated BinarySource for a direct URL download with SHA-256 verification.
func NewBinarySourceURL(rawURL, sha256 string) (BinarySource, error) {
	ref := &externalBinaryRef{URL: rawURL, SHA256: sha256}
	if err := ref.validate(); err != nil {
		return BinarySource{}, err
	}

	return BinarySource{ExternalRef: ref}, nil
}

// NewBinarySourceGitHubRelease returns a validated BinarySource for a GitHub release asset.
func NewBinarySourceGitHubRelease(repo, releaseTag, assetName, sha256 string) (BinarySource, error) {
	ref := &externalBinaryRef{Repo: repo, ReleaseTag: releaseTag, AssetName: assetName, SHA256: sha256}
	if err := ref.validate(); err != nil {
		return BinarySource{}, err
	}

	return BinarySource{ExternalRef: ref}, nil
}

// IsLocal reports whether the binary source is a local filesystem path.
func (b *BinarySource) IsLocal() bool {
	return strings.TrimSpace(b.LocalPath) != ""
}

// IsExternal reports whether the binary source uses an external reference.
func (b *BinarySource) IsExternal() bool {
	return b.ExternalRef != nil
}

// Validate trims LocalPath and checks that exactly one of local path or external ref is set.
func (b *BinarySource) Validate() error {
	b.LocalPath = strings.TrimSpace(b.LocalPath)
	hasLocal := b.IsLocal()
	hasExt := b.IsExternal()
	if hasLocal && hasExt {
		return errors.New("cre: binary: specify either localPath or externalRef, not both")
	}
	if !hasLocal && !hasExt {
		return errors.New("cre: binary: localPath or externalRef is required")
	}
	if hasExt {
		return b.ExternalRef.validate()
	}

	return nil
}

// externalBinaryRef describes a remote WASM (GitHub release asset or direct URL).
// Constructed by [NewBinarySourceURL] or [NewBinarySourceGitHubRelease], or
// unmarshaled via the parent [BinarySource] and validated with [BinarySource.Validate].
type externalBinaryRef struct {
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	ReleaseTag string `json:"releaseTag,omitempty" yaml:"releaseTag,omitempty"`
	AssetName  string `json:"assetName,omitempty" yaml:"assetName,omitempty"`
	Repo       string `json:"repo,omitempty" yaml:"repo,omitempty"`
	SHA256     string `json:"sha256,omitempty" yaml:"sha256,omitempty"`
}

// validate trims all string fields, then enforces one external mode (URL XOR GitHub release) and SHA256.
// The caller guarantees e is non-nil (BinarySource.Validate checks IsExternal first).
func (e *externalBinaryRef) validate() error {
	e.URL = strings.TrimSpace(e.URL)
	e.Repo = strings.TrimSpace(e.Repo)
	e.ReleaseTag = strings.TrimSpace(e.ReleaseTag)
	e.AssetName = strings.TrimSpace(e.AssetName)
	e.SHA256 = strings.TrimSpace(e.SHA256)

	hasURL := e.URL != ""
	hasRelease := e.Repo != "" && e.ReleaseTag != "" && e.AssetName != ""

	if hasURL && hasRelease {
		return errors.New("cre: external binary: specify either url or repo/releaseTag/assetName, not both")
	}
	if !hasURL && !hasRelease {
		return errors.New("cre: external binary: url or repo/releaseTag/assetName is required")
	}
	if e.SHA256 == "" {
		return errors.New("cre: external binary: sha256 is required")
	}
	if hasURL {
		if err := validURL(e.URL); err != nil {
			return fmt.Errorf("cre: external binary: %w", err)
		}
	}
	if hasRelease {
		if _, _, err := parseGitHubRepo(e.Repo); err != nil {
			return err
		}
	}

	return nil
}

// isURL reports whether the url field is non-empty. Caller guarantees e is non-nil.
func (e *externalBinaryRef) isURL() bool {
	return strings.TrimSpace(e.URL) != ""
}

// isGitHubRelease reports whether this ref uses GitHub Releases API resolution.
// Caller guarantees e is non-nil.
func (e *externalBinaryRef) isGitHubRelease() bool {
	return strings.TrimSpace(e.Repo) != "" && strings.TrimSpace(e.ReleaseTag) != "" && strings.TrimSpace(e.AssetName) != ""
}

// ConfigSource is either a local config file or an external reference.
// Create via [NewConfigSourceLocal], [NewConfigSourceURL], or [NewConfigSourceGitHub],
// or unmarshal from JSON/YAML and call [ConfigSource.Validate].
type ConfigSource struct {
	ExternalRef *externalConfigRef `json:"externalRef,omitempty" yaml:"externalRef,omitempty"`
	LocalPath   string             `json:"localPath,omitempty" yaml:"localPath,omitempty"`
}

// NewConfigSourceLocal returns a ConfigSource for a local config file at localPath.
func NewConfigSourceLocal(localPath string) ConfigSource {
	return ConfigSource{LocalPath: strings.TrimSpace(localPath)}
}

// NewConfigSourceURL returns a validated ConfigSource for a direct URL download.
func NewConfigSourceURL(rawURL string) (ConfigSource, error) {
	ref := &externalConfigRef{URL: rawURL}
	if err := ref.validate(); err != nil {
		return ConfigSource{}, err
	}

	return ConfigSource{ExternalRef: ref}, nil
}

// NewConfigSourceGitHub returns a validated ConfigSource for a GitHub file via the Contents API.
func NewConfigSourceGitHub(repo, ref, path string) (ConfigSource, error) {
	r := &externalConfigRef{Repo: repo, Ref: ref, Path: path}
	if err := r.validate(); err != nil {
		return ConfigSource{}, err
	}

	return ConfigSource{ExternalRef: r}, nil
}

// IsLocal reports whether the config source is a local filesystem path.
func (c *ConfigSource) IsLocal() bool {
	return strings.TrimSpace(c.LocalPath) != ""
}

// IsExternal reports whether the config source uses an external reference.
func (c *ConfigSource) IsExternal() bool {
	return c.ExternalRef != nil
}

// Validate trims LocalPath and checks that exactly one of local path or external ref is set.
func (c *ConfigSource) Validate() error {
	c.LocalPath = strings.TrimSpace(c.LocalPath)
	hasLocal := c.IsLocal()
	hasExt := c.IsExternal()
	if hasLocal && hasExt {
		return errors.New("cre: config: specify either localPath or externalRef, not both")
	}
	if !hasLocal && !hasExt {
		return errors.New("cre: config: localPath or externalRef is required")
	}
	if hasExt {
		return c.ExternalRef.validate()
	}

	return nil
}

// externalConfigRef describes remote config (GitHub file at ref, or arbitrary URL).
// Constructed by [NewConfigSourceURL] or [NewConfigSourceGitHub], or
// unmarshaled via the parent [ConfigSource] and validated with [ConfigSource.Validate].
type externalConfigRef struct {
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// validate trims all string fields (normalizing Path), then enforces GitHub file mode XOR URL mode.
// The caller guarantees e is non-nil (ConfigSource.Validate checks IsExternal first).
func (e *externalConfigRef) validate() error {
	e.URL = strings.TrimSpace(e.URL)
	e.Repo = strings.TrimSpace(e.Repo)
	e.Ref = strings.TrimSpace(e.Ref)
	e.Path = normalizeGitHubConfigPath(e.Path)

	if e.Repo != "" && e.Ref != "" && e.Path == "" {
		return errors.New("cre: external config: path must name a non-empty file path within the repository")
	}

	hasURL := e.URL != ""
	hasGH := e.Repo != "" && e.Ref != "" && e.Path != ""

	if hasURL && hasGH {
		return errors.New("cre: external config: specify either url or repo/ref/path, not both")
	}
	if !hasURL && !hasGH {
		return errors.New("cre: external config: url or repo/ref/path is required")
	}
	if hasURL {
		if err := validURL(e.URL); err != nil {
			return fmt.Errorf("cre: external config: %w", err)
		}
	}
	if hasGH {
		if _, _, err := parseGitHubRepo(e.Repo); err != nil {
			return err
		}
	}

	return nil
}

// isURL reports whether the url field is non-empty. Caller guarantees e is non-nil.
func (e *externalConfigRef) isURL() bool {
	return strings.TrimSpace(e.URL) != ""
}

// isGitHubFile reports whether this ref uses GitHub Contents API (repo + ref + path).
// Caller guarantees e is non-nil.
func (e *externalConfigRef) isGitHubFile() bool {
	return strings.TrimSpace(e.Repo) != "" && strings.TrimSpace(e.Ref) != "" && normalizeGitHubConfigPath(e.Path) != ""
}

// helpers

// normalizeGitHubConfigPath trims surrounding space and leading slashes so the path is suitable
// for the GitHub Contents API.
func normalizeGitHubConfigPath(p string) string {
	return strings.TrimLeft(strings.TrimSpace(p), "/")
}

// validURL ensures raw is parseable and uses an http scheme with a non-empty host.
func validURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		if u.Scheme == "" {
			return errors.New("url must be an absolute http or https URL with a host")
		}

		return fmt.Errorf("url scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("url must be an absolute http or https URL with a host")
	}

	return nil
}

func parseGitHubRepo(repo string) (owner, name string, err error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", "", errors.New("cre: repo is empty")
	}
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cre: repo must be owner/name, got %q", repo)
	}

	return parts[0], parts[1], nil
}
