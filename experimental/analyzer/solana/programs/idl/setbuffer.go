package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

type IDLSetBuffer struct{}

func (inst *IDLSetBuffer) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	return nil
}
