package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

type IDLResize struct{ DataLen uint64 }

func (inst *IDLResize) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	{
		err := decoder.Decode(&inst.DataLen)
		if err != nil {
			return err
		}
	}

	return nil
}
