package artifacts

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	mathrand "math/rand/v2"
)

// randomWorkDirSuffix returns exactly 16 hex chars (8 random bytes) for unique artifact filenames.
// Falls back to math/rand if the OS CSPRNG fails, which is weaker but sufficient for filename uniqueness.
func randomWorkDirSuffix() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		binary.BigEndian.PutUint64(b[:], mathrand.Uint64())
	}

	return hex.EncodeToString(b[:])
}

func newWorkDirBinaryFileName() string {
	return "workflow-" + randomWorkDirSuffix() + ".wasm"
}

func newWorkDirConfigFileName() string {
	return "workflow-config-" + randomWorkDirSuffix() + ".json"
}
