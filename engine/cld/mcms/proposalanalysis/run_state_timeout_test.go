package proposalanalysis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

func TestRunWithTimeout_AnalyzeCalledWhenCanAnalyzeTrue(t *testing.T) {
	t.Parallel()

	called := false
	anns, skipped, err := runWithTimeout(
		t.Context(),
		200*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			return true, nil
		},
		func(ctx context.Context) (annotation.Annotations, error) {
			called = true
			return annotation.Annotations{annotation.New("ok", "string", "v")}, nil
		},
	)
	require.NoError(t, err)
	require.False(t, skipped)
	require.True(t, called)
	require.Len(t, anns, 1)
	require.Equal(t, "ok", anns[0].Name())
}

func TestRunWithTimeout_SkipsWhenCanAnalyzeFalse(t *testing.T) {
	t.Parallel()

	called := false
	anns, skipped, err := runWithTimeout(
		t.Context(),
		200*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			return false, nil
		},
		func(ctx context.Context) (annotation.Annotations, error) {
			called = true
			return nil, nil
		},
	)
	require.NoError(t, err)
	require.True(t, skipped)
	require.False(t, called)
	require.Nil(t, anns)
}

func TestRunWithTimeout_PropagatesCanAnalyzeError(t *testing.T) {
	t.Parallel()

	errCanAnalyze := errors.New("can analyze failed")
	called := false
	anns, skipped, err := runWithTimeout(
		t.Context(),
		200*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			return false, errCanAnalyze
		},
		func(ctx context.Context) (annotation.Annotations, error) {
			called = true
			return nil, nil
		},
	)
	require.ErrorIs(t, err, errCanAnalyze)
	require.False(t, skipped)
	require.False(t, called)
	require.Nil(t, anns)
}

func TestRunWithTimeout_TimesOutWhenCanAnalyzeIgnoresContext(t *testing.T) {
	t.Parallel()

	anns, skipped, err := runWithTimeout(
		t.Context(),
		5*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			time.Sleep(80 * time.Millisecond)
			return true, nil
		},
		func(ctx context.Context) (annotation.Annotations, error) {
			return nil, nil
		},
	)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, skipped)
	require.Nil(t, anns)
}

func TestRunWithTimeout_TimesOutWhenAnalyzeIgnoresContext(t *testing.T) {
	t.Parallel()

	anns, skipped, err := runWithTimeout(
		t.Context(),
		5*time.Millisecond,
		func(ctx context.Context) (bool, error) {
			return true, nil
		},
		func(ctx context.Context) (annotation.Annotations, error) {
			time.Sleep(80 * time.Millisecond)
			return annotation.Annotations{annotation.New("late", "string", "value")}, nil
		},
	)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, skipped)
	require.Nil(t, anns)
}
