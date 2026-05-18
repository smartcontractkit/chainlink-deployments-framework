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
// context's registered addresses.
func resolveContractInfo(ctx ProposalContext, chainSelector uint64, mcmsTx types.Transaction) (contractType, contractVersion string) {
	// Prefer the transaction's own ContractVersion when available (set by proposal creator)
	if mcmsTx.ContractVersion != nil {
		contractVersion = mcmsTx.ContractVersion.String()
	}

	dpc, ok := ctx.(*DefaultProposalContext)
	if !ok {
		return mcmsTx.ContractType, contractVersion
	}

	addresses, ok := dpc.AddressesByChain[chainSelector]
	if !ok {
		return mcmsTx.ContractType, contractVersion
	}

	tv, ok := addresses[mcmsTx.To]
	if !ok {
		return mcmsTx.ContractType, contractVersion
	}

	ct := string(tv.Type)
	if ct == "" {
		ct = mcmsTx.ContractType
	}

	// If version wasn't already set from transaction metadata, use datastore version
	if contractVersion == "" && tv.Version.Original() != "" {
		contractVersion = tv.Version.String()
	}

	return ct, contractVersion
}
