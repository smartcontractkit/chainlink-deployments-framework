package environment

// NOTE: Although these constants are not used directly in the current codebase,
// they are imported into CLD and used in scaffolding templates.
// Do NOT delete them unless you are certain they are no longer referenced externally.

const (
	Local          = "local"
	StagingTestnet = "staging_testnet"
	StagingMainnet = "staging_mainnet"
	Staging        = "staging" // Note this is currently the equivalent of staging_testnet.
	ProdMainnet    = "prod_mainnet"
	ProdTestnet    = "prod_testnet"
	Prod           = "prod"

	// Legacy environments to be cleaned up once the migration to the above environments is completed.
	Testnet    = "testnet"
	Mainnet    = "mainnet"
	SolStaging = "solana-staging" // Note this is testnet staging for Solana.
)
