package config

import (
	"errors"
	"io/fs"
	"os"
	"slices"

	"github.com/spf13/viper"
)

// KMSConfig is the configuration for the AWS KMS.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type KMSConfig struct {
	KeyID     string `mapstructure:"key_id" yaml:"key_id"`         // Secret: AWS KMS Key ID
	KeyRegion string `mapstructure:"key_region" yaml:"key_region"` // Secret: AWS KMS Key Region (e.g. us-west-1)
}

// EVMConfig is the configuration for the EVM Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type EVMConfig struct {
	DeployerKey string      `mapstructure:"deployer_key" yaml:"deployer_key"` // Secret: The private key of the deployer account. Prefer to use KMS keys instead.
	Seth        *SethConfig `mapstructure:"seth" yaml:"seth,omitempty"`       // Seth configuration for transaction tracing
}

type SethConfig struct {
	ConfigFilePath  string   `mapstructure:"config_file_path" yaml:"config_file_path"`   // The path to the Seth config file
	GethWrapperDirs []string `mapstructure:"geth_wrapper_dirs" yaml:"geth_wrapper_dirs"` // The paths to the Geth wrapper directories
}

// SolanaConfig is the configuration for the Solana Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type SolanaConfig struct {
	WalletKey       string `mapstructure:"wallet_key" yaml:"wallet_key"`               // Secret: The private key of the wallet account.
	ProgramsDirPath string `mapstructure:"programs_dir_path" yaml:"programs_dir_path"` // The path to the Solana programs directory.
}

// TonConfig is the configuration for the TON Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type TonConfig struct {
	DeployerKey   string `mapstructure:"deployer_key" yaml:"deployer_key"`     // Secret: The private key of the deployer account.
	WalletVersion string `mapstructure:"wallet_version" yaml:"wallet_version"` // The version of the TON wallet
}

// AptosConfig is the configuration for the Aptos Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type AptosConfig struct {
	DeployerKey string `mapstructure:"deployer_key" yaml:"deployer_key"` // Secret: The private key of the deployer account.
}

// StellarConfig is the configuration for the Stellar Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type StellarConfig struct {
	DeployerKey string `mapstructure:"deployer_key" yaml:"deployer_key"` // Secret: The private key of the deployer account.
}

// SuiConfig is the configuration for the Sui Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type SuiConfig struct {
	DeployerKey string `mapstructure:"deployer_key" yaml:"deployer_key"` // Secret: The private key of the deployer account.
}

// TronConfig is the configuration for the Tron Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type TronConfig struct {
	DeployerKey string `mapstructure:"deployer_key" yaml:"deployer_key"` // Secret: The private key of the deployer account.
}

// CantonAuthType is the authentication scheme for Canton participant APIs.
const (
	CantonAuthTypeStatic            = "static"             // Pre-obtained JWT (e.g. from canton-login).
	CantonAuthTypeClientCredentials = "client_credentials" // CI: fetch token with client_id + client_secret + auth_url.
	CantonAuthTypeAuthorizationCode = "authorization_code" // Local: browser flow with client_id + auth_url.
)

// CantonConfig is the configuration for the Canton Chains.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type CantonConfig struct {
	// AuthType selects how to obtain the token: "static" (jwt_token), "client_credentials" (CI), or "authorization_code" (local browser).
	AuthType string `mapstructure:"auth_type" yaml:"auth_type"`
	// JWT token for static auth. Used when auth_type is "static".
	JWTToken string `mapstructure:"jwt_token" yaml:"jwt_token"` // Secret
	// AuthURL is the OIDC base URL (e.g. https://auth.example.com). Token URL is AuthURL/v1/token, authorize is AuthURL/v1/authorize.
	AuthURL string `mapstructure:"auth_url" yaml:"auth_url"`
	// ClientID is the OAuth2 client ID. Used for client_credentials and authorization_code.
	ClientID string `mapstructure:"client_id" yaml:"client_id"` // Secret
	// ClientSecret is the OAuth2 client secret. Required only for client_credentials (CI).
	ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"` // Secret
}

// JobDistributorConfig is the configuration for connecting and authenticating to the Job
// Distributor.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type JobDistributorConfig struct {
	Endpoints JobDistributorEndpoints `mapstructure:"endpoints" yaml:"endpoints"` // The URL endpoints for the Job Distributor
	Auth      *JobDistributorAuth     `mapstructure:"auth" yaml:"auth,omitempty"` // Secret: The authentication configuration for the Job Distributor
}

// JobDistributorAuth is the configuration for authenticating to the Job Distributor via Cognito.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type JobDistributorAuth struct {
	CognitoAppClientID     string `mapstructure:"cognito_app_client_id" yaml:"cognito_app_client_id"`         // Secret: The Cognito App Client ID
	CognitoAppClientSecret string `mapstructure:"cognito_app_client_secret" yaml:"cognito_app_client_secret"` // Secret: The Cognito App Client Secret
	AWSRegion              string `mapstructure:"aws_region" yaml:"aws_region"`                               // Secret: The AWS Region
	Username               string `mapstructure:"username" yaml:"username"`                                   // Secret: The Cognito username for the Job Distributor
	Password               string `mapstructure:"password" yaml:"password"`                                   // Secret: The Cognito password for the Job Distributor
}

