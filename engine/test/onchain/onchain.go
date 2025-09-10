package onchain

import (
	"sync"
)

// once ensures CTF initialization happens only once across all container startups.
var once = &sync.Once{}
