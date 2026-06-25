package deployment

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	mcms_types "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

const ethMainnetSelector = 5009297550715157269

func TestNewOutputBuilder_fromMutableDataStore(t *testing.T) {
	t.Parallel()
	ds := datastore.NewMemoryDataStore()
	require.NoError(t, ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethMainnetSelector,
		Type:          "Timelock",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0xabc",
	}))

	b := requireTestOutputBuilder(t, ds)

	out, err := b.Build()
	require.NoError(t, err)
	refs, err := out.DataStore.Addresses().Fetch()
	require.NoError(t, err)
	require.Len(t, refs, 1)
	require.Equal(t, "0xabc", refs[0].Address)
}

func TestBuild_noTimelockProposals(t *testing.T) {
	t.Parallel()
	b := newTestOutputBuilder(t)
	out, err := b.Build()
	require.NoError(t, err)
	require.Empty(t, out.MCMSTimelockProposals)
}

func TestBuild_invalidMCMSInput(t *testing.T) {
	t.Parallel()
	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	_, err := b.WithTimelockProposal(MCMSTimelockProposalInput{
		TimelockAction: mcms_types.TimelockActionSchedule,
		ValidUntil:     1,
		TimelockDelay:  mcms_types.NewDuration(time.Hour),
	}, []mcms_types.BatchOperation{sampleBatchOp()}).Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to validate MCMS timelock proposal input")
	require.ErrorIs(t, err, ErrInvalidMCMSTimelockProposalInput)
}

func TestWithTimelockProposal_filtersEmptyTransactions(t *testing.T) {
	t.Parallel()

	emptyThenSample := []mcms_types.BatchOperation{
		{ChainSelector: ethMainnetSelector, Transactions: nil},
		sampleBatchOp(),
	}

	tests := []struct {
		name string
		opts []BatchOpsOption
	}{
		{name: "merge by default"},
		{
			name: "without merge",
			opts: []BatchOpsOption{WithoutMergeBatchOpsPerChain()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := newTestOutputBuilder(t)
			b.WithMCMSReaderRegistry(testMCMSRegistry(t))
			out, err := b.WithTimelockProposal(validMCMSProposalInput(), emptyThenSample, tt.opts...).Build()
			require.NoError(t, err)
			require.Len(t, out.MCMSTimelockProposals, 1)
			require.Len(t, out.MCMSTimelockProposals[0].Operations, 1)
			require.Len(t, out.MCMSTimelockProposals[0].Operations[0].Transactions, 1)
		})
	}
}

func TestWithTimelockProposal_withoutMerge_keepsSeparateOpsPerChain(t *testing.T) {
	t.Parallel()
	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	out, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{
		sampleBatchOp(),
		sampleBatchOp(),
	}, WithoutMergeBatchOpsPerChain()).Build()
	require.NoError(t, err)
	require.Len(t, out.MCMSTimelockProposals[0].Operations, 2)
}

func TestBuild_allEmptyBatchOpsSkipsProposal(t *testing.T) {
	t.Parallel()
	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	out, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{
		{ChainSelector: ethMainnetSelector, Transactions: nil},
		{ChainSelector: ethMainnetSelector, Transactions: []mcms_types.Transaction{}},
	}).Build()
	require.NoError(t, err)
	require.Empty(t, out.MCMSTimelockProposals)
}

func TestBuild_unregisteredChainFamily(t *testing.T) {
	t.Parallel()
	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(newMCMSReaderRegistry())
	_, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{sampleBatchOp()}).Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "no MCMS reader registered for chain family")
}

func TestBuild_invalidChainSelector(t *testing.T) {
	t.Parallel()
	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	_, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{
		{
			ChainSelector: 0,
			Transactions: []mcms_types.Transaction{
				{To: "0x01", Data: []byte("0x01"), AdditionalFields: json.RawMessage{}},
			},
		},
	}).Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "chain family for selector 0")
}

func TestBuild_getTimelockRefError(t *testing.T) {
	t.Parallel()
	readerErr := errors.New("timelock ref failed")
	registry := newMCMSReaderRegistry()
	require.NoError(t, registry.Register("evm", &failingTimelockReader{err: readerErr}))

	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(registry)
	_, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{sampleBatchOp()}).Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "get timelock ref for chain")
	require.ErrorIs(t, err, readerErr)
}

func TestBuild_getChainMetadataError(t *testing.T) {
	t.Parallel()
	readerErr := errors.New("chain metadata failed")
	registry := newMCMSReaderRegistry()
	require.NoError(t, registry.Register("evm", &failingMetadataReader{err: readerErr}))

	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(registry)
	_, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{sampleBatchOp()}).Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "get chain metadata for chain")
	require.ErrorIs(t, err, readerErr)
}

