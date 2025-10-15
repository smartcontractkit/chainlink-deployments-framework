// Package environment provides configuration options for loading test environments
// with various blockchain networks and components.
package environment

import (
	"context"
	"errors"
	"fmt"
	"sync"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fcatalog "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/memory"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	focr "github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"
	foperations "github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

const (
	environmentName = "test_environment"
)

// New creates a new environment for testing.
//
// It loads the environment with the given options and returns the environment.
//
// If the environment fails to load, it returns an error.
func New(ctx context.Context, opts ...LoadOpt) (*fdeployment.Environment, error) {
	return NewLoader().Load(ctx, opts...)
}

// Loader instantiates a new environment with the given options.
type Loader struct{}

// NewLoader creates a new Loader instance.
func NewLoader() *Loader {
	return &Loader{}
}

// Load loads the environment with the given options.
func (l *Loader) Load(ctx context.Context, opts ...LoadOpt) (*fdeployment.Environment, error) {
	var (
		getCtx = func() context.Context { return ctx }
		cmps   = newComponents()
	)

	if err := applyOptions(cmps, opts); err != nil {
		return nil, err
	}

	ds := cmps.Datastore
	if ds == nil {
		ds = fdatastore.NewMemoryDataStore().Seal()
	}

	ab := cmps.AddressBook
	if ab == nil {
		ab = fdeployment.NewMemoryAddressBook()
	}

	nodeIDs := cmps.NodeIDs
	if len(nodeIDs) == 0 {
		nodeIDs = []string{}
	}

	// We do not set any default offchain client as it is not required for all tests.
	// We may want to set a default memory based offchain client in the future.
	oc := cmps.OffchainClient

	catalog, err := fcatalog.NewMemoryCatalogDataStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-memory catalog: %w", err)
	}

	return &fdeployment.Environment{
		Name:              environmentName,
		Logger:            cmps.Logger,
		BlockChains:       fchain.NewBlockChainsFromSlice(cmps.Chains),
		ExistingAddresses: ab,
		DataStore:         ds,
		Catalog:           catalog,
		NodeIDs:           nodeIDs,
		Offchain:          oc,
		GetContext:        getCtx,
		OCRSecrets:        focr.XXXGenerateTestOCRSecrets(),
		OperationsBundle:  foperations.NewBundle(getCtx, cmps.Logger, foperations.NewMemoryReporter()),
	}, nil
}

// applyOptions applies the given options to load various components for the environment.
// It executes all options concurrently and returns a combined error if any option fails.
// If multiple options fail, all errors are joined using errors.Join.
func applyOptions(cmps *components, opts []LoadOpt) error {
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
