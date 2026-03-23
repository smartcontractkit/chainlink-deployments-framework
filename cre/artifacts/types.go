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
}

// BinarySource is either a local path to an existing WASM file or an external reference.
type BinarySource struct {
	ExternalRef *ExternalBinaryRef `json:"externalRef,omitempty" yaml:"externalRef,omitempty"`
	LocalPath   string             `json:"localPath,omitempty" yaml:"localPath,omitempty"`
}

// ExternalBinaryRef describes a remote WASM (GitHub release asset or direct URL).
type ExternalBinaryRef struct {
	URL        string `json:"url,omitempty" yaml:"url,omitempty"`
	ReleaseTag string `json:"releaseTag,omitempty" yaml:"releaseTag,omitempty"`
	AssetName  string `json:"assetName,omitempty" yaml:"assetName,omitempty"`
	Repo       string `json:"repo,omitempty" yaml:"repo,omitempty"`
	SHA256     string `json:"sha256,omitempty" yaml:"sha256,omitempty"`
}

// ConfigSource is either a local config file or an external reference.
type ConfigSource struct {
	ExternalRef *ExternalConfigRef `json:"externalRef,omitempty" yaml:"externalRef,omitempty"`
	LocalPath   string             `json:"localPath,omitempty" yaml:"localPath,omitempty"`
}

// ExternalConfigRef describes remote config (GitHub file at ref, or arbitrary URL).
type ExternalConfigRef struct {
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`
	Ref  string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// IsLocal reports whether the binary source is a local filesystem path.
func (b BinarySource) IsLocal() bool {
	return strings.TrimSpace(b.LocalPath) != ""
}

// IsExternal reports whether the binary source uses an external reference.
func (b BinarySource) IsExternal() bool {
	return b.ExternalRef != nil
}

// Validate checks that exactly one of local path or external ref is set.
func (b BinarySource) Validate() error {
	hasLocal := b.IsLocal()
	hasExt := b.IsExternal()
	if hasLocal && hasExt {
		return errors.New("cre: binary: specify either localPath or externalRef, not both")
	}
	if !hasLocal && !hasExt {
		return errors.New("cre: binary: localPath or externalRef is required")
	}
	if hasExt {
		return b.ExternalRef.Validate()
	}
	return nil
}

// Validate enforces one external mode (URL XOR GitHub release) and SHA256 for WASM.
func (e *ExternalBinaryRef) Validate() error {
	if e == nil {
		return errors.New("cre: external binary ref is nil")
	}
	rawURL := strings.TrimSpace(e.URL)
	hasURL := rawURL != ""
	hasRelease := strings.TrimSpace(e.Repo) != "" && strings.TrimSpace(e.ReleaseTag) != "" && strings.TrimSpace(e.AssetName) != ""

	if hasURL && hasRelease {
		return errors.New("cre: external binary: specify either url or repo/releaseTag/assetName, not both")
	}
	if !hasURL && !hasRelease {
		return errors.New("cre: external binary: url or repo/releaseTag/assetName is required")
	}
	if strings.TrimSpace(e.SHA256) == "" {
		return errors.New("cre: external binary: sha256 is required")
	}
	if hasURL {
		if err := validURL(rawURL); err != nil {
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

// IsURL reports whether the url field is non-empty (after [strings.TrimSpace]).
// When true, [ExternalBinaryRef.Validate] requires an absolute http or https URL.
func (e *ExternalBinaryRef) IsURL() bool {
	if e == nil {
		return false
	}
	return strings.TrimSpace(e.URL) != ""
}

// IsGitHubRelease reports whether this ref uses GitHub Releases API resolution.
func (e *ExternalBinaryRef) IsGitHubRelease() bool {
	if e == nil {
		return false
	}
	return strings.TrimSpace(e.Repo) != "" && strings.TrimSpace(e.ReleaseTag) != "" && strings.TrimSpace(e.AssetName) != ""
}

// IsLocal reports whether the config source is a local filesystem path.
func (c ConfigSource) IsLocal() bool {
	return strings.TrimSpace(c.LocalPath) != ""
}

// IsExternal reports whether the config source uses an external reference.
func (c ConfigSource) IsExternal() bool {
	return c.ExternalRef != nil
}

// Validate checks that exactly one of local path or external ref is set.
func (c ConfigSource) Validate() error {
	hasLocal := c.IsLocal()
	hasExt := c.IsExternal()
	if hasLocal && hasExt {
		return errors.New("cre: config: specify either localPath or externalRef, not both")
	}
	if !hasLocal && !hasExt {
		return errors.New("cre: config: localPath or externalRef is required")
	}
	if hasExt {
		return c.ExternalRef.Validate()
	}
	return nil
}

// Validate enforces GitHub file mode XOR URL mode.
func (e *ExternalConfigRef) Validate() error {
	if e == nil {
		return errors.New("cre: external config ref is nil")
	}
	rawURL := strings.TrimSpace(e.URL)
	hasURL := rawURL != ""
	repo := strings.TrimSpace(e.Repo)
	ref := strings.TrimSpace(e.Ref)
	path := strings.TrimSpace(e.Path)
	hasGH := repo != "" && ref != "" && path != ""

	if hasURL && hasGH {
		return errors.New("cre: external config: specify either url or repo/ref/path, not both")
	}
	if !hasURL && !hasGH {
		return errors.New("cre: external config: url or repo/ref/path is required")
	}
	if hasURL {
		if err := validURL(rawURL); err != nil {
			return fmt.Errorf("cre: external config: %w", err)
		}
	}
	if hasGH {
		if _, _, err := parseGitHubRepo(repo); err != nil {
			return err
		}
	}
	return nil
}

// IsURL reports whether the url field is non-empty (after [strings.TrimSpace]).
// When true, [ExternalConfigRef.Validate] requires an absolute http or https URL.
func (e *ExternalConfigRef) IsURL() bool {
	if e == nil {
		return false
	}
	return strings.TrimSpace(e.URL) != ""
}

// IsGitHubFile reports whether this ref uses GitHub Contents API (repo + ref + path).
func (e *ExternalConfigRef) IsGitHubFile() bool {
	if e == nil {
		return false
	}
	return strings.TrimSpace(e.Repo) != "" && strings.TrimSpace(e.Ref) != "" && strings.TrimSpace(e.Path) != ""
}

// Validate validates the workflow bundle.
func (w WorkflowBundle) Validate() error {
	if strings.TrimSpace(w.WorkflowName) == "" {
		return errors.New("cre: workflowName is required")
	}
	if err := w.Binary.Validate(); err != nil {
		return err
	}
	return w.Config.Validate()
}

// ApplyDeployDefaults sets DonFamily from cre when empty (after trim). No-op if w is nil.
func (w *WorkflowBundle) ApplyDeployDefaults(cre cfgenv.CREConfig) {
	if w == nil {
		return
	}
	if strings.TrimSpace(w.DonFamily) != "" {
		return
	}
	w.DonFamily = ""
	if s := strings.TrimSpace(cre.DonFamily); s != "" {
		w.DonFamily = s
	}
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