func TestBuild_deduplicatesReaderCallsPerChain(t *testing.T) {
	t.Parallel()
	reader := &countingReader{}
	registry := newMCMSReaderRegistry()
	require.NoError(t, registry.Register("evm", reader))

	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(registry)
	_, err := b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{
		sampleBatchOp(),
		sampleBatchOp(),
	}).Build()
	require.NoError(t, err)
	require.Equal(t, 1, reader.timelockCalls)
	require.Equal(t, 1, reader.metadataCalls)
}

func TestWithTimelockProposal(t *testing.T) {
	t.Parallel()
	input := validMCMSProposalInput()
	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	out, err := b.WithTimelockProposal(input, []mcms_types.BatchOperation{sampleBatchOp()}).Build()
	require.NoError(t, err)
	require.Len(t, out.MCMSTimelockProposals, 1)

	prop := out.MCMSTimelockProposals[0]
	require.Equal(t, input.Description, prop.Description)
	require.Equal(t, input.ValidUntil, prop.ValidUntil)
	require.Equal(t, input.OverridePreviousRoot, prop.OverridePreviousRoot)
	require.Equal(t, input.TimelockAction, prop.Action)
	require.Equal(t, input.TimelockDelay, prop.Delay)
	require.Equal(t, "0x01", prop.TimelockAddresses[ethMainnetSelector])
	require.Equal(t, testOpCount, prop.ChainMetadata[ethMainnetSelector].StartingOpCount)
	require.Len(t, prop.Operations, 1)
	require.Len(t, prop.Operations[0].Transactions, 1)
}

func TestWithTimelockProposal_mergePerChain(t *testing.T) {
	t.Parallel()
	const chainA = ethMainnetSelector
	const chainB = uint64(4340886533089894000)

	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	batchOps := []mcms_types.BatchOperation{
		{
			ChainSelector: mcms_types.ChainSelector(chainA),
			Transactions: []mcms_types.Transaction{
				{To: "0x01", Data: []byte("0xdeadbeef"), AdditionalFields: json.RawMessage{}},
			},
		},
		{
			ChainSelector: mcms_types.ChainSelector(chainA),
			Transactions: []mcms_types.Transaction{
				{To: "0x01", Data: []byte("0xcafebabe"), AdditionalFields: json.RawMessage{}},
			},
		},
		{
			ChainSelector: mcms_types.ChainSelector(chainB),
			Transactions: []mcms_types.Transaction{
				{To: "0x03", Data: []byte("0xface"), AdditionalFields: json.RawMessage{}},
			},
		},
	}
	out, err := b.WithTimelockProposal(validMCMSProposalInput(), batchOps).Build()
	require.NoError(t, err)
	require.Len(t, out.MCMSTimelockProposals, 1)
	require.Len(t, out.MCMSTimelockProposals[0].Operations, 2)

	prop := out.MCMSTimelockProposals[0]
	require.Equal(t, mcms_types.ChainSelector(chainA), prop.Operations[0].ChainSelector)
	require.Equal(t, mcms_types.ChainSelector(chainB), prop.Operations[1].ChainSelector)

	var txCountChainA, txCountChainB int
	for _, op := range prop.Operations {
		switch op.ChainSelector {
		case mcms_types.ChainSelector(chainA):
			txCountChainA = len(op.Transactions)
		case mcms_types.ChainSelector(chainB):
			txCountChainB = len(op.Transactions)
		}
	}
	require.Equal(t, 2, txCountChainA)
	require.Equal(t, 1, txCountChainB)

	var chainATxs []mcms_types.Transaction
	for _, op := range prop.Operations {
		if op.ChainSelector == mcms_types.ChainSelector(chainA) {
			chainATxs = op.Transactions
			break
		}
	}
	require.Equal(t, []byte("0xdeadbeef"), chainATxs[0].Data)
	require.Equal(t, []byte("0xcafebabe"), chainATxs[1].Data)
}

