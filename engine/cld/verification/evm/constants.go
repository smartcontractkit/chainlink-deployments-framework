package evm

// Shared verification parameter keys used across multiple verifier implementations.
const (
	paramAddress              = "address"
	paramContractAddress      = "contractaddress"
	paramSourceCode           = "sourceCode"
	paramCodeFormat           = "codeformat"
	paramContractName         = "contractname"
	paramCompilerVersion      = "compilerversion"
	paramConstructorArguments = "constructorArguments"
	paramOffset               = "offset"
	paramChainShortName       = "chainShortName"

	codeFormatSolidityJSON = "solidity-standard-json-input"
	actionVerifySourceCode = "verifysourcecode"

	strategyNameEtherscan  = "etherscan"
	strategyNameRoutescan  = "routescan"
	strategyNameOklink     = "oklink"
	strategyNameSocialscan = "socialscan"
)