// JobDistributorEndpoints is the configuration for the URL endpoints for the Job Distributor.
type JobDistributorEndpoints struct {
	GRPC string `mapstructure:"grpc" yaml:"grpc"` // The gRPC URL for the Job Distributor. Used to interact with the Job Distributor API.
}

// OCRConfig is the configuration for the OCR.
//
// WARNING: This data type contains sensitive fields and should not be logged or set in file
// configuration.
type OCRConfig struct {
	XSigners   string `mapstructure:"x_signers" yaml:"x_signers"`     // Secret: BIP39 mnemonic phrase for the OCR signer.
	XProposers string `mapstructure:"x_proposers" yaml:"x_proposers"` // Secret: BIP39 mnemonic phrase for the OCR proposer.
}

// CatalogAuthConfig is the configuration for the Catalog authentication.
type CatalogAuthConfig struct {
	KMSKeyID     string `mapstructure:"kms_key_id" yaml:"kms_key_id"`         // AWS KMS Key ID (arn or alias)
	KMSKeyRegion string `mapstructure:"kms_key_region" yaml:"kms_key_region"` // AWS KMS Key Region (e.g. us-west-1)
}

// CatalogConfig is the configuration to connect to the Catalog.
type CatalogConfig struct {
	GRPC string             `mapstructure:"grpc" yaml:"grpc"`           // The gRPC URL for the Catalog. Used to interact with the Catalog API.
	Auth *CatalogAuthConfig `mapstructure:"auth" yaml:"auth,omitempty"` // The authentication configuration for the Catalog.
}

// OnchainConfig wraps the configuration for the onchain components.
type OnchainConfig struct {
	KMS     KMSConfig     `mapstructure:"kms" yaml:"kms"`
	EVM     EVMConfig     `mapstructure:"evm" yaml:"evm"`
	Solana  SolanaConfig  `mapstructure:"solana" yaml:"solana"`
	Aptos   AptosConfig   `mapstructure:"aptos" yaml:"aptos"`
	Sui     SuiConfig     `mapstructure:"sui" yaml:"sui"`
	Stellar StellarConfig `mapstructure:"stellar" yaml:"stellar"`
	Tron    TronConfig    `mapstructure:"tron" yaml:"tron"`
	Ton     TonConfig     `mapstructure:"ton" yaml:"ton"`
	Canton  CantonConfig  `mapstructure:"canton" yaml:"canton"`
}

// OffchainConfig wraps the configuration for the offchain components.
type OffchainConfig struct {
	JobDistributor JobDistributorConfig `mapstructure:"job_distributor" yaml:"job_distributor"`
	OCR            OCRConfig            `mapstructure:"ocr" yaml:"ocr"`
}

// Config wraps the entire configuration for the CLD engine.
type Config struct {
	Onchain  OnchainConfig  `mapstructure:"onchain" yaml:"onchain"`
	Offchain OffchainConfig `mapstructure:"offchain" yaml:"offchain"`
	Catalog  CatalogConfig  `mapstructure:"catalog" yaml:"catalog"`
}

