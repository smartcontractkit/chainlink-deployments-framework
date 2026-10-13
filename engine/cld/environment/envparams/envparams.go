// Package envparams resolves the BuildParams document that lets a command build
// its environment without a domain directory. It is shared with the mcms CLI.
package envparams

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

const (
	// FlagName carries a BuildParams YAML document.
	FlagName = "env-build-params"

	// EnvVar carries the same document when the flag is not set. Prefer it, as
	// the document holds secrets that should not be on a command line.
	EnvVar = "MCMS_ENV_BUILD_PARAMS"

	// DomainFlagName and EnvFlagName are the flags the document replaces.
	DomainFlagName = "domain"
	EnvFlagName    = "environment"
)

// ErrFlagNotAllowed reports a flag combined with build params that set the same thing.
var ErrFlagNotAllowed = errors.New("cannot be combined with build params")

// ErrEmptyDocument reports the build params flag passed empty.
var ErrEmptyDocument = errors.New("document is empty")

// RegisterFlag adds the build params flag to flags.
func RegisterFlag(flags *pflag.FlagSet) {
	flags.String(FlagName, "",
		"YAML document to build the environment from instead of reading a domain directory from disk. "+
			"Prefer supplying it as "+EnvVar+", since it carries deployer keys and catalog credentials.")
}

// Resolve returns the build params to use, or nil when none were supplied and
// the command should load from disk.
//
// The flag wins over EnvVar, and an explicitly empty flag is an error rather
// than a fall back. The document owns the domain and environment, so passing
// them as flags too is rejected. They are written back to the flags so required
// flag validation passes.
func Resolve(flags *pflag.FlagSet, getenv func(string) string) (*cldfenv.BuildParams, error) {
	doc := lookupString(flags, FlagName)
	if doc == "" {
		if flagChanged(flags, FlagName) {
			return nil, fmt.Errorf("--%s: %w", FlagName, ErrEmptyDocument)
		}

		doc = getenv(EnvVar)
	}
	if doc == "" {
		return nil, nil //nolint:nilnil // no params means fall back to disk
	}

	for _, name := range []string{DomainFlagName, EnvFlagName} {
		if flagChanged(flags, name) {
			return nil, fmt.Errorf("--%s %w", name, ErrFlagNotAllowed)
		}
	}

	params, err := Parse([]byte(doc))
	if err != nil {
		return nil, err
	}

	if params.Domain == "" {
		return nil, errors.New(`build params must set "domain"`)
	}
	if params.Environment == "" {
		return nil, errors.New(`build params must set "environment"`)
	}

	if err = setString(flags, DomainFlagName, params.Domain); err != nil {
		return nil, err
	}
	if err = setString(flags, EnvFlagName, params.Environment); err != nil {
		return nil, err
	}

	return &params, nil
}

// Parse decodes a BuildParams YAML document, rejecting unknown keys.
func Parse(data []byte) (cldfenv.BuildParams, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var params cldfenv.BuildParams
	if err := dec.Decode(&params); err != nil {
		if errors.Is(err, io.EOF) {
			return cldfenv.BuildParams{}, errors.New("failed to decode build params: document is empty")
		}

		return cldfenv.BuildParams{}, fmt.Errorf("failed to decode build params: %w", err)
	}

	return params, nil
}

// flagChanged reports whether the flag was explicitly passed, even if empty.
func flagChanged(flags *pflag.FlagSet, name string) bool {
	if flags == nil {
		return false
	}

	flag := flags.Lookup(name)

	return flag != nil && flag.Changed
}

// lookupString reads a string flag, treating an unregistered one as empty.
func lookupString(flags *pflag.FlagSet, name string) string {
	if flags == nil {
		return ""
	}

	flag := flags.Lookup(name)
	if flag == nil || flag.Value == nil {
		return ""
	}

	return flag.Value.String()
}

// setString writes a value to a flag, skipping an unregistered one.
func setString(flags *pflag.FlagSet, name, value string) error {
	if value == "" || flags == nil || flags.Lookup(name) == nil {
		return nil
	}

	if err := flags.Set(name, value); err != nil {
		return fmt.Errorf("failed to apply build params %s to --%s: %w", name, name, err)
	}

	return nil
}
