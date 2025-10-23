package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

// Logger is a basic logging interface implemented by smartcontractkit/chainlink/core/logger.Logger and go.uber.org/zap.SugaredLogger
//
// Loggers should be injected (and usually Named as well): e.g. lggr.Named("<service name>")
//
// Tests
//   - Tests should use a [Test] logger, with [New] being reserved for actual runtime and limited direct testing.
//
// Levels
//   - Fatal: Logs and then calls os.Exit(1). Be careful about using this since it does NOT unwind the stack and may exit uncleanly.
//   - Panic: Unrecoverable error. Example: invariant violation, programmer error
//   - Error: Something bad happened, and it was clearly on the node op side. No need for immediate action though. Example: database write timed out
//   - Warn: Something bad happened, not clear who/what is at fault. Node ops should have a rough look at these once in a while to see whether anything stands out. Example: connection to peer was closed unexpectedly. observation timed out.
//   - Info: High level information. First level we’d expect node ops to look at. Example: entered new epoch with leader, made an observation with value, etc.
//   - Debug: Useful for forensic debugging, but we don't expect nops to look at this. Example: Got a message, dropped a message, ...
//
// Node Operator Docs: https://docs.chain.link/docs/configuration-variables/#log_level
// This is a copy of the Logger interface from chainlink-common so we don't have to import the entire module in this repo.
type Logger interface {
	// Name returns the fully qualified name of the logger.
	Name() string

	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Panic(args ...any)
	// Fatal logs and then calls os.Exit(1)
	// Be careful about using this since it does NOT unwind the stack and may exit uncleanly
	Fatal(args ...any)

	Debugf(format string, values ...any)
	Infof(format string, values ...any)
	Warnf(format string, values ...any)
	Errorf(format string, values ...any)
	Panicf(format string, values ...any)
	Fatalf(format string, values ...any)

	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Panicw(msg string, keysAndValues ...any)
	Fatalw(msg string, keysAndValues ...any)

	// Sync flushes any buffered log entries.
	// Some insignificant errors are suppressed.
	Sync() error
}

type Config struct {
	Level zapcore.Level
}

var defaultConfig Config

// New returns a new Logger with the default configuration.
func New() (Logger, error) { return defaultConfig.New() }

// New returns a new Logger for Config.
func (c *Config) New() (Logger, error) {
	return NewWith(func(cfg *zap.Config) {
		cfg.Level.SetLevel(c.Level)
	})
}

// NewWith returns a new Logger from a modified [zap.Config].
func NewWith(cfgFn func(*zap.Config)) (Logger, error) {
	cfg := zap.NewProductionConfig()
	cfgFn(&cfg)
	core, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &logger{core.Sugar()}, nil
}

// Test returns a new test Logger for tb.
func Test(tb testing.TB) Logger {
	tb.Helper()
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000000000")
	lggr := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zaptest.NewTestingWriter(tb),
			zapcore.DebugLevel,
		),
	)

	return &logger{lggr.Sugar()}
}

// TestObserved returns a new test Logger for tb and ObservedLogs at the given Level.
func TestObserved(tb testing.TB, lvl zapcore.Level) (Logger, *observer.ObservedLogs) {
	tb.Helper()
	sl, logs := testObserved(tb, lvl)

	return &logger{sl}, logs
}

func testObserved(tb testing.TB, lvl zapcore.Level) (*zap.SugaredLogger, *observer.ObservedLogs) {
	tb.Helper()
	oCore, logs := observer.New(lvl)
	observe := zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewTee(c, oCore)
	})

	return zaptest.NewLogger(tb, zaptest.WrapOptions(observe, zap.AddCaller())).Sugar(), logs
}

// Nop returns a no-op Logger.
func Nop() Logger {
	return &logger{zap.New(zapcore.NewNopCore()).Sugar()}
}

type logger struct {
	*zap.SugaredLogger
}

func (l *logger) Name() string {
	return l.Desugar().Name()
}
