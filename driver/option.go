package driver

import (
	"strings"
	"time"

	dlog "github.com/starton-io/go-dispatch/logger"
)

const (
	OptionTypeTimeout      = 0x600
	OptionTypeLogger       = 0x601
	OptionTypeGlobalPrefix = 0x602
)

type Option interface {
	Type() int
}

type TimeoutOption struct{ timeout time.Duration }

func (to TimeoutOption) Type() int                         { return OptionTypeTimeout }
func NewTimeoutOption(timeout time.Duration) TimeoutOption { return TimeoutOption{timeout: timeout} }

type LoggerOption struct{ logger dlog.Logger }

func (to LoggerOption) Type() int                     { return OptionTypeLogger }
func NewLoggerOption(logger dlog.Logger) LoggerOption { return LoggerOption{logger: logger} }

type GlobalPrefixOption struct {
	globalPrefix string
}

func (to GlobalPrefixOption) Type() int {
	return OptionTypeGlobalPrefix
}

func NewGlobalPrefixOption(prefix string) GlobalPrefixOption {
	if prefix == "" {
		prefix = DefaultGlobalKeyPrefix
	} else if !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	return GlobalPrefixOption{globalPrefix: prefix}
}
