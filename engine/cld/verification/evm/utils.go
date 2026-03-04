package evm

import (
	"sync"
	"time"
)

const RoutescanRateLimit = 2

var routescanRateLimiter = struct {
	ticker *time.Ticker
	once   sync.Once
}{}
