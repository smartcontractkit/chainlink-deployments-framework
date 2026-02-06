package upf

import (
	"bytes"
	"slices"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

type UPFProposal struct {
	MsigType     string                               `json:"msigType"`
	ProposalHash string                               `json:"proposalHash"`
	McmsParams   *McmsParams                          `json:"mcmsParams,omitempty"`
	GnosisParams *GnosisParams                        `json:"gnosisParams,omitempty"`
	Transactions []Transaction                        `json:"transactions"`
	Signers      map[mcmstypes.ChainSelector][]string `json:"signers,omitempty"`
}

type GnosisParams map[string]any

type McmsParams struct {
	ValidUntil           uint32   `json:"validUntil"`
	MerkleRoot           string   `json:"merkleRoot"`
	ASCIIProposalHash    rawBytes `json:"asciiProposalHash,omitempty"`
	OverridePreviousRoot bool     `json:"overridePreviousRoot"`
}

type Transaction struct {
	Index           int       `json:"index"`
	ChainFamily     string    `json:"chainFamily"`
	ChainID         string    `json:"chainId"`
	ChainName       string    `json:"chainName,omitempty"`
	ChainShortName  string    `json:"chainShortName,omitempty"`
	MsigAddress     string    `json:"msigAddress"`
	TimelockAddress string    `json:"timelockAddress,omitempty"`
	To              string    `json:"to"`
	Value           int64     `json:"value"`
	Data            string    `json:"data"`
	TxNonce         uint64    `json:"txNonce"`
	Metadata        *Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	ContractType    string           `json:"contractType,omitempty"`
	Comment         string           `json:"comment,omitempty" `
	DecodedCalldata *DecodedCallData `json:"decodedCalldata,omitempty"`
}

type DecodedCallData struct {
	FunctionName string                      `json:"functionName"`
	FunctionArgs DecodedCalldataFunctionArgs `json:"functionArgs"`
}

type DecodedInnerCall struct {
	To    string           `json:"to"`
	Value int64            `json:"value"`
	Data  *DecodedCallData `json:"data,omitempty"`
}

type DecodedCalldataFunctionArgs map[string]any // TODO: use type that preserves insertion order

type rawBytes []byte

func (m rawBytes) MarshalYAML() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	// In YAML single-quoted strings, single quotes are escaped by doubling them
	escaped := bytes.ReplaceAll(m, []byte{'\''}, []byte{'\'', '\''})

	return slices.Concat([]byte{'\''}, escaped, []byte{'\''}), nil
}
