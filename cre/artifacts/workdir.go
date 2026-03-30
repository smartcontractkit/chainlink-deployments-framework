package artifacts

import (
	"crypto/rand"
	"encoding/hex"
)

// randomWorkDirSuffix returns exactly 16 hex chars (8 random bytes) for unique artifact filenames.
func randomWorkDirSuffix() string {
	var b [8]byte
	_, _ = rand.Read(b[:])

	return hex.EncodeToString(b[:])
}

func newWorkDirBinaryFileName() string {
	return "workflow-" + randomWorkDirSuffix() + ".wasm"
}

func newWorkDirConfigFileName() string {
	return "workflow-config-" + randomWorkDirSuffix() + ".json"
}
