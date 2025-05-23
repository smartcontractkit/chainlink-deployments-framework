package deployment

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
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

// todo: clean up in future once Chainlink is migrated
type OnchainClient = evm.OnchainClient
type Chain = evm.Chain

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
	DataStore         datastore.DataStore[
		datastore.DefaultMetadata,
		datastore.DefaultMetadata,
	]

	// Chains is being deprecated in favour of BlockChains field
	// use BlockChains.EVMChains()
	Chains map[uint64]Chain

	// SolChains is being deprecated in favour of BlockChains field
	// use BlockChains.SolanaChains()
	SolChains map[uint64]SolChain

	// AptosChains is being deprecated in favour of BlockChains field
	// use BlockChains.AptosChains()
	AptosChains map[uint64]AptosChain
	NodeIDs     []string
	Offchain    OffchainClient
	GetContext  func() context.Context
	OCRSecrets  OCRSecrets
	// OperationsBundle contains dependencies required by the operations API.
	OperationsBundle operations.Bundle
	// BlockChains is the container of all chains in the environment.
	BlockChains chain.BlockChains
}

// todo: remove once chainlink and cld are migrated to NewCLDFEnvironment
func NewEnvironment(
	name string,
	logger logger.Logger,
	existingAddrs AddressBook,
	dataStore datastore.DataStore[
		datastore.DefaultMetadata,
		datastore.DefaultMetadata,
	],
	chains map[uint64]Chain,
	solChains map[uint64]SolChain,
	aptosChains map[uint64]AptosChain,
	nodeIDs []string,
	offchain OffchainClient,
	ctx func() context.Context,
	secrets OCRSecrets,
) *Environment {
	return &Environment{
		Name:              name,
		Logger:            logger,
		ExistingAddresses: existingAddrs,
		DataStore:         dataStore,
		Chains:            chains,
		SolChains:         solChains,
		AptosChains:       aptosChains,
		NodeIDs:           nodeIDs,
		Offchain:          offchain,
		GetContext:        ctx,
		OCRSecrets:        secrets,
		// default to memory reporter as that is the only reporter available for now
		OperationsBundle: operations.NewBundle(ctx, logger, operations.NewMemoryReporter()),
	}
}

// NewCLDFEnvironment creates a new environment.
func NewCLDFEnvironment(
	name string,
	logger logger.Logger,
	existingAddrs AddressBook,
	dataStore datastore.DataStore[
		datastore.DefaultMetadata,
		datastore.DefaultMetadata,
	],
	chains map[uint64]Chain,
	solChains map[uint64]SolChain,
	aptosChains map[uint64]AptosChain,
	nodeIDs []string,
	offchain OffchainClient,
	ctx func() context.Context,
	secrets OCRSecrets,
	blockChains chain.BlockChains,
) *Environment {
	return &Environment{
		Name:              name,
		Logger:            logger,
		ExistingAddresses: existingAddrs,
		DataStore:         dataStore,
		Chains:            chains,
		SolChains:         solChains,
		AptosChains:       aptosChains,
		NodeIDs:           nodeIDs,
		Offchain:          offchain,
		GetContext:        ctx,
		OCRSecrets:        secrets,
		// default to memory reporter as that is the only reporter available for now
		OperationsBundle: operations.NewBundle(ctx, logger, operations.NewMemoryReporter()),
		BlockChains:      blockChains,
	}
}

// Clone creates a copy of the environment with a new reference to the address book.
func (e Environment) Clone() Environment {
	ab := NewMemoryAddressBook()
	if err := ab.Merge(e.ExistingAddresses); err != nil {
		panic(fmt.Sprintf("failed to copy address book: %v", err))
	}

	ds := datastore.NewMemoryDataStore[
		datastore.DefaultMetadata,
		datastore.DefaultMetadata,
	]()
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
		Chains:            e.Chains,
		SolChains:         e.SolChains,
		AptosChains:       e.AptosChains,
		NodeIDs:           e.NodeIDs,
		Offchain:          e.Offchain,
		GetContext:        e.GetContext,
		OCRSecrets:        e.OCRSecrets,
		OperationsBundle:  e.OperationsBundle,
		BlockChains:       e.BlockChains,
	}
}

// AllChainSelectors is being deprecated.
// Use e.BlockChains.ListChainSelectors(...) instead.
func (e Environment) AllChainSelectors() []uint64 {
	selectors := make([]uint64, 0, len(e.Chains))
	for sel := range e.Chains {
		selectors = append(selectors, sel)
	}
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

// AllChainSelectorsExcluding is being deprecated.
// Use e.BlockChains.ListChainSelectors(...) instead.
func (e Environment) AllChainSelectorsExcluding(excluding []uint64) []uint64 {
	selectors := make([]uint64, 0, len(e.Chains))
	for sel := range e.Chains {
		excluded := false
		for _, toExclude := range excluding {
			if sel == toExclude {
				excluded = true
			}
		}
		if excluded {
			continue
		}
		selectors = append(selectors, sel)
	}
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

// AllChainSelectorsSolana is being deprecated.
// Use e.BlockChains.ListChainSelectors(...) instead.
func (e Environment) AllChainSelectorsSolana() []uint64 {
	selectors := make([]uint64, 0, len(e.SolChains))
	for sel := range e.SolChains {
		selectors = append(selectors, sel)
	}
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

// AllChainSelectorsAptos is being deprecated.
// Use e.BlockChains.ListChainSelectors(...) instead.
func (e Environment) AllChainSelectorsAptos() []uint64 {
	selectors := make([]uint64, 0, len(e.AptosChains))
	for sel := range e.AptosChains {
		selectors = append(selectors, sel)
	}
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

// AllChainSelectorsAllFamilies is being deprecated.
// Use e.BlockChains.ListChainSelectors instead.
func (e Environment) AllChainSelectorsAllFamilies() []uint64 {
	selectors := make([]uint64, 0, len(e.Chains)+len(e.SolChains)+len(e.AptosChains))
	for sel := range e.Chains {
		selectors = append(selectors, sel)
	}
	for sel := range e.SolChains {
		selectors = append(selectors, sel)
	}
	for sel := range e.AptosChains {
		selectors = append(selectors, sel)
	}
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

// AllChainSelectorsAllFamiliesExcluding is being deprecated.
// Use e.BlockChains.ListChainSelectors instead.
func (e Environment) AllChainSelectorsAllFamiliesExcluding(excluding []uint64) []uint64 {
	selectors := e.AllChainSelectorsAllFamilies()
	ret := make([]uint64, 0)
	// remove the excluded selectors
	for _, sel := range selectors {
		if slices.Contains(excluding, sel) {
			continue
		}
		ret = append(ret, sel)
	}

	return ret
}

func (e Environment) AllDeployerKeys() []common.Address {
	deployerKeys := make([]common.Address, 0, len(e.Chains))
	for sel := range e.Chains {
		deployerKeys = append(deployerKeys, e.Chains[sel].DeployerKey.From)
	}

	return deployerKeys
}

// ConfirmIfNoError confirms the transaction if no error occurred.
// if the error is a DataError, it will return the decoded error message and data.
func ConfirmIfNoError(chain Chain, tx *types.Transaction, err error) (uint64, error) {
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
func ConfirmIfNoErrorWithABI(chain Chain, tx *types.Transaction, abi string, err error) (uint64, error) {
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
