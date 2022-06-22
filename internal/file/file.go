package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/randlabs/go-logger/internal/util"
)

//------------------------------------------------------------------------------

var newLine = "\n"

//------------------------------------------------------------------------------

// Logger is the object that controls file logging.
type Logger struct {
	mtx          sync.Mutex
	fd           *os.File
	lastWasError int32
	directory    string
	daysToKeep   uint
	prefix       string
	dayOfFile    int
	errorHandler ErrorHandler
}

// Options specifies the file logger settings to use when it is created.
type Options struct {
	// Filename prefix to use when a file is created. Defaults to the binary name.
	Prefix       string `json:"prefix,omitempty"`

	// Destination directory to store log files.
	Directory    string `json:"dir,omitempty"`

	// Amount of days to keep old logs.
	DaysToKeep   uint   `json:"daysToKeep,omitempty"`

	// A callback to call if an internal error is encountered.
	ErrorHandler ErrorHandler
}

// ErrorHandler is a callback to call if an internal error must be notified.
type ErrorHandler func(message string)

//------------------------------------------------------------------------------

func init() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
}

//------------------------------------------------------------------------------

// Create creates a new file logger.
func Create(options Options) (*Logger, error) {
	var err error

	if len(options.Prefix) == 0 {
		// If no prefix was given, use the base name of the executable.
		options.Prefix, err = os.Executable()
		if err != nil {
			return nil, err
		}
		options.Prefix = filepath.Base(options.Prefix)

		extLen := len(filepath.Ext(options.Prefix))
		if len(options.Prefix) > extLen {
			options.Prefix = options.Prefix[:(len(options.Prefix) - extLen)]
		}
	}

	// Create file logger
	logger := &Logger{
		prefix:       options.Prefix,
		dayOfFile:    -1,
		errorHandler: options.ErrorHandler,
	}

	// Set the number of days to keep the old files
	if options.DaysToKeep < 365 {
		logger.daysToKeep = options.DaysToKeep
	} else {
		logger.daysToKeep = 365
	}

	// Establishes the target directory
	if len(options.Directory) > 0 {
		logger.directory = filepath.ToSlash(options.Directory)
	} else {
		logger.directory = "logs"
	}

	if !filepath.IsAbs(logger.directory) {
		var workingDir string

		workingDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}

		logger.directory = filepath.Join(workingDir, logger.directory)
	}
	logger.directory = filepath.Clean(logger.directory)
	if !strings.HasSuffix(logger.directory, string(filepath.Separator)) {
		logger.directory += string(filepath.Separator)
	}

	// Delete old files
	logger.cleanOldFiles()

	// Done
	return logger, nil
}

// Destroy shuts down the file logger.
func (logger *Logger) Destroy() {
	logger.mtx.Lock()
	if logger.fd != nil {
		_ = logger.fd.Sync()
		_ = logger.fd.Close()
		logger.fd = nil
	}
	logger.mtx.Unlock()
}

// LogError saves an error message to in the file.
func (logger *Logger) LogError(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logger.write(now, "ERROR", msg)
	} else {
		logger.writeJSON(now, "error", msg)
	}
}

// LogWarning saves a warning message to in the file.
func (logger *Logger) LogWarning(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logger.write(now, "WARNING", msg)
	} else {
		logger.writeJSON(now, "warning", msg)
	}
}

// LogInfo saves an information message to in the file.
func (logger *Logger) LogInfo(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logger.write(now, "INFO", msg)
	} else {
		logger.writeJSON(now, "info", msg)
	}
}

// LogDebug saves a debug message to in the file.
func (logger *Logger) LogDebug(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logger.write(now, "DEBUG", msg)
	} else {
		logger.writeJSON(now, "debug", msg)
	}
}

//------------------------------------------------------------------------------
// Private methods

func (logger *Logger) write(now time.Time, level string, msg string) {
	// Lock access
	logger.mtx.Lock()

	err := logger.openOrRotateFile(now)
	if err == nil {
		// Save message to file
		_, err = logger.fd.WriteString(now.Format("2006-01-02 15:04:05.000") + " [" + level + "]: " + msg + newLine)
	}

	// Unlock access
	logger.mtx.Unlock()

	// Handle error
	logger.handleLoggingError(err)
}

func (logger *Logger) writeJSON(now time.Time, level string, msg string) {
	// Lock access
	logger.mtx.Lock()

	err := logger.openOrRotateFile(now)
	if err == nil {
		// Save message to file
		_, err = logger.fd.WriteString(util.AddPayloadToJSON(msg, &now, strings.ToLower(level)) + newLine)
	}

	// Unlock access
	logger.mtx.Unlock()

	// Handle error
	logger.handleLoggingError(err)
}

func (logger *Logger) openOrRotateFile(now time.Time) error {
	// Check if we have to rotate files
	if logger.fd == nil || now.Day() != logger.dayOfFile {
		var err error

		if logger.fd != nil {
			_ = logger.fd.Sync()
			_ = logger.fd.Close()
			logger.fd = nil
		}

		// Delete old files
		logger.cleanOldFiles()

		// Create target directory if it does not exist
		_ = os.MkdirAll(logger.directory, 0755)

		filename := logger.directory + strings.ToLower(logger.prefix) + "." + now.Format("2006-01-02") + ".log"

		// Create a new log file
		logger.fd, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}

		logger.dayOfFile = now.Day()
	}

	// Done
	return nil
}

func (logger *Logger) handleLoggingError(err error) {
	// Handle error
	if err == nil {
		atomic.StoreInt32(&logger.lastWasError, 0)
	} else {
		if atomic.CompareAndSwapInt32(&logger.lastWasError, 0, 1) && logger.errorHandler != nil {
			logger.errorHandler(fmt.Sprintf("Unable to save notification in file [%v]", err))
		}
	}
}

func (logger *Logger) cleanOldFiles() {
	if logger.daysToKeep > 0 {
		lowestTime := time.Now().UTC().AddDate(0, 0, -(int(logger.daysToKeep)))

		files, err := ioutil.ReadDir(logger.directory)
		if err == nil {
			for _, f := range files {
				if !f.IsDir() {
					var nameLC = strings.ToLower(f.Name())

					if (!f.IsDir()) && strings.HasSuffix(nameLC, ".log") {

						if getFileCreationtime(f).Before(lowestTime) {
							_ = os.Remove(logger.directory + f.Name())
						}
					}
				}
			}
		}
	}
}
