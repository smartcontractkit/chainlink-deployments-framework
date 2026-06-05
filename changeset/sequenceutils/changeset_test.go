package sequenceutils

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	mcms_types "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const ethMainnetSelector = 5009297550715157269

const testValidUntilUnix = uint32(2756219818)

type testCfg struct{}

func TestNewOnChainChangesetFromSequence_verifyParams(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}

	t.Run("nil sequence", func(t *testing.T) {
		t.Parallel()
		cs := NewOnChainChangesetFromSequence(newTestChangesetParams(nil, nil, nil))
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidConfig)
		require.ErrorContains(t, err, "sequence is required")
	})

	t.Run("nil ResolveInput", func(t *testing.T) {
		t.Parallel()
		seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{}, nil
		})
		cs := NewOnChainChangesetFromSequence(NewOnChainChangesetFromSequenceParams[struct{}, struct{}, testCfg]{
			Sequence:   seq,
			ResolveDep: func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil },
		})
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidConfig)
		require.ErrorContains(t, err, "ResolveInput is required")
	})

	t.Run("nil ResolveDep", func(t *testing.T) {
		t.Parallel()
		seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
			return OnChainOutput{}, nil
		})
		cs := NewOnChainChangesetFromSequence(NewOnChainChangesetFromSequenceParams[struct{}, struct{}, testCfg]{
			Sequence:     seq,
			ResolveInput: func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil },
		})
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidConfig)
		require.ErrorContains(t, err, "ResolveDep is required")
	})
}

func TestNewOnChainChangesetFromSequence_verifyEnvironment(t *testing.T) {
	t.Parallel()

	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))
	cfg := testCfg{}

	t.Run("missing logger", func(t *testing.T) {
		t.Parallel()
		env := testEnvironment(t)
		env.Logger = nil
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidEnvironment)
		require.ErrorContains(t, err, "logger is required")
	})

	t.Run("missing GetContext", func(t *testing.T) {
		t.Parallel()
		env := testEnvironment(t)
		env.GetContext = nil
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidEnvironment)
		require.ErrorContains(t, err, "GetContext is required")
	})

	t.Run("unconfigured OperationsBundle", func(t *testing.T) {
		t.Parallel()
		err := cs.VerifyPreconditions(deployment.Environment{
			Logger:     logger.Test(t),
			GetContext: func() context.Context { return t.Context() },
		}, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidEnvironment)
		require.ErrorContains(t, err, "OperationsBundle is not configured")
	})
}

func TestNewOnChainChangesetFromSequence_verifyResolveErrors(t *testing.T) {
	t.Parallel()

	resolveErr := errors.New("resolve failed")
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{}, nil
	})
	env := testEnvironment(t)
	cfg := testCfg{}

	t.Run("ResolveInput failure", func(t *testing.T) {
		t.Parallel()
		cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq,
			func(deployment.Environment, testCfg) (struct{}, error) {
				return struct{}{}, resolveErr
			},
			func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil },
		))
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidConfig)
		require.ErrorIs(t, err, resolveErr)
		require.ErrorContains(t, err, "failed to resolve input")
		require.ErrorContains(t, err, seq.ID())
	})

	t.Run("ResolveDep failure", func(t *testing.T) {
		t.Parallel()
		cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq,
			func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil },
			func(deployment.Environment, testCfg) (struct{}, error) {
				return struct{}{}, resolveErr
			},
		))
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, deployment.ErrInvalidConfig)
		require.ErrorIs(t, err, resolveErr)
		require.ErrorContains(t, err, "failed to resolve dependencies")
	})
}

func TestNewOnChainChangesetFromSequence_verifyMCMS(t *testing.T) {
	t.Parallel()

	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{}, nil
	})
	env := testEnvironment(t)
	cfg := testCfg{}

	t.Run("invalid MCMS input", func(t *testing.T) {
		t.Parallel()
		cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{
			Cfg:  cfg,
			MCMS: &deployment.MCMSTimelockProposalInput{},
		})
		require.ErrorIs(t, err, deployment.ErrInvalidConfig)
		require.ErrorIs(t, err, deployment.ErrInvalidMCMSTimelockProposalInput)
	})

	t.Run("custom Verify hook", func(t *testing.T) {
		t.Parallel()
		verifyErr := errors.New("on-chain verify failed")
		params := newTestChangesetParams(seq, nil, nil)
		params.Verify = func(deployment.Environment, WithMCMS[testCfg]) error {
			return verifyErr
		}
		cs := NewOnChainChangesetFromSequence(params)
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
		require.ErrorIs(t, err, verifyErr)
	})

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))
		err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{
			Cfg:  cfg,
			MCMS: ptr(validMCMSProposalInput()),
		})
		require.NoError(t, err)
	})
}

