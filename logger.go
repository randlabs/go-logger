package go_logger

import (
	"sync"

	"github.com/randlabs/go-logger/internal/console"
	"github.com/randlabs/go-logger/internal/file"
	"github.com/randlabs/go-logger/internal/syslog"
)

//------------------------------------------------------------------------------

// LogLevel defines the level of message verbosity.
type LogLevel uint

const (
	LogLevelQuiet   LogLevel = 0
	LogLevelError            = 1
	LogLevelWarning          = 2
	LogLevelInfo             = 3
	LogLevelDebug            = 4
)

// Options specifies the logger settings to use when initialized.
type Options struct {
	DisableConsole bool           `json:"disableConsole,omitempty"`
	FileLog        *FileOptions   `json:"fileLog,omitempty"`
	SysLog         *SyslogOptions `json:"sysLog,omitempty"`
	Level          LogLevel       `json:"level,omitempty"`
	DebugLevel     uint           `json:"debugLevel,omitempty"`
	UseLocalTime   bool           `json:"useLocalTime,omitempty"`
	ErrorHandler   ErrorHandler
}

// ErrorHandler is a callback to call if an internal error must be notified.
type ErrorHandler func(message string)

type FileOptions file.Options
type SyslogOptions syslog.Options

// Logger is the object that controls logging.
type Logger struct {
	mtx            sync.RWMutex
	level          LogLevel
	debugLevel     uint
	disableConsole bool
	file           *file.Logger
	syslog         *syslog.Logger
	useLocalTime   bool
	errorHandler   ErrorHandler
}

//------------------------------------------------------------------------------

var (
	defaultLoggerInit = sync.Once{}
	defaultLogger     *Logger
)

//------------------------------------------------------------------------------

func Default() *Logger {
	defaultLoggerInit.Do(func() {
		defaultLogger, _ = Create(Options{
			Level: LogLevelWarning,
		})
	})
	return defaultLogger
}

// Create creates a new logger.
func Create(options Options) (*Logger, error) {
	var err error

	// Create file logger
	logger := &Logger{
		mtx:            sync.RWMutex{},
		level:          options.Level,
		debugLevel:     options.DebugLevel,
		disableConsole: options.DisableConsole,
		errorHandler:   options.ErrorHandler,
	}

	// Create file logger if options were specified
	if options.FileLog != nil {
		fileOpts := file.Options(*options.FileLog)
		if fileOpts.ErrorHandler == nil && logger.errorHandler != nil {
			fileOpts.ErrorHandler = logger.forwardLogError
		}

		logger.file, err = file.Create(fileOpts)
		if err != nil {
			logger.Destroy()
			return nil, err
		}
	}

	// Create syslog logger if options were specified
	if options.SysLog != nil {
		syslogOpts := syslog.Options(*options.SysLog)
		if syslogOpts.ErrorHandler == nil && logger.errorHandler != nil {
			syslogOpts.ErrorHandler = logger.forwardLogError
		}

		logger.syslog, err = syslog.Create(syslogOpts)
		if err != nil {
			logger.Destroy()
			return nil, err
		}
	}

	// Done
	return logger, nil
}

// Destroy shuts down the logger.
func (logger *Logger) Destroy() {
	// The default logger cannot be destroyed
	if logger == defaultLogger {
		return
	}

	if logger.syslog != nil {
		logger.syslog.Destroy()
		logger.syslog = nil
	}
	if logger.file != nil {
		logger.file.Destroy()
		logger.file = nil
	}
}

// SetLevel sets the minimum level for all messages.
func (logger *Logger) SetLevel(newLevel LogLevel) {
	// Lock access
	logger.mtx.Lock()
	defer logger.mtx.Unlock()

	logger.level = newLevel
}

// SetDebugLevel sets the minimum level for debug messages.
func (logger *Logger) SetDebugLevel(newLevel uint) {
	// Lock access
	logger.mtx.Lock()
	defer logger.mtx.Unlock()

	logger.debugLevel = newLevel
}

// Error emits an error message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (logger *Logger) Error(obj interface{}) {
	// Lock access
	logger.mtx.RLock()

	if logger.level >= LogLevelError {
		msg, isJSON, ok := logger.parseObj(obj)
		if ok {
			now := logger.getTimestamp()

			if !logger.disableConsole {
				console.LogError(now, msg, isJSON)
			}
			if logger.file != nil {
				logger.file.LogError(now, msg, isJSON)
			}
			if logger.syslog != nil {
				logger.syslog.LogError(now, msg, isJSON)
			}
		}
	}

	// Unlock access
	logger.mtx.RUnlock()
}

// Warning emits a warning message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (logger *Logger) Warning(obj interface{}) {
	// Lock access
	logger.mtx.RLock()

	if logger.level >= LogLevelWarning {
		msg, isJSON, ok := logger.parseObj(obj)
		if ok {
			now := logger.getTimestamp()

			if !logger.disableConsole {
				console.LogWarning(now, msg, isJSON)
			}
			if logger.file != nil {
				logger.file.LogWarning(now, msg, isJSON)
			}
			if logger.syslog != nil {
				logger.syslog.LogWarning(now, msg, isJSON)
			}
		}
	}

	// Unlock access
	logger.mtx.RUnlock()
}

// Info emits an information message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (logger *Logger) Info(obj interface{}) {
	// Lock access
	logger.mtx.RLock()

	if logger.level >= LogLevelInfo {
		msg, isJSON, ok := logger.parseObj(obj)
		if ok {
			now := logger.getTimestamp()

			if !logger.disableConsole {
				console.LogInfo(now, msg, isJSON)
			}
			if logger.file != nil {
				logger.file.LogInfo(now, msg, isJSON)
			}
			if logger.syslog != nil {
				logger.syslog.LogInfo(now, msg, isJSON)
			}
		}
	}

	// Unlock access
	logger.mtx.RUnlock()
}

// Debug emits a debug message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (logger *Logger) Debug(level uint, obj interface{}) {
	// Lock access
	logger.mtx.RLock()

	if logger.level >= LogLevelDebug && logger.debugLevel >= level {
		msg, isJSON, ok := logger.parseObj(obj)
		if ok {
			now := logger.getTimestamp()

			if !logger.disableConsole {
				console.LogDebug(now, msg, isJSON)
			}
			if logger.file != nil {
				logger.file.LogDebug(now, msg, isJSON)
			}
			if logger.syslog != nil {
				logger.syslog.LogDebug(now, msg, isJSON)
			}
		}
	}

	// Unlock access
	logger.mtx.RUnlock()
}
