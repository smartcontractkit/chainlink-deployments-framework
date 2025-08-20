package environment

// Environment names which correspond to the directory
// structure in each product. For example:
// keystone/staging
// keystone/mainnet
// ccip/mainnet
// etc.
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