func TestNewOnChainChangesetFromSequence_applySuccess(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}
	ref := datastore.AddressRef{
		Address:       "0xabc",
		ChainSelector: 1,
		Type:          "Timelock",
		Version:       semver.MustParse("1.0.0"),
	}
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			Metadata: datastore.MetadataBundle{Addresses: []datastore.AddressRef{ref}},
		}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))

	out, err := cs.Apply(env, WithMCMS[testCfg]{Cfg: cfg})
	require.NoError(t, err)
	require.NotEmpty(t, out.Reports)

	refs, fetchErr := out.DataStore.Addresses().Fetch()
	require.NoError(t, fetchErr)
	require.Len(t, refs, 1)
	require.Equal(t, "0xabc", refs[0].Address)
	require.Empty(t, out.MCMSTimelockProposals)
}

func TestNewOnChainChangesetFromSequence_applyWithBatchOpsAndMCMS(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}
	mcmsInput := validMCMSProposalInput()
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{BatchOps: []mcms_types.BatchOperation{sampleBatchOp()}}, nil
	})
	cs := NewOnChainChangesetFromSequence(
		newTestChangesetParams(seq, nil, nil),
		WithMCMSRegistry(testMCMSRegistry(t)),
	)

	out, err := cs.Apply(env, WithMCMS[testCfg]{Cfg: cfg, MCMS: &mcmsInput})
	require.NoError(t, err)
	require.NotEmpty(t, out.Reports)
	require.NotNil(t, out.DataStore)
	require.Len(t, out.MCMSTimelockProposals, 1)

	prop := out.MCMSTimelockProposals[0]
	require.Equal(t, mcmsInput.Description, prop.Description)
	require.Equal(t, mcmsInput.ValidUntil, prop.ValidUntil)
	require.Equal(t, mcmsInput.TimelockAction, prop.Action)
	require.Equal(t, "0x01", prop.TimelockAddresses[ethMainnetSelector])
	require.Equal(t, uint64(42), prop.ChainMetadata[ethMainnetSelector].StartingOpCount)
	require.Len(t, prop.Operations, 1)
	require.Len(t, prop.Operations[0].Transactions, 1)
}

func TestNewOnChainChangesetFromSequence_applyBatchOpsWithoutMCMS(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{BatchOps: []mcms_types.BatchOperation{sampleBatchOp()}}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))

	err := cs.VerifyPreconditions(env, WithMCMS[testCfg]{Cfg: cfg})
	require.NoError(t, err)

	_, err = cs.Apply(env, WithMCMS[testCfg]{Cfg: cfg})
	require.Error(t, err)
	require.ErrorIs(t, err, deployment.ErrInvalidConfig)
	require.ErrorIs(t, err, ErrBatchOpsWithoutMCMSInput)
}

func TestNewOnChainChangesetFromSequence_applyBatchOpsWithoutMCMS_preservesMetadata(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}
	ref := datastore.AddressRef{
		Address:       "0xabc",
		ChainSelector: 1,
		Type:          "Timelock",
		Version:       semver.MustParse("1.0.0"),
	}
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			Metadata: datastore.MetadataBundle{Addresses: []datastore.AddressRef{ref}},
			BatchOps: []mcms_types.BatchOperation{sampleBatchOp()},
		}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))

	out, err := cs.Apply(env, WithMCMS[testCfg]{Cfg: cfg})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrBatchOpsWithoutMCMSInput)
	require.NotEmpty(t, out.Reports)
	require.NotNil(t, out.DataStore)

	refs, fetchErr := out.DataStore.Addresses().Fetch()
	require.NoError(t, fetchErr)
	require.Len(t, refs, 1)
	require.Equal(t, "0xabc", refs[0].Address)
}

