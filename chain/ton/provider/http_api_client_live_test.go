package provider

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	tonlib "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

// liveURLEnv enables the live tests below. They are read-only and run against whatever network the
// URL points at (the addresses used are mainnet contracts).
const liveURLEnv = "TON_HTTP_V2_URL"

const (
	// USDT jetton master: get_jetton_data returns (int, int, slice, cell, cell).
	liveJettonMaster = "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"
	// A TON Foundation wallet: active, with a long transaction history.
	liveWallet = "EQCD39VS5jcptHL8vMjEXrzGaRcCVYto7HUn4bpAOg8xqB2N"
)

func liveClient(t *testing.T) *httpAPIClient {
	t.Helper()

	u := os.Getenv(liveURLEnv)
	if u == "" {
		t.Skipf("%s not set", liveURLEnv)
	}

	return newHTTPAPIClient(u, os.Getenv(toncenterAPIKeyEnv))
}

func Test_Live_MasterchainAndBlockHeader(t *testing.T) {
	c := liveClient(t)

	blk, err := c.CurrentMasterchainInfo(t.Context())
	require.NoError(t, err)
	require.Equal(t, int32(-1), blk.Workchain)
	require.NotZero(t, blk.SeqNo)
	require.Len(t, blk.RootHash, 32)
	require.Len(t, blk.FileHash, 32)

	data, err := c.GetBlockData(t.Context(), blk)
	require.NoError(t, err)
	genTime := time.Unix(int64(data.BlockInfo.GenUtime), 0)
	t.Logf("masterchain seqno=%d gen_utime=%s", blk.SeqNo, genTime)
	require.WithinDuration(t, time.Now(), genTime, 5*time.Minute)
}

func Test_Live_RunGetMethod(t *testing.T) {
	c := liveClient(t)
	ctx := t.Context()

	blk, err := c.CurrentMasterchainInfo(ctx)
	require.NoError(t, err)

	t.Run("num, slice and cells", func(t *testing.T) {
		res, err := c.WaitForBlock(blk.SeqNo).RunGetMethod(ctx, blk, address.MustParseAddr(liveJettonMaster), "get_jetton_data")
		require.NoError(t, err)

		supply, err := res.Int(0)
		require.NoError(t, err)
		require.Positive(t, supply.Sign())

		mintable, err := res.Int(1)
		require.NoError(t, err)
		t.Logf("total_supply=%s mintable=%s", supply, mintable)

		admin, err := res.Slice(2)
		require.NoError(t, err)
		adminAddr, err := admin.LoadAddr()
		require.NoError(t, err)
		t.Logf("admin=%s", adminAddr)

		content, err := res.Cell(3)
		require.NoError(t, err)
		require.NotNil(t, content)

		code, err := res.Cell(4)
		require.NoError(t, err)
		require.NotNil(t, code)
	})

	t.Run("address argument as slice", func(t *testing.T) {
		// get_wallet_address(slice) -> slice
		res, err := c.RunGetMethod(ctx, blk, address.MustParseAddr(liveJettonMaster), "get_wallet_address",
			address.MustParseAddr(liveWallet))
		require.NoError(t, err)

		s, err := res.Slice(0)
		require.NoError(t, err)
		a, err := s.LoadAddr()
		require.NoError(t, err)
		t.Logf("jetton wallet=%s", a)
	})

	t.Run("big.Int argument", func(t *testing.T) {
		// get_nft_address_by_index is not on a jetton; use a method that fails with a non-zero exit code
		// to check ContractExecError mapping.
		_, err := c.RunGetMethod(ctx, blk, address.MustParseAddr(liveJettonMaster), "get_nft_address_by_index", big.NewInt(1))
		require.Error(t, err)
		t.Logf("expected failure: %v", err)
	})

	t.Run("wallet seqno", func(t *testing.T) {
		res, err := c.RunGetMethod(ctx, blk, address.MustParseAddr(liveWallet), "seqno")
		require.NoError(t, err)
		n, err := res.Int(0)
		require.NoError(t, err)
		t.Logf("seqno=%s", n)
	})
}

