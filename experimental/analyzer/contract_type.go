package analyzer

import (
	"github.com/Masterminds/semver/v3"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
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
	LinkToken       cldf.ContractType = "LinkToken"
	StaticLinkToken cldf.ContractType = "StaticLinkToken"

	// MCMS
	AccessControllerProgram    cldf.ContractType = "AccessControllerProgram"
	ManyChainMultiSigProgram   cldf.ContractType = "ManyChainMultiSigProgram"
	RBACTimelockProgram        cldf.ContractType = "RBACTimelockProgram"
	RBACTimelock               cldf.ContractType = "RBACTimelock"
	CallProxy                  cldf.ContractType = "CallProxy"
	ManyChainMultisig          cldf.ContractType = "ManyChainMultiSig"
	ProposerManyChainMultisig  cldf.ContractType = "ProposerManyChainMultiSig"
	CancellerManyChainMultisig cldf.ContractType = "CancellerManyChainMultiSig"
	BypasserManyChainMultisig  cldf.ContractType = "BypasserManyChainMultiSig"

	// Keystone
	CapabilitiesRegistry cldf.ContractType = "CapabilitiesRegistry"
	WorkflowRegistry     cldf.ContractType = "WorkflowRegistry"
	KeystoneForwarder    cldf.ContractType = "KeystoneForwarder"
	OCR3Capability       cldf.ContractType = "OCR3Capability"
	FeedConsumer         cldf.ContractType = "FeedConsumer"

	// Data Streams
	ChannelConfigStore cldf.ContractType = "ChannelConfigStore"
	Configurator       cldf.ContractType = "Configurator"
	FeeManager         cldf.ContractType = "FeeManager"
	Verifier           cldf.ContractType = "Verifier"
	VerifierProxy      cldf.ContractType = "VerifierProxy"

	// CCIP
	BurnMintTokenPool              cldf.ContractType = "BurnMintTokenPool"
	OffRamp                        cldf.ContractType = "OffRamp"
	Router                         cldf.ContractType = "Router"
	TestRouter                     cldf.ContractType = "TestRouter"
	CCIPReceiver                   cldf.ContractType = "CCIPReceiver"
	FeeQuoter                      cldf.ContractType = "FeeQuoter"
	LockReleaseTokenPool           cldf.ContractType = "LockReleaseTokenPool"
	RMNRemote                      cldf.ContractType = "RMNRemote"
	OnRamp                         cldf.ContractType = "OnRamp"
	ARMProxy                       cldf.ContractType = "ARMProxy"
	RMNHome                        cldf.ContractType = "RMNHome"
	WETH9                          cldf.ContractType = "WETH9"
	NonceManager                   cldf.ContractType = "NonceManager"
	TokenAdminRegistry             cldf.ContractType = "TokenAdminRegistry"
	RegistryModule                 cldf.ContractType = "RegistryModuleOwnerCustom"
	USDCToken                      cldf.ContractType = "USDCToken"
	USDCTokenPool                  cldf.ContractType = "USDCTokenPool"
	HybridLockReleaseUSDCTokenPool cldf.ContractType = "HybridLockReleaseUSDCTokenPool"
	USDCMockTransmitter            cldf.ContractType = "USDCMockTransmitter"
	USDCTokenMessenger             cldf.ContractType = "USDCTokenMessenger"
	CCIPHome                       cldf.ContractType = "CCIPHome"
	LogMessageDataReceiver         cldf.ContractType = "LogMessageDataReceiver"
	Multicall3                     cldf.ContractType = "Multicall3"
	PriceFeed                      cldf.ContractType = "PriceFeed"
	BurnWithFromMintTokenPool      cldf.ContractType = "BurnWithFromMintTokenPool"
	BurnFromMintTokenPool          cldf.ContractType = "BurnFromMintTokenPool"
	BurnMintToken                  cldf.ContractType = "BurnMintToken"
	ERC20Token                     cldf.ContractType = "ERC20Token"
	ERC677Token                    cldf.ContractType = "ERC677Token"
	CommitStore                    cldf.ContractType = "CommitStore"
	PriceRegistry                  cldf.ContractType = "PriceRegistry"
	RMN                            cldf.ContractType = "RMN"
	MockRMN                        cldf.ContractType = "MockRMN"
)
