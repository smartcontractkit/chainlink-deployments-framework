package deployment

import "os"

// DomainConfigGetter retrieves config values by key.
type DomainConfigGetter interface {
	Get(key string) (value string, found bool)
}

type envVarDomainConfigGetter struct{}

func (envVarDomainConfigGetter) Get(key string) (string, bool) {
	return os.LookupEnv(key)
}