func Test_Live_GetAccount(t *testing.T) {
	c := liveClient(t)

	acc, err := c.GetAccount(t.Context(), nil, address.MustParseAddr(liveWallet))
	require.NoError(t, err)
	require.True(t, acc.IsActive)
	require.NotNil(t, acc.State)
	require.Equal(t, tlb.AccountStatus(tlb.AccountStatusActive), acc.State.Status)
	require.NotZero(t, acc.LastTxLT)
	require.Len(t, acc.LastTxHash, 32)
	require.NotNil(t, acc.Code)
	require.NotNil(t, acc.Data)
	t.Logf("balance=%s lastLT=%d", acc.State.Balance.String(), acc.LastTxLT)

	// An address that was never used reports as inactive.
	empty := address.NewAddress(0, 0, make([]byte, 32))
	acc, err = c.GetAccount(t.Context(), nil, empty)
	require.NoError(t, err)
	require.False(t, acc.IsActive)
}

func Test_Live_ListTransactions(t *testing.T) {
	c := liveClient(t)
	ctx := t.Context()
	addr := address.MustParseAddr(liveWallet)

	acc, err := c.GetAccount(ctx, nil, addr)
	require.NoError(t, err)

	txs, err := c.ListTransactions(ctx, addr, 5, acc.LastTxLT, acc.LastTxHash)
	require.NoError(t, err)
	require.NotEmpty(t, txs)

	// oldest first, newest is the account's last transaction, and the chain links up
	require.Equal(t, acc.LastTxLT, txs[len(txs)-1].LT)
	require.Equal(t, acc.LastTxHash, txs[len(txs)-1].Hash)
	for i := 1; i < len(txs); i++ {
		require.Equal(t, txs[i-1].LT, txs[i].PrevTxLT)
		require.Equal(t, txs[i-1].Hash, txs[i].PrevTxHash)
	}

	// paging: continue from the predecessor of the oldest returned
	older, err := c.ListTransactions(ctx, addr, 5, txs[0].PrevTxLT, txs[0].PrevTxHash)
	require.NoError(t, err)
	require.NotEmpty(t, older)
	require.Equal(t, txs[0].PrevTxLT, older[len(older)-1].LT)
}

func Test_Live_SubscribeOnTransactions(t *testing.T) {
	c := liveClient(t)
	ctx := t.Context()
	addr := address.MustParseAddr(liveWallet)

	acc, err := c.GetAccount(ctx, nil, addr)
	require.NoError(t, err)
	txs, err := c.ListTransactions(ctx, addr, 5, acc.LastTxLT, acc.LastTxHash)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(txs), 3)

	// pretend we processed everything up to the 3rd-newest tx; we expect the 2 newer ones, oldest first
	processed := txs[len(txs)-3].LT

	subCtx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	ch := make(chan *tlb.Transaction)
	go c.SubscribeOnTransactions(subCtx, addr, processed, ch)

	var got []*tlb.Transaction
	for len(got) < 2 {
		select {
		case tx, ok := <-ch:
			require.True(t, ok, "channel closed early")
			got = append(got, tx)
		case <-subCtx.Done():
			t.Fatalf("timed out; got %d txs", len(got))
		}
	}
	cancel()
	for range ch { // drain until closed
	}

	require.Equal(t, txs[len(txs)-2].LT, got[0].LT)
	require.Equal(t, txs[len(txs)-1].LT, got[1].LT)
}

func Test_Live_SendBocRejectsInvalidMessage(t *testing.T) {
	c := liveClient(t)

	// A well-formed external message with a garbage body to a never-initialised address: the node
	// must refuse it (and nothing can happen on chain).
	dst := address.NewAddress(0, 0, make([]byte, 32))
	err := c.SendExternalMessage(t.Context(), &tlb.ExternalMessage{
		DstAddr: dst,
		Body:    cell.BeginCell().MustStoreUInt(0xdeadbeef, 32).EndCell(),
	})
	require.Error(t, err)
	t.Logf("expected rejection: %v", err)
}

func Test_Live_ProviderInitializeAndReadOnly(t *testing.T) {
	u := os.Getenv(liveURLEnv)
	if u == "" {
		t.Skipf("%s not set", liveURLEnv)
	}

	p := NewRPCChainProvider(1, RPCChainProviderConfig{
		HTTPURL:           u,
		DeployerSignerGen: PrivateKeyRandom(),
	})
	bc, err := p.Initialize(t.Context())
	require.NoError(t, err)
	require.NotNil(t, bc)

	_, ok := bc.(interface{ ChainSelector() uint64 })
	require.True(t, ok)

	// The wallet's address is derivable and the client is the HTTP one.
	require.IsType(t, &httpAPIClient{}, p.chain.Client)
	require.Equal(t, u, p.chain.HTTPURL)
	require.Empty(t, p.chain.URL)
	var _ tonlib.APIClientWrapped = p.chain.Client
}
