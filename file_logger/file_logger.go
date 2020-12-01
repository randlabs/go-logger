package file_logger

//goland:noinspection ALL
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

	consolelogger "github.com/randlabs/go-simple-log/console"
)

//------------------------------------------------------------------------------

var newLine = "\n"

// FileLogOptions ...
type FileLogOptions struct {
	Directory     string `json:"dir,omitempty"`
	DaysToKeep    uint `json:"daysToKeep,omitempty"`
}

type FileLogLogger struct {
	mtx        sync.Mutex
	fd           *os.File
	lastWasError int32
	directory    string
	daysToKeep   uint
	appName      string
	dayOfFile    int
}

//------------------------------------------------------------------------------

func init() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
	return
}

func CreateFileLogger(appName string, options *FileLogOptions) (*FileLogLogger, error) {
	var err error

	fileLogger := &FileLogLogger{}
	fileLogger.appName = appName
	fileLogger.dayOfFile = -1

	if options.DaysToKeep < 365 {
		fileLogger.daysToKeep = options.DaysToKeep
	} else {
		fileLogger.daysToKeep = 365
	}

	if len(options.Directory) > 0 {
		fileLogger.directory = options.Directory
	} else {
		fileLogger.directory = "logs"
	}

	if !filepath.IsAbs(fileLogger.directory) {
		var exeFileName string

		exeFileName, err = os.Executable()
		if err != nil {
			return nil, err
		}

		fileLogger.directory = filepath.Join(filepath.Dir(exeFileName), fileLogger.directory)
	}
	fileLogger.directory = filepath.Clean(fileLogger.directory)
	if !strings.HasSuffix(fileLogger.directory, string(filepath.Separator)) {
		fileLogger.directory += string(filepath.Separator)
	}

	//cleanup
	fileLogger.cleanOldFiles()

	//done
	return fileLogger, nil
}

func (logger *FileLogLogger) Shutdown() {
	logger.mtx.Lock()
	if logger.fd != nil {
		_ = logger.fd.Sync()
		_ = logger.fd.Close()
		logger.fd = nil
	}
	logger.mtx.Unlock()
}

// Error ...
func (logger *FileLogLogger) Error(now time.Time, msg string) {
	logger.write(now, "ERROR", msg)
}

// Warn ...
func (logger *FileLogLogger) Warn(now time.Time, msg string) {
	logger.write(now, "WARN", msg)
}

// Info ...
func (logger *FileLogLogger) Info(now time.Time, msg string) {
	logger.write(now, "INFO", msg)
}

// Debug ...
func (logger *FileLogLogger) Debug(now time.Time, msg string) {
	logger.write(now, "DEBUG", msg)
}

//------------------------------------------------------------------------------
// Private methods

func (logger *FileLogLogger) write(now time.Time, title string, msg string) {
	var err error

	logger.mtx.Lock()

	if logger.fd == nil || now.Day() != logger.dayOfFile {
		var filename string

		if logger.fd != nil {
			_ = logger.fd.Sync()
			_ = logger.fd.Close()
			logger.fd = nil
		}

		//cleanup
		logger.cleanOldFiles()

		_ = os.MkdirAll(logger.directory, 0755)

		filename = logger.directory + strings.ToLower(logger.appName) + "." + now.Format("2006-01-02") + ".log"

		logger.fd, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err == nil {
			logger.dayOfFile = now.Day()
		}
	}
	if err == nil {
		_, err = logger.fd.WriteString(now.Format("2006-01-02 15:04:05") + " [" + title + "]: " + msg + newLine)
	}

	logger.mtx.Unlock()

	if err == nil {
		atomic.StoreInt32(&logger.lastWasError, 0)
	} else {
		if atomic.CompareAndSwapInt32(&logger.lastWasError, 0, 1) {
			consolelogger.Error(now, fmt.Sprintf("Unable to save notification in file [%v]", err))
		}
	}
}

func (logger *FileLogLogger) cleanOldFiles() {
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
