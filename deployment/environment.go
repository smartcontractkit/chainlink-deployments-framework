package deployment

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
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
