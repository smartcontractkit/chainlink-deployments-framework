package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

// IDLCreate instruction
type IDLCreate struct{ DataLen uint64 } //nolint:revive // renaming would be a breaking change

func (inst *IDLCreate) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	{
		err := decoder.Decode(&inst.DataLen)
		if err != nil {
			return err
		}
	}

	return nil
}
