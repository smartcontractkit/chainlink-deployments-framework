package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func NewLogger() (logger.Logger, error) {
	lggr, err := logger.NewWith(func(cfg *zap.Config) {
		*cfg = zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(zapcore.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	})
	if err != nil {
		return nil, err
	}

	return lggr, nil
}
