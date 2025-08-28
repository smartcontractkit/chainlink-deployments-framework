package cli

import (
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewCLILogger(logLevel zapcore.Level) (logger.Logger, error) {
	lggr, err := logger.NewWith(func(cfg *zap.Config) {
		*cfg = zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(logLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	})
	if err != nil {
		return nil, err
	}

	return lggr, nil
}
