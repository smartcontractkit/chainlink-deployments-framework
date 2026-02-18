package analyzer

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
// context's registered addresses
func resolveContractInfo(ctx ProposalContext, chainSelector uint64, address string) (contractType, contractVersion string) {
	dpc, ok := ctx.(*DefaultProposalContext)
	if !ok {
		return "", ""
	}

	addresses, ok := dpc.AddressesByChain[chainSelector]
	if !ok {
		return "", ""
	}

	tv, ok := addresses[address]
	if !ok {
		return "", ""
	}

	ct := string(tv.Type)

	var cv string
	if tv.Version.Original() != "" {
		cv = tv.Version.String()
	}

	return ct, cv
}
