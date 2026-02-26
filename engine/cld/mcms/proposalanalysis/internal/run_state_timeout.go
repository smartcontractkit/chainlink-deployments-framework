package proposalanalysis

import (
	"context"
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"
)

func runWithTimeout(
	ctx context.Context,
	timeout time.Duration,
	canAnalyzeFn func(context.Context) bool,
	analyzeFn func(context.Context) (annotation.Annotations, error),
) (annotation.Annotations, bool, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	canAnalyze, err := callCanAnalyze(runCtx, canAnalyzeFn)
	if err != nil {
		return nil, false, err
	}
	if !canAnalyze {
		return nil, true, nil
	}

	anns, err := callAnalyze(runCtx, analyzeFn)
	if err != nil {
		return nil, false, err
	}

	return anns, false, nil
}

func callCanAnalyze(ctx context.Context, fn func(context.Context) bool) (bool, error) {
	done := make(chan bool, 1)
	go func() {
		done <- fn(ctx)
	}()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case canAnalyze := <-done:
		return canAnalyze, nil
	}
}

func callAnalyze(
	ctx context.Context,
	fn func(context.Context) (annotation.Annotations, error),
) (annotation.Annotations, error) {
	type analyzeResult struct {
		annotations annotation.Annotations
		err         error
	}

	done := make(chan analyzeResult, 1)
	go func() {
		anns, err := fn(ctx)
		done <- analyzeResult{
			annotations: anns,
			err:         err,
		}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		return result.annotations, result.err
	}
}
