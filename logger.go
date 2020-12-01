package go_logger

//goland:noinspection ALL
import (
	"errors"
	"fmt"
	"time"

	consolelogger "github.com/randlabs/go-simple-log/console"
	filelogger "github.com/randlabs/go-simple-log/file_logger"
	sysloglogger "github.com/randlabs/go-simple-log/syslog_logger"
)

//------------------------------------------------------------------------------

// Options ...
type Options struct {
	AppName       string `json:"appName"`
	FileLog       *filelogger.FileLogOptions `json:"fileLog,omitempty"`
	SysLog        *sysloglogger.SysLogOptions `json:"sysLog,omitempty"`
	DebugLevel    uint `json:"debugLevel,omitempty"`
	UseLocalTime  bool `json:"useLocalTime,omitempty"`
}

type loggerData struct {
	appName      string
	fileLogger   *filelogger.FileLogLogger
	syslogLogger *sysloglogger.SysLogLogger
	debugLevel   uint
	useLocalTime bool
}

//------------------------------------------------------------------------------

var logger *loggerData

//------------------------------------------------------------------------------

// Initialize ...
func Initialize(options Options) error {
	var err error

	newLogger := &loggerData{}

	if len(options.AppName) == 0 {
		return errors.New("invalid application name")
	}
	newLogger.appName = options.AppName
	newLogger.debugLevel = options.DebugLevel
	newLogger.useLocalTime = options.UseLocalTime

	if options.FileLog != nil {
		newLogger.fileLogger, err = filelogger.CreateFileLogger(newLogger.appName, options.FileLog)
		if err != nil {
			newLogger.destroy()
			return err
		}
	}

	if options.SysLog != nil {
		newLogger.syslogLogger, err = sysloglogger.CreateSysLogLogger(newLogger.appName, options.SysLog)
		if err != nil {
			newLogger.destroy()
			return err
		}
	}

	logger = newLogger

	//done
	return nil
}

// Finalize ...
func Finalize() {
	if logger != nil {
		logger.destroy()

		logger = nil
	}
}

// SetDebugLevel ...
func SetDebugLevel(level uint) {
	if logger != nil {
		logger.debugLevel = level
	}
}

// Error ...
func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := getTimestamp()

	if logger != nil {
		if logger.fileLogger != nil {
			logger.fileLogger.Error(now, msg)
		}
		if logger.syslogLogger != nil {
			logger.syslogLogger.Error(now, msg)
		}
	}
	consolelogger.Error(now, msg)
}

// Warn ...
func Warn(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := getTimestamp()

	if logger != nil {
		if logger.fileLogger != nil {
			logger.fileLogger.Warn(now, msg)
		}
		if logger.syslogLogger != nil {
			logger.syslogLogger.Warn(now, msg)
		}
	}
	consolelogger.Warn(now, msg)
}

// Info ...
func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := getTimestamp()

	if logger != nil {
		if logger.fileLogger != nil {
			logger.fileLogger.Info(now, msg)
		}
		if logger.syslogLogger != nil {
			logger.syslogLogger.Info(now, msg)
		}
	}
	consolelogger.Info(now, msg)
}

// Debug ...
func Debug(level uint, format string, a ...interface{}) {
	if logger != nil && level > logger.debugLevel {
		return
	}

	msg := fmt.Sprintf(format, a...)
	now := getTimestamp()

	if logger != nil {
		if logger.fileLogger != nil {
			logger.fileLogger.Debug(now, msg)
		}
	}
	consolelogger.Debug(now, msg)
}

//------------------------------------------------------------------------------
// Private methods

func getTimestamp() time.Time {
	now := time.Now()
	if logger == nil || (!logger.useLocalTime) {
		now = now.UTC()
	}
	return now
}

func (lg *loggerData) destroy() {
	if lg.syslogLogger != nil {
		lg.syslogLogger.Shutdown()
		lg.syslogLogger = nil
	}
	if lg.fileLogger != nil {
		lg.fileLogger.Shutdown()
		lg.fileLogger = nil
	}
}
