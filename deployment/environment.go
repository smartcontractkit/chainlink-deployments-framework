package deployment

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/zksync-sdk/zksync2-go/accounts"
	"github.com/zksync-sdk/zksync2-go/clients"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// OnchainClient is an EVM chain client.
// For EVM specifically we can use existing geth interface
// to abstract chain clients.
type OnchainClient interface {
	bind.ContractBackend
	bind.DeployBackend
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
	NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error)
}

// OffchainClient interacts with the job-distributor
// which is a family agnostic interface for performing
// DON operations.
type OffchainClient interface {
	jobv1.JobServiceClient
	nodev1.NodeServiceClient
	csav1.CSAServiceClient
}

// Chain represents an EVM chain.
type Chain struct {
	// Selectors used as canonical chain identifier.
	Selector uint64
	Client   OnchainClient
	// Note the Sign function can be abstract supporting a variety of key storage mechanisms (e.g. KMS etc).
	DeployerKey *bind.TransactOpts
	Confirm     func(tx *types.Transaction) (uint64, error)
	// Users are a set of keys that can be used to interact with the chain.
	// These are distinct from the deployer key.
	Users []*bind.TransactOpts

	// ZK deployment specifics
	IsZkSyncVM          bool
	ClientZkSyncVM      *clients.Client
	DeployerKeyZkSyncVM *accounts.Wallet
}

func (c Chain) String() string {
	chainInfo, err := ChainInfo(c.Selector)
	if err != nil {
		// we should never get here, if the selector is invalid it should not be in the environment
		panic(err)
	}

	return fmt.Sprintf("%s (%d)", chainInfo.ChainName, chainInfo.ChainSelector)
}

func (c Chain) Name() string {
	chainInfo, err := ChainInfo(c.Selector)
	if err != nil {
		// we should never get here, if the selector is invalid it should not be in the environment
		panic(err)
	}
	if chainInfo.ChainName == "" {
		return strconv.FormatUint(c.Selector, 10)
	}

	return chainInfo.ChainName
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
	DataStore         datastore.DataStore[
		datastore.DefaultMetadata,
		datastore.DefaultMetadata,
	]
	Chains      map[uint64]Chain
	SolChains   map[uint64]SolChain
	AptosChains map[uint64]AptosChain
	TonChains   map[uint64]TonChain
	NodeIDs     []string
	Offchain    OffchainClient
	GetContext  func() context.Context
	OCRSecrets  OCRSecrets
	// OperationsBundle contains dependencies required by the operations API.
	OperationsBundle operations.Bundle
}

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
	tonChains map[uint64]TonChain,
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
		TonChains:         tonChains,
		NodeIDs:           nodeIDs,
		Offchain:          offchain,
		GetContext:        ctx,
		OCRSecrets:        secrets,
		// default to memory reporter as that is the only reporter available for now
		OperationsBundle: operations.NewBundle(ctx, logger, operations.NewMemoryReporter()),
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
	}
}

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

func (e Environment) AllChainSelectorsTon() []uint64 {
	selectors := make([]uint64, 0, len(e.TonChains))
	for sel := range e.TonChains {
		selectors = append(selectors, sel)
	}
	sort.Slice(selectors, func(i, j int) bool {
		return selectors[i] < selectors[j]
	})

	return selectors
}

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
