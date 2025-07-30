package deployment

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// OffchainClient interacts with the job-distributor
// which is a family agnostic interface for performing
// DON operations.
type OffchainClient interface {
	jobv1.JobServiceClient
	nodev1.NodeServiceClient
	csav1.CSAServiceClient
}

func MaybeDataErr(err error) error {
	//revive:disable
	var d rpc.DataError
	ok := errors.As(err, &d)
	if ok {
		return fmt.Errorf("%s: %v", d.Error(), d.ErrorData())
	}

	return err
}

// Environment represents an instance of a deployed product
// including on and offchain components. It is intended to be
// cross-family to enable a coherent view of a product deployed
// to all its chains.
// TODO: Add SolChains, AptosChain etc.
// using Go bindings/libraries from their respective
// repositories i.e. chainlink-solana, chainlink-cosmos
// You can think of ExistingAddresses as a set of
// family agnostic "onchain pointers" meant to be used in conjunction
// with chain fields to read/write relevant chain state. Similarly,
// you can think of NodeIDs as "offchain pointers" to be used in
// conjunction with the Offchain client to read/write relevant
// offchain state (i.e. state in the DON(s)).
type Environment struct {
	Name   string
	Logger logger.Logger
	// Deprecated: AddressBook is deprecated and will be removed in future versions.
	// Please use DataStore instead. If you still need to use AddressBook in your code,
	// be aware that you may encounter CI failures due to linting errors.
	// To work around this, you can disable the linter for that specific line using the //nolint directive.
	ExistingAddresses AddressBook
	DataStore         datastore.DataStore

	Catalog datastore.CatalogStore

	NodeIDs    []string
	Offchain   OffchainClient
	GetContext func() context.Context
	OCRSecrets OCRSecrets
	// OperationsBundle contains dependencies required by the operations API.
	OperationsBundle operations.Bundle
	// BlockChains is the container of all chains in the environment.
	BlockChains chain.BlockChains
}

// EnvironmentOption is a functional option for configuring an Environment
type EnvironmentOption func(*Environment)

// WithCatalog sets the catalog store for the environment
func WithCatalog(catalog datastore.CatalogStore) EnvironmentOption {
	return func(e *Environment) {
		e.Catalog = catalog
	}
}

// NewEnvironment creates a new environment for CLDF.
func NewEnvironment(
	name string,
	logger logger.Logger,
	existingAddrs AddressBook,
	dataStore datastore.DataStore,
	nodeIDs []string,
	offchain OffchainClient,
	ctx func() context.Context,
	secrets OCRSecrets,
	blockChains chain.BlockChains,
	opts ...EnvironmentOption,
) *Environment {
	env := &Environment{
		Name:              name,
		Logger:            logger,
		ExistingAddresses: existingAddrs,
		DataStore:         dataStore,
		NodeIDs:           nodeIDs,
		Offchain:          offchain,
		GetContext:        ctx,
		OCRSecrets:        secrets,
		// default to memory reporter as that is the only reporter available for now
		OperationsBundle: operations.NewBundle(ctx, logger, operations.NewMemoryReporter()),
		BlockChains:      blockChains,
	}

	// Apply functional options
	for _, opt := range opts {
		opt(env)
	}

	return env
}

// Clone creates a copy of the environment with a new reference to the address book.
func (e Environment) Clone() Environment {
	ab := NewMemoryAddressBook()
	if err := ab.Merge(e.ExistingAddresses); err != nil {
		panic(fmt.Sprintf("failed to copy address book: %v", err))
	}

	ds := datastore.NewMemoryDataStore()
	if e.DataStore != nil {
		if err := ds.Merge(e.DataStore); err != nil {
			panic(fmt.Sprintf("failed to copy datastore: %v", err))
		}
	}

	return Environment{
		Name:              e.Name,
		Logger:            e.Logger,
		ExistingAddresses: ab,
		DataStore:         ds.Seal(),
		Catalog:           e.Catalog, // Preserve the catalog reference
		NodeIDs:           e.NodeIDs,
		Offchain:          e.Offchain,
		GetContext:        e.GetContext,
		OCRSecrets:        e.OCRSecrets,
		OperationsBundle:  e.OperationsBundle,
		BlockChains:       e.BlockChains,
	}
}

// ConfirmIfNoError confirms the transaction if no error occurred.
// if the error is a DataError, it will return the decoded error message and data.
func ConfirmIfNoError(chain cldf_evm.Chain, tx *types.Transaction, err error) (uint64, error) {
	if err != nil {
		//revive:disable
		var d rpc.DataError
		ok := errors.As(err, &d)
		if ok {
			return 0, fmt.Errorf("transaction reverted on chain %s: Error %s ErrorData %v", chain.String(), d.Error(), d.ErrorData())
		}

		return 0, err
	}

	return chain.Confirm(tx)
}

// ConfirmIfNoErrorWithABI confirms the transaction if no error occurred.
// if the error is a DataError, it will return the decoded error message and data.
func ConfirmIfNoErrorWithABI(chain cldf_evm.Chain, tx *types.Transaction, abi string, err error) (uint64, error) {
	if err != nil {
		return 0, fmt.Errorf("transaction reverted on chain %s: Error %w",
			chain.String(), DecodedErrFromABIIfDataErr(err, abi))
	}

	return chain.Confirm(tx)
}

// DecodedErrFromABIIfDataErr decodes the error message and data from a DataError.
func DecodedErrFromABIIfDataErr(err error, abi string) error {
	var d rpc.DataError
	ok := errors.As(err, &d)
	if ok {
		errReason, parseErr := parseErrorFromABI(fmt.Sprintf("%s", d.ErrorData()), abi)
		if parseErr != nil {
			return fmt.Errorf("%s: %v", d.Error(), d.ErrorData())
		}

		return fmt.Errorf("%s due to %s: %v", d.Error(), errReason, d.ErrorData())
	}

	return err
}
