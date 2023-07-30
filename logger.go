package go_logger

import (
	"sync"
)

//------------------------------------------------------------------------------

// LogLevel defines the level of message verbosity.
type LogLevel uint

const (
	LogLevelQuiet   LogLevel = 0
	LogLevelError   LogLevel = 1
	LogLevelWarning LogLevel = 2
	LogLevelInfo    LogLevel = 3
	LogLevelDebug   LogLevel = 4
)

// Logger is the object that controls logging.
type Logger struct {
	mtx            sync.RWMutex
	//level          LogLevel
	//debugLevel     uint
	//disableConsole bool
	adapters       []internalLogger
	useLocalTime   bool
}

// Options specifies the logger settings to use when initialized.
type Options struct {
	// Optionally establish console logging settings.
	Console ConsoleOptions `json:"console,omitempty"`

	// Optionally enable file logging and establish its settings.
	File *FileOptions `json:"file,omitempty"`

	// Optionally enable syslog logging and establish its settings.
	SysLog *SysLogOptions `json:"sysLog,omitempty"`

	// Set the initial logging level to use.
	Level LogLevel `json:"level,omitempty"`

	// Set the initial logging level for debug output to use.
	DebugLevel uint `json:"debugLevel,omitempty"`

	// Use the local computer time instead of UTC.
	UseLocalTime  bool `json:"useLocalTime,omitempty"`

	// A callback to call if an internal error is encountered.
	ErrorHandler ErrorHandler
}

// ErrorHandler is a callback to call if an internal error must be notified.
type ErrorHandler func(message string)

//------------------------------------------------------------------------------

var (
	defaultLoggerInit = sync.Once{}
	defaultLogger     *Logger
)

//------------------------------------------------------------------------------

// Default returns a logger that only outputs error and warnings to the console.
func Default() *Logger {
	defaultLoggerInit.Do(func() {
		defaultLogger, _ = Create(Options{
			Level: LogLevelInfo,
		})
	})
	return defaultLogger
}

// WithLevel is a helper to set up a log level override.
func WithLevel(level LogLevel) *LogLevel {
	return &level
}

// WithDebugLevel is a helper to set up a debug log level override.
func WithDebugLevel(debugLevel uint) *uint {
	return &debugLevel
}

// Create creates a new logger.
func Create(opts Options) (*Logger, error) {
	// Create logger
	lg := &Logger{
		mtx:      sync.RWMutex{},
		adapters: make([]internalLogger, 0),
	}

	// Initialize global options
	glbOpts := globalOptions{
		Level:        opts.Level,
		DebugLevel:   opts.DebugLevel,
		ErrorHandler: opts.ErrorHandler,
	}

	// Create console adapter
	if !opts.Console.Disable {
		adapter := createConsoleAdapter(opts.Console, glbOpts)

		// Add to list of adapters
		lg.adapters = append(lg.adapters, adapter)
	}

	// Create file adapter if opts were specified
	if opts.File != nil {
		adapter, err := createFileAdapter(*opts.File, glbOpts)
		if err != nil {
			lg.Destroy()
			return nil, err
		}

		// Add to list of adapters
		lg.adapters = append(lg.adapters, adapter)
	}

	// Create syslog adapter if opts were specified
	if opts.SysLog != nil {
		adapter, err := createSysLogAdapter(*opts.SysLog, glbOpts)
		if err != nil {
			lg.Destroy()
			return nil, err
		}

		// Add to list of adapters
		lg.adapters = append(lg.adapters, adapter)
	}

	// Done
	return lg, nil
}

// Destroy shuts down the logger.
func (lg *Logger) Destroy() {
	// The default logger cannot be destroyed
	if lg == defaultLogger {
		return
	}

	// Destroy all adapters
	for _, adapter := range lg.adapters {
		adapter.destroy()
	}
	lg.adapters = nil
}

// SetLevel sets the minimum level for all messages.
func (lg *Logger) SetLevel(level LogLevel, debugLevel uint, class string) {
	// Lock access
	lg.mtx.Lock()
	defer lg.mtx.Unlock()

	for _, adapter := range lg.adapters {
		if class == "" || class == "all" || class == adapter.class() {
			adapter.setLevel(level, debugLevel)
		}
	}
}

// Error emits an error message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (lg *Logger) Error(obj interface{}) {
	// Lock access
	lg.mtx.RLock()

	msg, isJSON, ok := lg.parseObj(obj)
	if ok {
		now := lg.getTimestamp()
		raw := false
		if isJSON {
			msg = addPayloadToJSON(msg, now, "error")
			raw = true
		}

		for _, adapter := range lg.adapters {
			adapter.logError(now, msg, raw)
		}
	}

	// Unlock access
	lg.mtx.RUnlock()
}

// Warning emits a warning message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (lg *Logger) Warning(obj interface{}) {
	// Lock access
	lg.mtx.RLock()

	msg, isJSON, ok := lg.parseObj(obj)
	if ok {
		now := lg.getTimestamp()
		raw := false
		if isJSON {
			msg = addPayloadToJSON(msg, now, "warning")
			raw = true
		}

		for _, adapter := range lg.adapters {
			adapter.logWarning(now, msg, raw)
		}
	}

	// Unlock access
	lg.mtx.RUnlock()
}

// Info emits an information message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (lg *Logger) Info(obj interface{}) {
	// Lock access
	lg.mtx.RLock()

	msg, isJSON, ok := lg.parseObj(obj)
	if ok {
		now := lg.getTimestamp()
		raw := false
		if isJSON {
			msg = addPayloadToJSON(msg, now, "info")
			raw = true
		}

		for _, adapter := range lg.adapters {
			adapter.logInfo(now, msg, raw)
		}
	}

	// Unlock access
	lg.mtx.RUnlock()
}

// Debug emits a debug message into the configured targets.
// If a string is passed, output format will be in DATE [LEVEL] MESSAGE.
// If a struct is passed, output will be in json with level and timestamp fields automatically added.
func (lg *Logger) Debug(level uint, obj interface{}) {
	// Lock access
	lg.mtx.RLock()

	msg, isJSON, ok := lg.parseObj(obj)
	if ok {
		now := lg.getTimestamp()
		raw := false
		if isJSON {
			msg = addPayloadToJSON(msg, now, "debug")
			raw = true
		}

		for _, adapter := range lg.adapters {
			adapter.logDebug(level, now, msg, raw)
		}
	}

	// Unlock access
	lg.mtx.RUnlock()
}
