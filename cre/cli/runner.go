package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	fcre "github.com/smartcontractkit/chainlink-deployments-framework/cre"
)

const defaultBinary = "cre"

// envCREAPIKey is the environment variable name the CRE CLI reads for API key authentication.
const envCREAPIKey = "CRE_API_KEY" //nolint:gosec // G101: env var name, not a secret value

// CLIRunnerOption configures a [cliRunner] from [NewCLIRunner].
type CLIRunnerOption func(*cliRunner)

// namedAPIKeys is the parsed form of CRE_API_KEY when it carries multiple named API keys
type namedAPIKeys map[string]string

// sortedNames returns the configured names in lexical order.
func (n namedAPIKeys) sortedNames() []string {
	names := make([]string, 0, len(n))
	for name := range n {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// cliRunner runs the CRE CLI via os/exec. Run executes the binary and captures stdout/stderr.
// It implements [fcre.CLIRunner].
type cliRunner struct {
	binaryPath string
	// rawAPIKey preserves the original value passed to NewCLIRunner verbatim,
	// regardless of which mode (single-key or named-keys) was selected.
	rawAPIKey string
	// apiKey is the resolved single API key value used by Run. In named-keys
	// mode it is empty until a name is selected via WithNamedAPIKey, which
	// returns a clone with apiKey populated from apiKeysByName.
	apiKey string
	// apiKeysByName is populated only when CRE_API_KEY decoded as a JSON object
	// of {name: apiKey} entries. nil in single-key mode.
	apiKeysByName            namedAPIKeys
	defaultContextRegistries []fcre.ContextRegistryEntry
	// Stdout, if set, receives a real-time copy of the process stdout while it runs.
	Stdout io.Writer
	// Stderr, if set, receives a real-time copy of the process stderr while it runs.
	Stderr io.Writer
}

var _ fcre.CLIRunner = (*cliRunner)(nil)

// NewCLIRunner returns a [cliRunner] for the given binary path and API key.
// An empty binaryPath defaults to "cre" (resolved via PATH).
func NewCLIRunner(binaryPath string, apiKey string, opts ...CLIRunnerOption) *cliRunner {
	if binaryPath == "" {
		binaryPath = defaultBinary
	}

	// Default to real-time terminal streaming so CLI output preserves original
	// formatting (newlines/indentation) during manual durable-pipeline runs.
	r := &cliRunner{
		binaryPath: binaryPath,
		rawAPIKey:  apiKey,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	if named, ok := parseNamedAPIKeys(r.rawAPIKey); ok {
		r.apiKeysByName = named
	} else {
		r.apiKey = apiKey
	}
	for _, o := range opts {
		o(r)
	}

	return r
}

// parseNamedAPIKeys decodes raw as a JSON object of {name: apiKey} entries.
// Returns (map, true) only when raw is a non-empty JSON object with no empty
// keys or values. Any other shape — non-JSON values, JSON arrays/strings/
// numbers, empty objects, or entries with empty values — returns (nil, false)
func parseNamedAPIKeys(raw string) (namedAPIKeys, bool) {
	var keys namedAPIKeys
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, false
	}
	if len(keys) == 0 {
		return nil, false
	}
	for name, value := range keys {
		if name == "" || value == "" {
			return nil, false
		}
	}

	return keys, true
}

// WithNamedAPIKey returns a clone of this runner that uses the API key
// registered under name. the clone preserves the underlying name->key map so further WithNamedAPIKey
// calls remain valid.
func (r cliRunner) WithNamedAPIKey(name string) (fcre.CLIRunner, error) {

	if len(r.apiKeysByName) == 0 {
		return nil, errors.New("cre cli: runner is not configured with named API keys")
	}
	key, ok := r.apiKeysByName[name]
	if !ok {
		return nil, fmt.Errorf("cre cli: API key %q not configured (available: %v)", name, r.apiKeysByName.sortedNames())
	}
	r.apiKey = key

	return &r, nil
}

// WithContextRegistries sets domain-level registry entries for CRE context.yaml generation
// (e.g. loaded from domain.yaml cre.default_registries by engine/cld/environment.Load).
func WithContextRegistries(registries []fcre.ContextRegistryEntry) CLIRunnerOption {
	return func(r *cliRunner) {
		r.defaultContextRegistries = append([]fcre.ContextRegistryEntry{}, registries...)
	}
}

// WithOutputWriters overrides real-time streaming writers.
// Pass nil for either stream to disable live output for that stream.
func WithOutputWriters(stdout io.Writer, stderr io.Writer) CLIRunnerOption {
	return func(r *cliRunner) {
		r.Stdout = stdout
		r.Stderr = stderr
	}
}

// ContextRegistries returns a copy of domain defaults attached to this runner, or nil if none.
func (r *cliRunner) ContextRegistries() []fcre.ContextRegistryEntry {
	if r == nil || len(r.defaultContextRegistries) == 0 {
		return nil
	}

	return append([]fcre.ContextRegistryEntry{}, r.defaultContextRegistries...)
}

// envForCRECLI returns the full environment for the subprocess: we copy os.Environ() so PATH and
// other inherited vars stay. We strip any existing CRE_API_KEY= when api key is provided
// so we do not duplicate the key;
func envForCRECLI(apiKey string, extraEnv map[string]string) []string {
	env := os.Environ()
	if apiKey == "" && len(extraEnv) == 0 {
		return append([]string{}, env...)
	}

	out := make([]string, 0, len(env)+1+len(extraEnv))
	excludedPrefixes := make([]string, 0, 1+len(extraEnv))
	if apiKey != "" {
		excludedPrefixes = append(excludedPrefixes, envCREAPIKey+"=")
	}
	for k := range extraEnv {
		excludedPrefixes = append(excludedPrefixes, k+"=")
	}

	for _, e := range env {
		keep := true
		for _, p := range excludedPrefixes {
			if strings.HasPrefix(e, p) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, e)
		}
	}

	if apiKey != "" {
		out = append(out, envCREAPIKey+"="+apiKey)
	}
	for k, v := range extraEnv {
		// Keep runner-configured API key when present.
		if apiKey != "" && k == envCREAPIKey {
			continue
		}
		out = append(out, k+"="+v)
	}

	return out
}

// Run executes the binary and captures stdout and stderr. Exit code 0 returns (res, nil);
// exit code != 0 returns (res, *fcre.ExitError) so callers get both result and error.
// CLI invocation failures (binary not found, context canceled) return (nil, err).
func (r *cliRunner) Run(ctx context.Context, env map[string]string, args ...string) (*fcre.CallResult, error) {
	if r.apiKey == "" && len(r.apiKeysByName) > 0 {
		return nil, fmt.Errorf("cre cli: named API keys configured but no name selected; call CLIRunner.WithNamedAPIKey(<name>) first (available: %v)", r.apiKeysByName.sortedNames())
	}

	//nolint:gosec // G204: This is intentional - we're running a CLI tool with user-provided arguments.
	// The binary path is controlled via configuration, and args are expected to be user-provided CLI arguments.
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)

	// API key and per-invocation env vars are set on the child's environment only.
	if r.apiKey != "" || len(env) > 0 {
		cmd.Env = envForCRECLI(r.apiKey, env)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = wrapWriter(&stdout, r.Stdout)
	cmd.Stderr = wrapWriter(&stderr, r.Stderr)

	err := cmd.Run()
	res := &fcre.CallResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.ExitCode = exitErr.ExitCode()
			return res, &fcre.ExitError{
				ExitCode: res.ExitCode,
				Stdout:   res.Stdout,
				Stderr:   res.Stderr,
			}
		}

		return nil, err
	}

	return res, nil
}

func wrapWriter(buf *bytes.Buffer, stream io.Writer) io.Writer {
	if stream == nil {
		return buf
	}

	return io.MultiWriter(buf, stream)
}
