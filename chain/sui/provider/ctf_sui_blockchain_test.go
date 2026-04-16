package provider

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_demuxDockerExecOutput(t *testing.T) {
	t.Parallel()
	assert.Empty(t, demuxDockerExecOutput(""))
	assert.Equal(t, `{"x":1}`, demuxDockerExecOutput(`{"x":1}`))
}

// encodeDockerStdoutFrame wraps payload in Docker exec attach multiplex framing (stdout stream).
func encodeDockerStdoutFrame(payload string) string {
	hdr := make([]byte, 8)
	hdr[0] = 1 // stdcopy.Stdout
	pl := uint64(len(payload))
	if pl > math.MaxUint32 {
		panic("encodeDockerStdoutFrame: payload length exceeds uint32")
	}
	binary.BigEndian.PutUint32(hdr[4:], uint32(pl))

	return string(hdr) + payload
}

func Test_parseSuiKeytoolGenerateJSON_dockerMuxStdout(t *testing.T) {
	t.Parallel()
	const addr = "0x2edc1dcfec4c0bc179157acab81fcb1b01f7e560ea9a7e08108840b20d49c7b9"
	compact := `{"alias":null,"flag":0,"keyScheme":"ED25519","mnemonic":"a b c","peerId":"p","publicBase64Key":"k","suiAddress":"` + addr + `"}`
	raw := encodeDockerStdoutFrame(compact)
	got, err := parseSuiKeytoolGenerateJSON(raw)
	require.NoError(t, err)
	require.Equal(t, addr, got.SuiAddress)
}

func Test_parseSuiKeytoolGenerateJSON(t *testing.T) {
	t.Parallel()

	const addr = "0x2edc1dcfec4c0bc179157acab81fcb1b01f7e560ea9a7e08108840b20d49c7b9"
	compact := `{"alias":null,"flag":0,"keyScheme":"ED25519","mnemonic":"a b c","peerId":"p","publicBase64Key":"k","suiAddress":"` + addr + `"}`
	pretty := "noise line\n{\n  \"alias\": null,\n  \"flag\": 0,\n  \"keyScheme\": \"ED25519\",\n  \"mnemonic\": \"a b c\",\n  \"peerId\": \"p\",\n  \"publicBase64Key\": \"k\",\n  \"suiAddress\": \"" + addr + "\"\n}\n"

	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "compact", raw: compact},
		{name: "preamble_plus_compact", raw: "please wait...\n" + compact},
		{name: "pretty", raw: pretty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseSuiKeytoolGenerateJSON(tc.raw)
			require.NoError(t, err)
			require.Equal(t, addr, got.SuiAddress)
			require.Equal(t, "ED25519", got.KeyScheme)
		})
	}
}
