package logger

import (
	"fmt"
	"log"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

type PrintfLogger interface {
	Printf(string, ...any)
}

type Logger interface {
	PrintfLogger
	Debug(msg string, fields ...interface{})
	Debugf(msg string, args ...interface{})
	Info(msg string, fields ...interface{})
	Infof(msg string, args ...interface{})
	Warn(msg string, fields ...interface{})
	Warnf(msg string, args ...interface{})
	Error(msg string, fields ...interface{})
	Errorf(msg string, args ...interface{})
	Fatal(msg string, fields ...interface{})
	Fatalf(msg string, args ...interface{})
	Logf(msg string, args ...interface{})
}

type ZapLogger struct {
	logger       *zap.Logger
	loggerConfig zap.Config
}

type optionFunc func(*ZapLogger)

func NewZapLogger(opts ...optionFunc) (*ZapLogger, error) {
	loggerConfig := zap.NewProductionConfig()
	loggerZap := &ZapLogger{loggerConfig: loggerConfig}
	for _, opt := range opts {
		opt(loggerZap)
	}
	var err error
	loggerZap.logger, err = loggerZap.loggerConfig.Build()
	if err != nil {
		return nil, err
	}
	return loggerZap, nil
}

func NewZapLoggerForTest(t *testing.T) Logger {
	return &ZapLogger{
		logger: zaptest.NewLogger(t),
	}
}

func WithLevel(level zapcore.Level) optionFunc {
	return func(zl *ZapLogger) {
		zl.loggerConfig.Level = zap.NewAtomicLevelAt(level)
	}
}
func WithEncodeTime(timeKey string, timeEncoder zapcore.TimeEncoder) optionFunc {
	return func(zl *ZapLogger) {
		zl.loggerConfig.EncoderConfig.TimeKey = timeKey
		zl.loggerConfig.EncoderConfig.EncodeTime = timeEncoder
	}
}

func (l *ZapLogger) Debug(msg string, fields ...interface{}) {
	l.logger.Sugar().Debugw(msg, fields...)
}

func (l *ZapLogger) Debugf(msg string, args ...interface{}) {
	l.logger.Sugar().Debugf(msg, args...)
}

func (l *ZapLogger) Info(msg string, fields ...interface{}) {
	l.logger.Sugar().Infow(msg, fields...)
}

func (l *ZapLogger) Infof(msg string, args ...interface{}) {
	l.logger.Sugar().Infof(msg, args...)
}

func (l *ZapLogger) Warn(msg string, fields ...interface{}) {
	l.logger.Sugar().Warnw(msg, fields...)
}

func (l *ZapLogger) Warnf(msg string, args ...interface{}) {
	l.logger.Sugar().Warnf(msg, args...)
}

func (l *ZapLogger) Error(msg string, fields ...interface{}) {
	l.logger.Sugar().Errorw(msg, fields...)
}

func (l *ZapLogger) Errorf(msg string, fields ...interface{}) {
	l.logger.Sugar().Errorf(msg, fields...)
}

func (l *ZapLogger) Fatal(msg string, fields ...interface{}) {
	l.logger.Sugar().Fatalw(msg, fields...)
}

func (l *ZapLogger) Fatalf(msg string, fields ...interface{}) {
	l.logger.Sugar().Fatalf(msg, fields...)
}

func (l *ZapLogger) Printf(msg string, args ...interface{}) {
	l.logger.Sugar().Infof(msg, args...)
}

func (l *ZapLogger) Logf(msg string, args ...interface{}) {
	l.logger.Sugar().Infof(msg, args...)
}

func DefaultLogger() Logger {
	logger, _ := NewZapLogger(WithLevel(zap.InfoLevel), WithEncodeTime("timestamp", zapcore.ISO8601TimeEncoder))
	return logger
}

type ZapLoggerAdapter struct {
	zapLogger *log.Logger
}

func (z *ZapLoggerAdapter) Printf(format string, args ...any) {
	z.zapLogger.Printf(format, args...)
}

// StdLoggerAdapter adapts a standard library logger to the PrintfLogger interface.
type StdLoggerAdapter struct {
	stdLogger *log.Logger
}

// Printf implements the PrintfLogger interface using the standard library logger.
func (a *StdLoggerAdapter) Printf(format string, args ...any) {
	a.stdLogger.Printf(format, args...)
}

// DefaultPrintfLogger initializes a default logger that wraps a standard library logger for Printf logging,
// and integrates it with Zap for structured logging.
//func DefaultPrintfLogger(stdLogger *log.Logger) Logger {
//	// Wrap the standard library logger in an adapter to match the PrintfLogger interface.
//	adapter := &StdLoggerAdapter{stdLogger: stdLogger}
//
//	// Use the adapter to create a ZapLogger that uses the standard library logger for Printf logging.
//	// This assumes that ZapLogger can directly use an adapter that implements PrintfLogger.
//	// If ZapLogger needs direct access to *zap.Logger, further adaptation may be required.
//	return &ZapLogger{
//		logger: adapter.stdLogger, // Assuming zap.NewExample() is a placeholder. Use the appropriate method to get a *zap.Logger instance.
//	}
//}

func DefaultPrintfLogger() Logger {
	return &ZapLogger{logger: zap.NewExample(), loggerConfig: zap.NewProductionConfig()}
}

//func DefaultPrintfLogger(l PrintfLogger) Logger {
//	return &ZapLogger{logger: l, loggerConfig: zap.NewProductionConfig()}
//}

func NewLogger(loggerType string, loggerLevel string) (Logger, error) {
	var err error

	switch loggerType {
	case "zap":
		var zapLevel zapcore.Level
		zapLevel, err = zapcore.ParseLevel(loggerLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to parse logger level: %v", err)
		}
		return NewZapLogger(WithLevel(zapLevel), WithEncodeTime("timestamp", zapcore.ISO8601TimeEncoder))
	default:
		return nil, fmt.Errorf("unsupported logger type: %s", loggerType)
	}
}

type LogWriterAdapter struct {
	*log.Logger
}

// Write implements the io.Writer interface.
func (lwa *LogWriterAdapter) Write(p []byte) (n int, err error) {
	message := string(p)                // Convert byte slice to string
	err = lwa.Logger.Output(2, message) // Use Output with call depth of 2
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Sync implements the zapcore.WriteSyncer interface.
func (lwa *LogWriterAdapter) Sync() error {
	return nil // *log.Logger does not support syncing, so this is a no-op.
}

// NewZapLoggerWithStdLog integrates *log.Logger with Zap.
func NewZapLoggerWithStdLog(stdLogger *log.Logger) *ZapLogger {
	writeSyncer := &LogWriterAdapter{Logger: stdLogger}

	// Setup the core with the custom WriteSyncer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), // or NewConsoleEncoder for console output
		writeSyncer,
		zap.InfoLevel,
	)

	// Create the Zap logger with the custom core
	logger := zap.New(core)

	return &ZapLogger{logger: logger}
}
