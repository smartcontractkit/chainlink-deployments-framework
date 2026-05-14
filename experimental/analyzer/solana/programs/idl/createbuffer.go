package idl

import (
	ag_binary "github.com/gagliardetto/binary"
)

type IDLCreateBuffer struct{} //nolint:revive // renaming would be a breaking change

func (inst *IDLCreateBuffer) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	return nil
}
