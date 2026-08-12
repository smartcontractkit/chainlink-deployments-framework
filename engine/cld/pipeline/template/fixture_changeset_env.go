package template

import (
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// envInputFixtureConfig is the config struct for envInputFixtureChangeset.
// It is used by the golden test to verify WithEnvInput path output, where the
// input type comes from the changeset's generic type parameter C rather than
// from a config resolver's function signature.
//
// It MUST live in a regular .go file (not a _test.go file) because
// packages.Load does not load _test.go files by default.
type envInputFixtureConfig struct {
	// FeedURL is the HTTP endpoint the workflow polls for data.
	FeedURL string `yaml:"feedURL" json:"feedURL"`

	// PollIntervalSec is the interval between polls in seconds.
	// Must be a positive integer.
	PollIntervalSec int `yaml:"pollIntervalSec" json:"pollIntervalSec"`

	// Enabled controls whether the feed is active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	/* BlockCommentField demonstrates a multi-line block comment
	 * with star-prefixed lines, which should be stripped in the
	 * generated YAML output.
	 */
	BlockCommentField string `yaml:"blockComment" json:"blockComment"`

	NoComment string `yaml:"noComment" json:"noComment"`
}

// envInputFixtureChangeset is a stub changeset typed with envInputFixtureConfig,
// used by the golden test to verify the WithEnvInput path. This changeset's
// generic type C is envInputFixtureConfig, so cfg.InputType is populated and
// the InputType branch of generateChangesetSection is exercised.
//
// It MUST live in a regular .go file (not a _test.go file) because
// packages.Load does not load _test.go files by default.
type envInputFixtureChangeset struct{}

func (envInputFixtureChangeset) Apply(_ fdeployment.Environment, _ envInputFixtureConfig) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (envInputFixtureChangeset) VerifyPreconditions(_ fdeployment.Environment, _ envInputFixtureConfig) error {
	return nil
}

var _ fdeployment.ChangeSetV2[envInputFixtureConfig] = (*envInputFixtureChangeset)(nil)

// resolverInputStruct is the input type accepted by typedResolverFixtureResolver.
// It is intentionally different from envInputFixtureConfig (the changeset's
// generic type C) to verify that the ConfigResolver path shows the resolver's
// input type, not the changeset's config type.
//
// It MUST live in a regular .go file (not a _test.go file) because
// packages.Load does not load _test.go files by default.
type resolverInputStruct struct {
	// ChainID is the EVM chain ID (not selector) to target.
	ChainID uint64 `yaml:"chainID" json:"chainID"`

	// ContractAddress is the deployed contract address to interact with.
	ContractAddress string `yaml:"contractAddress" json:"contractAddress"`
}

// typedResolverFixtureChangeset is a stub changeset typed with
// envInputFixtureConfig, but wired with a config resolver that accepts
// resolverInputStruct as its input. This verifies that the generated YAML
// template shows the resolver's input type (resolverInputStruct), not the
// changeset's generic config type (envInputFixtureConfig).
//
// It MUST live in a regular .go file (not a _test.go file) because
// packages.Load does not load _test.go files by default.
type typedResolverFixtureChangeset struct{}

func (typedResolverFixtureChangeset) Apply(_ fdeployment.Environment, _ envInputFixtureConfig) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (typedResolverFixtureChangeset) VerifyPreconditions(_ fdeployment.Environment, _ envInputFixtureConfig) error {
	return nil
}

var _ fdeployment.ChangeSetV2[envInputFixtureConfig] = (*typedResolverFixtureChangeset)(nil)
