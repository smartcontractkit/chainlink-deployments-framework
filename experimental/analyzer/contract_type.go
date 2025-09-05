package analyzer

import (
	"github.com/Masterminds/semver/v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

var (
	Version0_5_0 = *semver.MustParse("0.5.0")
	Version1_0_0 = *semver.MustParse("1.0.0")
	Version1_1_0 = *semver.MustParse("1.1.0")
	Version1_2_0 = *semver.MustParse("1.2.0")
	Version1_5_0 = *semver.MustParse("1.5.0")
	Version1_5_1 = *semver.MustParse("1.5.1")
	Version1_6_0 = *semver.MustParse("1.6.0")
)

//nolint:gosec // G101: These are contract type identifiers, not hardcoded credentials
const (
	// Common
	LinkToken       deployment.ContractType = "LinkToken"
	StaticLinkToken deployment.ContractType = "StaticLinkToken"

	// MCMS
	AccessControllerProgram    deployment.ContractType = "AccessControllerProgram"
	ManyChainMultiSigProgram   deployment.ContractType = "ManyChainMultiSigProgram"
	RBACTimelockProgram        deployment.ContractType = "RBACTimelockProgram"
	RBACTimelock               deployment.ContractType = "RBACTimelock"
	CallProxy                  deployment.ContractType = "CallProxy"
	ManyChainMultisig          deployment.ContractType = "ManyChainMultiSig"
	ProposerManyChainMultisig  deployment.ContractType = "ProposerManyChainMultiSig"
	CancellerManyChainMultisig deployment.ContractType = "CancellerManyChainMultiSig"
	BypasserManyChainMultisig  deployment.ContractType = "BypasserManyChainMultiSig"

	// Keystone
	CapabilitiesRegistry deployment.ContractType = "CapabilitiesRegistry"
	WorkflowRegistry     deployment.ContractType = "WorkflowRegistry"
	KeystoneForwarder    deployment.ContractType = "KeystoneForwarder"
	OCR3Capability       deployment.ContractType = "OCR3Capability"
	FeedConsumer         deployment.ContractType = "FeedConsumer"

	// Data Streams
	ChannelConfigStore deployment.ContractType = "ChannelConfigStore"
	Configurator       deployment.ContractType = "Configurator"
	FeeManager         deployment.ContractType = "FeeManager"
	Verifier           deployment.ContractType = "Verifier"
	VerifierProxy      deployment.ContractType = "VerifierProxy"

	// CCIP
	BurnMintTokenPool              deployment.ContractType = "BurnMintTokenPool"
	OffRamp                        deployment.ContractType = "OffRamp"
	Router                         deployment.ContractType = "Router"
	TestRouter                     deployment.ContractType = "TestRouter"
	CCIPReceiver                   deployment.ContractType = "CCIPReceiver"
	FeeQuoter                      deployment.ContractType = "FeeQuoter"
	LockReleaseTokenPool           deployment.ContractType = "LockReleaseTokenPool"
	RMNRemote                      deployment.ContractType = "RMNRemote"
	OnRamp                         deployment.ContractType = "OnRamp"
	ARMProxy                       deployment.ContractType = "ARMProxy"
	RMNHome                        deployment.ContractType = "RMNHome"
	WETH9                          deployment.ContractType = "WETH9"
	NonceManager                   deployment.ContractType = "NonceManager"
	TokenAdminRegistry             deployment.ContractType = "TokenAdminRegistry"
	RegistryModule                 deployment.ContractType = "RegistryModuleOwnerCustom"
	USDCToken                      deployment.ContractType = "USDCToken"
	USDCTokenPool                  deployment.ContractType = "USDCTokenPool"
	HybridLockReleaseUSDCTokenPool deployment.ContractType = "HybridLockReleaseUSDCTokenPool"
	USDCMockTransmitter            deployment.ContractType = "USDCMockTransmitter"
	USDCTokenMessenger             deployment.ContractType = "USDCTokenMessenger"
	CCIPHome                       deployment.ContractType = "CCIPHome"
	LogMessageDataReceiver         deployment.ContractType = "LogMessageDataReceiver"
	Multicall3                     deployment.ContractType = "Multicall3"
	PriceFeed                      deployment.ContractType = "PriceFeed"
	BurnWithFromMintTokenPool      deployment.ContractType = "BurnWithFromMintTokenPool"
	BurnFromMintTokenPool          deployment.ContractType = "BurnFromMintTokenPool"
	BurnMintToken                  deployment.ContractType = "BurnMintToken"
	ERC20Token                     deployment.ContractType = "ERC20Token"
	ERC677Token                    deployment.ContractType = "ERC677Token"
	CommitStore                    deployment.ContractType = "CommitStore"
	PriceRegistry                  deployment.ContractType = "PriceRegistry"
	RMN                            deployment.ContractType = "RMN"
	MockRMN                        deployment.ContractType = "MockRMN"
)