func TestNewOnChainChangesetFromSequence_applyEmptyBatchOpsWithoutMCMS(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			BatchOps: []mcms_types.BatchOperation{
				{ChainSelector: ethMainnetSelector, Transactions: nil},
			},
		}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))

	out, err := cs.Apply(env, WithMCMS[testCfg]{Cfg: cfg})
	require.NoError(t, err)
	require.NotEmpty(t, out.Reports)
	require.Empty(t, out.MCMSTimelockProposals)
}

func TestNewOnChainChangesetFromSequence_applyMCMSBuildFailure(t *testing.T) {
	t.Parallel()

	env := testEnvironment(t)
	cfg := testCfg{}
	mcmsInput := validMCMSProposalInput()
	ref := datastore.AddressRef{
		Address:       "0xabc",
		ChainSelector: 1,
		Type:          "Timelock",
		Version:       semver.MustParse("1.0.0"),
	}
	readerErr := errors.New("timelock ref failed")
	registry := &deployment.MCMSReaderRegistry{}
	require.NoError(t, registry.Register("evm", &failingTimelockReader{err: readerErr}))

	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			Metadata: datastore.MetadataBundle{Addresses: []datastore.AddressRef{ref}},
			BatchOps: []mcms_types.BatchOperation{sampleBatchOp()},
		}, nil
	})
	cs := NewOnChainChangesetFromSequence(
		newTestChangesetParams(seq, nil, nil),
		WithMCMSRegistry(registry),
	)

	out, err := cs.Apply(env, WithMCMS[testCfg]{Cfg: cfg, MCMS: &mcmsInput})
	require.Error(t, err)
	require.ErrorIs(t, err, readerErr)
	require.ErrorContains(t, err, "get timelock ref for chain")
	require.NotEmpty(t, out.Reports)
	require.NotNil(t, out.DataStore)

	refs, fetchErr := out.DataStore.Addresses().Fetch()
	require.NoError(t, fetchErr)
	require.Len(t, refs, 1)
	require.Equal(t, "0xabc", refs[0].Address)
	require.Empty(t, out.MCMSTimelockProposals)
}

func TestNewOnChainChangesetFromSequence_applySequenceError(t *testing.T) {
	t.Parallel()

	seqErr := errors.New("sequence failed")
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{}, seqErr
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))

	out, err := cs.Apply(testEnvironment(t), WithMCMS[testCfg]{Cfg: testCfg{}})
	require.Error(t, err)
	require.ErrorContains(t, err, seqErr.Error())
	require.ErrorContains(t, err, "failed to execute sequence")
	require.NotEmpty(t, out.Reports)
	require.Nil(t, out.DataStore)
}

func TestNewOnChainChangesetFromSequence_applyWriteMetadataError(t *testing.T) {
	t.Parallel()

	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{
			Metadata: datastore.MetadataBundle{
				Addresses: []datastore.AddressRef{{
					Address:       "0xabc",
					ChainSelector: 1,
					Type:          "Timelock",
				}},
			},
		}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq, nil, nil))

	out, err := cs.Apply(testEnvironment(t), WithMCMS[testCfg]{Cfg: testCfg{}})
	require.Error(t, err)
	require.ErrorIs(t, err, datastore.ErrAddressRefVersionRequired)
	require.ErrorContains(t, err, "failed to write metadata to datastore")
	require.NotEmpty(t, out.Reports)
	require.Nil(t, out.DataStore)
}

func TestNewOnChainChangesetFromSequence_applyResolveErrorEmptyOutput(t *testing.T) {
	t.Parallel()

	resolveErr := errors.New("bad config")
	seq := testSequence(t, func(_ operations.Bundle, _ struct{}, _ struct{}) (OnChainOutput, error) {
		return OnChainOutput{}, nil
	})
	cs := NewOnChainChangesetFromSequence(newTestChangesetParams(seq,
		func(deployment.Environment, testCfg) (struct{}, error) {
			return struct{}{}, resolveErr
		},
		func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil },
	))

	out, err := cs.Apply(testEnvironment(t), WithMCMS[testCfg]{Cfg: testCfg{}})
	require.Error(t, err)
	require.ErrorIs(t, err, deployment.ErrInvalidConfig)
	require.Empty(t, out.Reports)
	require.Nil(t, out.DataStore)
}

