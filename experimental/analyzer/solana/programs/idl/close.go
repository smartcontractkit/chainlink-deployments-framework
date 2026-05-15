package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

type IDLClose struct{}

func (inst *IDLClose) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	return nil
}
