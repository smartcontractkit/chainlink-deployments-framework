package analyzer

import (
	"github.com/smartcontractkit/mcms/types"
)

const (
	// Magic number constants
	MinStructFieldsForPrettyFormat = 2
	MinDataLengthForMethodID       = 4
	DefaultAnalyzersCount          = 2
)

type DecodedCall struct {
	Address         string
	Method          string
	Inputs          []NamedField
	Outputs         []NamedField
	ContractType    string
	ContractVersion string
}

// String renders a human-readable representation of the decoded call using the default text renderer.
// This method is kept for backwards compatibility but rendering should be done through renderers.
func (d *DecodedCall) String(context *FieldContext) string {
	// Use the text renderer to provide proper formatting
	renderer := NewTextRenderer()
	return renderer.RenderDecodedCall(d, context)
}

// resolveContractInfo looks up the contract type and version from the proposal
// context's registered addresses using the MCMS identity target (tx.To).
func resolveContractInfo(ctx ProposalContext, chainSelector uint64, mcmsTx types.Transaction) (contractType, contractVersion string) {
	fallbackVersion := ""
	if mcmsTx.ContractVersion != nil {
		fallbackVersion = mcmsTx.ContractVersion.String()
	}

	return lookupContractInfoByAddress(ctx, chainSelector, mcmsTx.To, mcmsTx.ContractType, fallbackVersion)
}

// lookupContractInfoByAddress resolves contract type and version for an address from
// the proposal context address book, falling back to the provided defaults.
func lookupContractInfoByAddress(
	ctx ProposalContext,
	chainSelector uint64,
	address string,
	fallbackType string,
	fallbackVersion string,
) (contractType, contractVersion string) {
	contractType = fallbackType
	contractVersion = fallbackVersion

	dpc, ok := ctx.(*DefaultProposalContext)
	if !ok {
		return contractType, contractVersion
	}

	addresses, ok := dpc.AddressesByChain[chainSelector]
	if !ok {
		return contractType, contractVersion
	}

	tv, ok := addresses[address]
	if !ok {
		return contractType, contractVersion
	}

	if ct := string(tv.Type); ct != "" {
		contractType = ct
	}

	if tv.Version.Original() != "" {
		contractVersion = tv.Version.String()
	}

	return contractType, contractVersion
}
