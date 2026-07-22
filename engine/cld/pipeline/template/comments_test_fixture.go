package template

// commentedFixtureStruct is a non-test Go struct with doc comments on its
// fields, used by comments_test.go to verify that the commentExtractor can
// read source-level // doc comments via AST parsing.
//
// It MUST live in a regular .go file (not a _test.go file) because
// packages.Load does not load _test.go files by default.
type commentedFixtureStruct struct {
	// ChainSelector is the EVM chain selector to deploy to.
	ChainSelector uint64 `yaml:"chainSelector" json:"chainSelector"`

	// WorkflowName is the name of the CRE workflow that consumes this feed.
	// It must match the workflow name registered in the DON config.
	WorkflowName string `yaml:"workflowName" json:"workflowName"`

	// Decimals is the on-chain precision the consumer-facing
	// AggregatorProxy.decimals() view will report.
	Decimals uint8 `yaml:"decimals" json:"decimals"`

	NoCommentField string `yaml:"noComment" json:"noComment"`

	// TrailingComment shows a same-line trailing comment. //nolint:unused
	TrailingComment string `yaml:"trailing" json:"trailing"` //nolint:unused
}
