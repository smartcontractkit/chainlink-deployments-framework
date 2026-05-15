package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

type IDLCreateBuffer struct{}

func (inst *IDLCreateBuffer) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	return nil
}