func newTestChangesetParams(
	seq *operations.Sequence[struct{}, OnChainOutput, struct{}],
	resolveInput func(deployment.Environment, testCfg) (struct{}, error),
	resolveDep func(deployment.Environment, testCfg) (struct{}, error),
) NewOnChainChangesetFromSequenceParams[struct{}, struct{}, testCfg] {
	if resolveInput == nil {
		resolveInput = func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil }
	}
	if resolveDep == nil {
		resolveDep = func(deployment.Environment, testCfg) (struct{}, error) { return struct{}{}, nil }
	}

	return NewOnChainChangesetFromSequenceParams[struct{}, struct{}, testCfg]{
		Sequence:     seq,
		ResolveInput: resolveInput,
		ResolveDep:   resolveDep,
	}
}

func testEnvironment(t *testing.T) deployment.Environment {
	t.Helper()

	lggr := logger.Test(t)

	return deployment.Environment{
		Logger:     lggr,
		GetContext: func() context.Context { return t.Context() },
		OperationsBundle: operations.NewBundle(
			func() context.Context { return t.Context() },
			lggr,
			operations.NewMemoryReporter(),
		),
	}
}

func testSequence(
	t *testing.T,
	handler operations.SequenceHandler[struct{}, OnChainOutput, struct{}],
) *operations.Sequence[struct{}, OnChainOutput, struct{}] {
	t.Helper()

	return operations.NewSequence(
		"test-seq",
		semver.MustParse("1.0.0"),
		"test sequence",
		handler,
	)
}

func validMCMSProposalInput() deployment.MCMSTimelockProposalInput {
	return deployment.MCMSTimelockProposalInput{
		TimelockAction: mcms_types.TimelockActionSchedule,
		ValidUntil:     testValidUntilUnix,
		TimelockDelay:  mcms_types.NewDuration(time.Hour),
		Description:    "proposal",
	}
}

func sampleBatchOp() mcms_types.BatchOperation {
	return mcms_types.BatchOperation{
		ChainSelector: ethMainnetSelector,
		Transactions: []mcms_types.Transaction{
			{To: "0x01", Data: []byte("0xdeadbeef"), AdditionalFields: json.RawMessage("{}")},
		},
	}
}

func testMCMSRegistry(t *testing.T) *deployment.MCMSReaderRegistry {
	t.Helper()

	r := &deployment.MCMSReaderRegistry{}
	require.NoError(t, r.Register("evm", &testMCMSReader{}))

	return r
}

type testMCMSReader struct{}

func (m *testMCMSReader) GetChainMetadata(_ deployment.Environment, _ uint64, _ deployment.MCMSTimelockProposalInput) (mcms_types.ChainMetadata, error) {
	return mcms_types.ChainMetadata{StartingOpCount: 42}, nil
}

func (m *testMCMSReader) GetTimelockRef(_ deployment.Environment, selector uint64, _ deployment.MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{
		ChainSelector: selector,
		Address:       "0x01",
		Type:          datastore.ContractType("Timelock"),
		Version:       semver.MustParse("1.0.0"),
	}, nil
}

func (m *testMCMSReader) GetMCMSRef(_ deployment.Environment, selector uint64, _ deployment.MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{
		ChainSelector: selector,
		Address:       "0x02",
		Type:          datastore.ContractType("MCM"),
		Version:       semver.MustParse("1.0.0"),
	}, nil
}

type failingTimelockReader struct {
	err error
}

func (r *failingTimelockReader) GetChainMetadata(_ deployment.Environment, _ uint64, _ deployment.MCMSTimelockProposalInput) (mcms_types.ChainMetadata, error) {
	return mcms_types.ChainMetadata{}, nil
}

func (r *failingTimelockReader) GetTimelockRef(_ deployment.Environment, _ uint64, _ deployment.MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{}, r.err
}

func (r *failingTimelockReader) GetMCMSRef(_ deployment.Environment, _ uint64, _ deployment.MCMSTimelockProposalInput) (datastore.AddressRef, error) {
	return datastore.AddressRef{}, nil
}

func ptr[T any](v T) *T {
	return &v
}