// Load loads the config from the file path, falling back to env vars if the file does not exist.
// If the file exists, any env vars that are set will override the values loaded from the file.
func Load(filePath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	// Bind environment variables
	if err := bindEnvs(v); err != nil {
		return nil, err
	}

	// If the config file exists, we continue to read it, otherwise we fallback to using
	// environment variables
	if _, err := os.Stat(filePath); !errors.Is(err, fs.ErrNotExist) {
		if err := v.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	cfg := &Config{}
	err := v.Unmarshal(cfg)

	return cfg, err
}

// LoadEnv loads the config from the environment variables.
func LoadEnv() (*Config, error) {
	v := viper.New()

	// Bind environment variables
	if err := bindEnvs(v); err != nil {
		return nil, err
	}

	cfg := &Config{}
	err := v.Unmarshal(cfg)

	return cfg, err
}

// LoadFile loads the config from a file.
func LoadFile(filePath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	err := v.Unmarshal(cfg)

	return cfg, err
}

var (
	// envBindings defines how environment variables map to configuration keys used by Viper.
	// Each entry maps a config key (as used in the struct, e.g. "onchain.kms.key_id") to a list of
	// environment variable names that can provide its value.
	//
	// The first element in the list is the preferred (new) environment variable name, and the second
	// (if present) is a legacy or backwards-compatible name. This allows the config loader to support
	// both new and old environment variable conventions seamlessly, ensuring smooth transitions and
	// compatibility with existing deployments.
	//
	// When loading, Viper will check each listed environment variable in order and use the first one
	// that is set.
	envBindings = map[string][]string{
		"onchain.kms.key_id":                                      {"ONCHAIN_KMS_KEY_ID", "KMS_DEPLOYER_KEY_ID"},
		"onchain.kms.key_region":                                  {"ONCHAIN_KMS_KEY_REGION", "KMS_DEPLOYER_KEY_REGION"},
		"onchain.evm.deployer_key":                                {"ONCHAIN_EVM_DEPLOYER_KEY", "TEST_WALLET_KEY"},
		"onchain.evm.seth.config_file_path":                       {"ONCHAIN_EVM_SETH_CONFIG_FILE_PATH", "SETH_CONFIG_FILE"},
		"onchain.evm.seth.geth_wrapper_dirs":                      {"ONCHAIN_EVM_SETH_GETH_WRAPPER_DIRS", "GETH_WRAPPERS_DIRS"},
		"onchain.solana.wallet_key":                               {"ONCHAIN_SOLANA_WALLET_KEY", "SOLANA_WALLET_KEY"},
		"onchain.solana.programs_dir_path":                        {"ONCHAIN_SOLANA_PROGRAMS_DIR_PATH", "SOLANA_PROGRAM_PATH"},
		"onchain.aptos.deployer_key":                              {"ONCHAIN_APTOS_DEPLOYER_KEY", "APTOS_DEPLOYER_KEY"},
		"onchain.tron.deployer_key":                               {"ONCHAIN_TRON_DEPLOYER_KEY", "TRON_DEPLOYER_KEY"},
		"onchain.sui.deployer_key":                                {"ONCHAIN_SUI_DEPLOYER_KEY", "SUI_DEPLOYER_KEY"},
		"onchain.stellar.deployer_key":                            {"ONCHAIN_STELLAR_DEPLOYER_KEY"},
		"onchain.ton.deployer_key":                                {"ONCHAIN_TON_DEPLOYER_KEY", "TON_DEPLOYER_KEY"},
		"onchain.ton.wallet_version":                              {"ONCHAIN_TON_WALLET_VERSION", "TON_WALLET_VERSION"},
		"onchain.canton.auth_type":                                {"ONCHAIN_CANTON_AUTH_TYPE"},
		"onchain.canton.jwt_token":                                {"ONCHAIN_CANTON_JWT_TOKEN"},
		"onchain.canton.auth_url":                                 {"ONCHAIN_CANTON_AUTH_URL"},
		"onchain.canton.client_id":                                {"ONCHAIN_CANTON_CLIENT_ID"},
		"onchain.canton.client_secret":                            {"ONCHAIN_CANTON_CLIENT_SECRET"},
		"offchain.job_distributor.auth.cognito_app_client_id":     {"OFFCHAIN_JD_AUTH_COGNITO_APP_CLIENT_ID", "JD_AUTH_COGNITO_APP_CLIENT_ID"},
		"offchain.job_distributor.auth.cognito_app_client_secret": {"OFFCHAIN_JD_AUTH_COGNITO_APP_CLIENT_SECRET", "JD_AUTH_COGNITO_APP_CLIENT_SECRET"},
		"offchain.job_distributor.auth.aws_region":                {"OFFCHAIN_JD_AUTH_AWS_REGION", "JD_AUTH_AWS_REGION"},
		"offchain.job_distributor.auth.username":                  {"OFFCHAIN_JD_AUTH_USERNAME", "JD_AUTH_USERNAME"},
		"offchain.job_distributor.auth.password":                  {"OFFCHAIN_JD_AUTH_PASSWORD", "JD_AUTH_PASSWORD"},
		"offchain.job_distributor.endpoints.grpc":                 {"OFFCHAIN_JD_ENDPOINTS_GRPC", "JD_GRPC"},
		"offchain.ocr.x_signers":                                  {"OFFCHAIN_OCR_X_SIGNERS", "OCR_X_SIGNERS"},
		"offchain.ocr.x_proposers":                                {"OFFCHAIN_OCR_X_PROPOSERS", "OCR_X_PROPOSERS"},
		"catalog.grpc":                                            {"CATALOG_GRPC"},
		"catalog.auth.kms_key_id":                                 {"CATALOG_AUTH_KMS_KEY_ID"},
		"catalog.auth.kms_key_region":                             {"CATALOG_AUTH_KMS_KEY_REGION"},
	}
)

// bindEnvs binds the environment variables to the viper instance.
func bindEnvs(v *viper.Viper) error {
	// Bind environment variables mappings to the viper instance
	for key, envs := range envBindings {
		// Prepend the env key to the start of the arguments
		inputs := slices.Insert(envs, 0, key)

		if err := v.BindEnv(inputs...); err != nil {
			return err
		}
	}

	return nil
}
