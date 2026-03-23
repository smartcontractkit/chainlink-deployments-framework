package artifacts

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

// randomWorkDirSuffix returns 16 hex chars for unique artifact filenames within a WorkDir.
func randomWorkDirSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}

func newWorkDirBinaryFileName() string {
	return "workflow-" + randomWorkDirSuffix() + ".wasm"
}

func newWorkDirConfigFileName() string {
	return "workflow-config-" + randomWorkDirSuffix() + ".json"
}
