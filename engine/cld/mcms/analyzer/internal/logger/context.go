package logger

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type contextKey string

const loggerKey contextKey = "logger"

func ContextWithLogger(ctx context.Context, lggr logger.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, lggr)
}

func FromContext(ctx context.Context) logger.Logger {
	lggr, found := ctx.Value(loggerKey).(logger.Logger)
	if !found {
		lggr, _ = NewLogger()
	}

	return lggr
}
