package go_logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//------------------------------------------------------------------------------

// FileOptions specifies the file logger settings to use when it is created.
type FileOptions struct {
	// Filename prefix to use when a file is created. Defaults to the binary name.
	Prefix string `json:"prefix,omitempty"`

	// Destination directory to store log files.
	Directory string `json:"dir,omitempty"`

	// Amount of days to keep old logs.
	DaysToKeep uint   `json:"daysToKeep,omitempty"`

	// Set the initial logging level to use.
	Level *LogLevel `json:"level,omitempty"`

	// Set the initial logging level for debug output to use.
	DebugLevel *uint `json:"debugLevel,omitempty"`
}

type fileAdapter struct {
	mtx          sync.Mutex
	fd           *os.File
	lastWasError int32
	directory    string
	daysToKeep   uint
	prefix       string
	dayOfFile    int
	globals      globalOptions
}

//------------------------------------------------------------------------------

func createFileAdapter(opts FileOptions, glbOpts globalOptions) (internalLogger, error) {
	var err error

	if len(opts.Prefix) == 0 {
		// If no prefix was given, use the base name of the executable.
		opts.Prefix, err = os.Executable()
		if err != nil {
			return nil, err
		}
		opts.Prefix = filepath.Base(opts.Prefix)

		extLen := len(filepath.Ext(opts.Prefix))
		if len(opts.Prefix) > extLen {
			opts.Prefix = opts.Prefix[:(len(opts.Prefix) - extLen)]
		}
	}

	// Create file adapter
	lg := &fileAdapter{
		prefix:    opts.Prefix,
		dayOfFile: -1,
		globals:   glbOpts,
	}

	// Set output level based on globals or overrides
	if opts.Level != nil {
		lg.globals.Level = *opts.Level
		lg.globals.DebugLevel = 1
	}
	if opts.DebugLevel != nil {
		lg.globals.DebugLevel = *opts.DebugLevel
	}

	// Set the number of days to keep the old files
	if opts.DaysToKeep < 365 {
		lg.daysToKeep = opts.DaysToKeep
	} else {
		lg.daysToKeep = 365
	}

	// Establishes the target directory
	if len(opts.Directory) > 0 {
		lg.directory = filepath.ToSlash(opts.Directory)
	} else {
		lg.directory = "logs"
	}

	if !filepath.IsAbs(lg.directory) {
		var workingDir string

		workingDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}

		lg.directory = filepath.Join(workingDir, lg.directory)
	}
	lg.directory = filepath.Clean(lg.directory)
	if !strings.HasSuffix(lg.directory, string(filepath.Separator)) {
		lg.directory += string(filepath.Separator)
	}

	// Delete old files
	lg.cleanOldFiles()

	// Done
	return lg, nil
}

func (lg *fileAdapter) class() string {
	return "file"
}

func (lg *fileAdapter) destroy() {
	lg.mtx.Lock()
	if lg.fd != nil {
		_ = lg.fd.Sync()
		_ = lg.fd.Close()
		lg.fd = nil
	}
	lg.mtx.Unlock()
}

func (lg *fileAdapter) setLevel(level LogLevel, debugLevel uint) {
	lg.globals.Level = level
	lg.globals.DebugLevel = debugLevel
}

func (lg *fileAdapter) logError(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelError {
		if !raw {
			lg.write(now, "ERROR", msg)
		} else {
			lg.writeRAW(now, msg)
		}
	}
}

func (lg *fileAdapter) logWarning(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelWarning {
		if !raw {
			lg.write(now, "WARNING", msg)
		} else {
			lg.writeRAW(now, msg)
		}
	}
}

func (lg *fileAdapter) logInfo(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelInfo {
		if !raw {
			lg.write(now, "INFO", msg)
		} else {
			lg.writeRAW(now, msg)
		}
	}
}

func (lg *fileAdapter) logDebug(level uint, now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelDebug && lg.globals.DebugLevel >= level {
		if !raw {
			lg.write(now, "DEBUG", msg)
		} else {
			lg.writeRAW(now, msg)
		}
	}
}

func (lg *fileAdapter) write(now time.Time, level string, msg string) {
	// Lock access
	lg.mtx.Lock()

	err := lg.openOrRotateFile(now)
	if err == nil {
		// Save message to file
		_, err = lg.fd.WriteString(now.Format("2006-01-02 15:04:05.000") + " [" + level + "]: " + msg + newLine)
	}

	// Unlock access
	lg.mtx.Unlock()

	// Handle error
	lg.handleLoggingError(err)
}

func (lg *fileAdapter) writeRAW(now time.Time, msg string) {
	// Lock access
	lg.mtx.Lock()

	err := lg.openOrRotateFile(now)
	if err == nil {
		// Save message to file
		_, err = lg.fd.WriteString(msg + newLine)
	}

	// Unlock access
	lg.mtx.Unlock()

	// Handle error
	lg.handleLoggingError(err)
}

func (lg *fileAdapter) openOrRotateFile(now time.Time) error {
	// Check if we have to rotate files
	if lg.fd == nil || now.Day() != lg.dayOfFile {
		var err error

		if lg.fd != nil {
			_ = lg.fd.Sync()
			_ = lg.fd.Close()
			lg.fd = nil
		}

		// Delete old files
		lg.cleanOldFiles()

		// Create target directory if it does not exist
		_ = os.MkdirAll(lg.directory, 0755)

		filename := lg.directory + strings.ToLower(lg.prefix) + "." + now.Format("2006-01-02") + ".log"

		// Create a new log file
		lg.fd, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			return err
		}

		lg.dayOfFile = now.Day()
	}

	// Done
	return nil
}

func (lg *fileAdapter) handleLoggingError(err error) {
	// Handle error
	if err == nil {
		atomic.StoreInt32(&lg.lastWasError, 0)
	} else {
		if atomic.CompareAndSwapInt32(&lg.lastWasError, 0, 1) && lg.globals.ErrorHandler != nil {
			lg.globals.ErrorHandler(fmt.Sprintf("Unable to save notification in file [%v]", err))
		}
	}
}

func (lg *fileAdapter) cleanOldFiles() {
	if lg.daysToKeep > 0 {
		lowestTime := time.Now().UTC().AddDate(0, 0, -(int(lg.daysToKeep)))

		files, err := ioutil.ReadDir(lg.directory)
		if err == nil {
			for _, f := range files {
				if !f.IsDir() {
					var nameLC = strings.ToLower(f.Name())

					if (!f.IsDir()) && strings.HasSuffix(nameLC, ".log") {
						f.ModTime()

						if getFileCreationTime(f).Before(lowestTime) {
							_ = os.Remove(lg.directory + f.Name())
						}
					}
				}
			}
		}
	}
}