func TestWithTimelockProposal_multipleSpecs(t *testing.T) {
	t.Parallel()

	scheduleInput := validMCMSProposalInput()
	cancelInput := validMCMSProposalInput()
	cancelInput.Description = "Cancel proposal"
	cancelInput.TimelockAction = mcms_types.TimelockActionCancel
	cancelInput.TimelockDelay = mcms_types.NewDuration(0)

	b := newTestOutputBuilder(t)
	b.WithMCMSReaderRegistry(testMCMSRegistry(t))
	b.WithTimelockProposal(scheduleInput, []mcms_types.BatchOperation{sampleBatchOp()})
	b.WithTimelockProposal(cancelInput, []mcms_types.BatchOperation{
		{
			ChainSelector: ethMainnetSelector,
			Transactions: []mcms_types.Transaction{
				{To: "0x02", Data: []byte("0x01"), AdditionalFields: json.RawMessage{}},
				{To: "0x02", Data: []byte("0x02"), AdditionalFields: json.RawMessage{}},
			},
		},
	})

	out, err := b.Build()
	require.NoError(t, err)
	require.Len(t, out.MCMSTimelockProposals, 2)
	require.Equal(t, scheduleInput.Description, out.MCMSTimelockProposals[0].Description)
	require.Equal(t, cancelInput.Description, out.MCMSTimelockProposals[1].Description)
	require.Len(t, out.MCMSTimelockProposals[0].Operations[0].Transactions, 1)
	require.Len(t, out.MCMSTimelockProposals[1].Operations[0].Transactions, 2)
}

func TestBuild_returnsPartialOutputOnProposalFailure(t *testing.T) {
	t.Parallel()

	ds := datastore.NewMemoryDataStore()
	require.NoError(t, ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: ethMainnetSelector,
		Type:          "Timelock",
		Version:       semver.MustParse("1.0.0"),
		Address:       "0xabc",
	}))
	b := NewOutputBuilder(Environment{}, ds).
		WithMCMSReaderRegistry(testMCMSRegistry(t))
	b.WithTimelockProposal(validMCMSProposalInput(), []mcms_types.BatchOperation{sampleBatchOp()})
	b.WithTimelockProposal(MCMSTimelockProposalInput{
		TimelockAction: mcms_types.TimelockActionSchedule,
		ValidUntil:     1,
		TimelockDelay:  mcms_types.NewDuration(time.Hour),
	}, []mcms_types.BatchOperation{sampleBatchOp()})

	out, err := b.Build()
	require.Error(t, err)
	require.ErrorContains(t, err, "timelock proposal spec at index 1")
	require.NotNil(t, out.DataStore)
	refs, fetchErr := out.DataStore.Addresses().Fetch()
	require.NoError(t, fetchErr)
	require.Len(t, refs, 1)
	require.Len(t, out.MCMSTimelockProposals, 1)
}

func validMCMSProposalInput() MCMSTimelockProposalInput {
	input := validTestMCMSInput()
	input.OverridePreviousRoot = false
	input.TimelockDelay = mcms_types.NewDuration(3 * time.Hour)
	input.Description = "Proposal"

	return input
}

func sampleBatchOp() mcms_types.BatchOperation {
	return mcms_types.BatchOperation{
		ChainSelector: ethMainnetSelector,
		Transactions: []mcms_types.Transaction{
			{To: "0x01", Data: []byte("0xdeadbeef"), AdditionalFields: json.RawMessage{}},
		},
	}
}

func newTestOutputBuilder(t *testing.T) *OutputBuilder {
	t.Helper()

	return requireTestOutputBuilder(t, datastore.NewMemoryDataStore())
}

func requireTestOutputBuilder(t *testing.T, ds datastore.MutableDataStore) *OutputBuilder {
	t.Helper()

	return NewOutputBuilder(Environment{}, ds)
}

func testMCMSRegistry(t *testing.T) *MCMSReaderRegistry {
	t.Helper()
	r := newMCMSReaderRegistry()
	require.NoError(t, r.Register("evm", &mockReader{}))

	return r
}

// ---- Mock Readers ----

type failingTimelockReader struct {
	mockReader
	err error
}

func (r *failingTimelockReader) GetTimelockRef(_ Environment, _ uint64, _ MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{}, r.err
}

type failingMetadataReader struct {
	mockReader
	err error
}

func (r *failingMetadataReader) GetChainMetadata(_ Environment, _ uint64, _ MCMSTimelockProposalInput) (mcms_types.ChainMetadata, error) {
	return mcms_types.ChainMetadata{}, r.err
}

type countingReader struct {
	mockReader
	timelockCalls int
	metadataCalls int
}

func (r *countingReader) GetTimelockRef(e Environment, selector uint64, input MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	r.timelockCalls++
	return r.mockReader.GetTimelockRef(e, selector, input)
}

func (r *countingReader) GetChainMetadata(e Environment, selector uint64, input MCMSTimelockProposalInput) (mcms_types.ChainMetadata, error) {
	r.metadataCalls++
	return r.mockReader.GetChainMetadata(e, selector, input)
}
