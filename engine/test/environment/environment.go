// Package environment provides configuration options for loading test environments
// with various blockchain networks and components.
package environment

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	focr "github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

const (
	environmentName = "test_environment"
)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(t *testing.T, opts ...LoadOpt) (*deployment.Environment, error) {
	t.Helper()

	var (
		lggr   = logger.Test(t)
		getCtx = func() context.Context { return t.Context() }
		cmps   = newComponents()
	)

	if err := applyOptions(t, cmps, opts); err != nil {
		return nil, err
	}

	return &deployment.Environment{
		Name:              environmentName,
		Logger:            lggr,
		BlockChains:       fchain.NewBlockChainsFromSlice(cmps.Chains),
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         fdatastore.NewMemoryDataStore().Seal(),
		Catalog:           nil,        // Unimplemented for now
		NodeIDs:           []string{}, // Unimplemented for now
		Offchain:          nil,        // Unimplemented for now
		GetContext:        getCtx,
		OCRSecrets:        focr.XXXGenerateTestOCRSecrets(),
		OperationsBundle:  foperations.NewBundle(getCtx, lggr, foperations.NewMemoryReporter()),
	}, nil
}

// applyOptions applies the given options to load various components for the environment.
// It executes all options concurrently and returns a combined error if any option fails.
// If multiple options fail, all errors are joined using errors.Join.
func applyOptions(t *testing.T, cmps *components, opts []LoadOpt) error {
	t.Helper()

	// Handle empty options case
	if len(opts) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(opts))

	for _, opt := range opts {
		wg.Add(1)
		go func(option LoadOpt) {
			defer wg.Done()
			if err := option(cmps); err != nil {
				errChan <- err
			}
		}(opt)
	}
	wg.Wait()
	close(errChan)

	// Collect and combine any errors that occurred during option execution
	if len(errChan) > 0 {
		var merr error
		for err := range errChan {
			merr = errors.Join(merr, err)
		}

		return merr
	}

	return nil
}
