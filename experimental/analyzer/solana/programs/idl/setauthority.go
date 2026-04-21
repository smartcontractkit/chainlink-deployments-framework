package idl

import (
	ag_binary "github.com/gagliardetto/binary"
	ag_solanago "github.com/gagliardetto/solana-go"
)

type IDLSetAuthority struct{ NewAuthority ag_solanago.PublicKey }

func (inst *IDLSetAuthority) UnmarshalWithDecoder(decoder *ag_binary.Decoder) error {
	{
		err := decoder.Decode(&inst.NewAuthority)
		if err != nil {
			return err
		}
	}

	return nil
}
