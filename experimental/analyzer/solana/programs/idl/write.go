package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

type IDLWrite struct{ Data []byte } //nolint:revive // renaming would be a breaking change

func (inst *IDLWrite) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	{
		err := decoder.Decode(&inst.Data)
		if err != nil {
			return err
		}
	}

	return nil
}
