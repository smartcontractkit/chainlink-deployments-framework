package exceptions

import (
	"os"

	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

// ExceptionType defines the type of exception.
type ExceptionType string

const (
	ExceptionTypeRateLimit  ExceptionType = "RateLimit"
	ExceptionTypeDeprecated ExceptionType = "Deprecated"
)

// Exception represents a single exception entry.
type Exception struct {
	Resource string         `yaml:"resource"`
	Address  common.Address `yaml:"address"`
	Reason   string         `yaml:"reason"`
	Type     ExceptionType  `yaml:"type"`
}

// Exceptions holds a mapping of network names to their respective exceptions.
type Exceptions struct {
	Exceptions map[string][]Exception `yaml:"exceptions"`
}

// Load loads exceptions from a YAML file at the specified path.
func Load(filePath string) (*Exceptions, error) {
	d, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var e Exceptions
	if err := yaml.Unmarshal(d, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

// IsExpected checks if the given address on the specified network is an expected exception of the given type.
func (e *Exceptions) IsExpected(network string, address common.Address, exceptionType ExceptionType) (bool, string) {
	if _, ok := e.Exceptions[network]; ok {
		return e.isAddressExpected(network, address, exceptionType)
	}
	return false, ""
}

// isAddressExpected checks if the address is in the exceptions for the given network and type.
func (e *Exceptions) isAddressExpected(network string, address common.Address, exceptionType ExceptionType) (bool, string) {
	for _, exception := range e.Exceptions[network] {
		if exception.Address == address && exception.Type == exceptionType {
			return true, exception.Reason
		}
	}
	return false, ""
}
