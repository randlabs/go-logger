package go_simple_log

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gookit/color"
	swcgo "github.com/randlabs/server-watchdog-go"
)

//------------------------------------------------------------------------------

var appName string
var newLine = "\n"
var fileOptions *FileOptions
var fileMutex sync.Mutex
var logFd *os.File = nil
var dayOfFile int
var swc *swcgo.ServerWatcherClient
var shutdownProcessRegister chan struct{}
var consoleLogger *IConsoleLogger

//------------------------------------------------------------------------------

// Options ...
type Options struct {
	AppName string
	BaseFolder string
	FileOpts *FileOptions               `json:"file,omitempty"`
	ServerWatchdog *swcgo.ClientOptions `json:"serverWatchdog,omitempty"`
	ConsoleLogger *IConsoleLogger
}

// FileOptions ...
type FileOptions struct {
	DaysToKeep uint   `json:"daysToKeep,omitempty"`
	Folder     string `json:"folder,omitempty"`
	UseLocalTime bool `json:"useLocalTime,omitempty"`
}

// IConsoleLogger ...
type IConsoleLogger interface {
	Error(now time.Time, msg string)
	Warn(now time.Time, msg string)
	Info(now time.Time, msg string)
	Debug(now time.Time, msg string)
}

//------------------------------------------------------------------------------

func init() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
	return
}

//------------------------------------------------------------------------------

// Initialize...
func Initialize(options Options) error {
	var err error

	if len(options.AppName) == 0 {
		return errors.New("invalid application name")
	}
	appName = options.AppName

	if options.FileOpts != nil {
		fileOptions = &FileOptions{
			DaysToKeep:   options.FileOpts.DaysToKeep,
			Folder:       options.FileOpts.Folder,
			UseLocalTime: options.FileOpts.UseLocalTime,
		}
		if len(options.FileOpts.Folder) > 0 {
			fileOptions.Folder = options.FileOpts.Folder
		} else {
			fileOptions.Folder = "logs"
		}

		if !filepath.IsAbs(fileOptions.Folder) {
			fileOptions.Folder = filepath.Join(options.BaseFolder, fileOptions.Folder)
		}
		fileOptions.Folder = filepath.Clean(fileOptions.Folder)
		if !strings.HasSuffix(fileOptions.Folder, string(filepath.Separator)) {
			fileOptions.Folder += string(filepath.Separator)
		}
	}

	consoleLogger = options.ConsoleLogger

	if options.ServerWatchdog != nil {
		shutdownProcessRegister = make(chan struct{})

		swc, err = swcgo.Create(*options.ServerWatchdog)
		if err != nil {
			Finalize()
			return  err
		}

		go func() {
			lastRegistrationSucceeded := registerAppInServerWatchdog(true)

			loop := true
			for loop {
				select {
				case <-shutdownProcessRegister:
					loop = false

				case <-time.After(1 * time.Minute):
					lastRegistrationSucceeded = registerAppInServerWatchdog(lastRegistrationSucceeded)
				}
			}
		}()
	}

	cleanOldFiles()

	return nil
}

func Finalize() {
	if swc != nil {
		close(shutdownProcessRegister)

		_ = swc.ProcessUnwatch(os.Getpid(), "")

		swc = nil
	}

	fileMutex.Lock()
	if logFd != nil {
		_ = logFd.Sync()
		_ = logFd.Close()
		logFd = nil
	}
	fileMutex.Unlock()

	consoleLogger = nil
}

func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := time.Now().UTC()

	err := writeLog("[ERROR]", msg, color.Error, now)
	if err != nil {
		printError(now, "Unable to save notification in file", err)
	}

	if swc != nil {
		err = swc.Error(msg, "")
		if err != nil {
			printError(now, "Unable to deliver notification to Server Watchdog", err)
		}
	}
	return
}

func Warn(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := getCurrentTime()

	err := writeLog("[WARN]", msg, color.Warn, now)
	if err != nil {
		printError(now, "Unable to save notification in file", err)
	}

	if swc != nil {
		err = swc.Warn(msg, "")
		if err != nil {
			printError(now, "Unable to deliver notification to Server Watchdog", err)
		}
	}
	return
}

func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := getCurrentTime()

	err := writeLog("[INFO]", msg, color.Info, now)
	if err != nil {
		printError(now, "Unable to save notification in file", err)
	}

	if swc != nil {
		err = swc.Info(msg, "")
		if err != nil {
			printError(now, "Unable to deliver notification to Server Watchdog", err)
		}
	}
	return
}

func Debug(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	now := getCurrentTime()

	err := writeLog("[DEBUG]", msg, color.Debug, now)
	if err != nil {
		printError(now, "Unable to save notification in file", err)
	}
}

//------------------------------------------------------------------------------

func writeLog(title string, msg string, theme *color.Theme, now time.Time) error {
	var err error

	if consoleLogger != nil {
		switch theme {
		case color.Error:
			(*consoleLogger).Error(now, msg)
		case color.Warn:
			(*consoleLogger).Warn(now, msg)
		case color.Info:
			(*consoleLogger).Info(now, msg)
		case color.Debug:
			(*consoleLogger).Debug(now, msg)
		}
	} else {
		color.Print(now.Format("2006-01-02 15:04:05") + " ")
		theme.Print(title)
		color.Print(" - " + msg + "\n")
	}

	err = nil

	if fileOptions != nil {
		fileMutex.Lock()
		if logFd == nil || now.Day() != dayOfFile {
			var filename string

			if logFd != nil {
				_ = logFd.Sync()
				_ = logFd.Close()
				logFd = nil
			}

			cleanOldFiles()

			_ = os.MkdirAll(fileOptions.Folder, 0755)

			filename = fileOptions.Folder + strings.ToLower(appName) + "." + now.Format("2006-01-02") + ".log"

			logFd, err = os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
			if err == nil {
				dayOfFile = now.Day()
			}
		}
		if err == nil {
			_, err = logFd.WriteString("[" + now.Format("2006-01-02 15:04:05") + "] " + title + " - " + msg + newLine)
		}
		fileMutex.Unlock()
	}

	return err
}

func registerAppInServerWatchdog(logError bool) bool {
	err := swc.ProcessWatch(os.Getpid(), appName, "error", "")
	if err == nil {
		return true
	}
	if logError {
		now := getCurrentTime()
		err := writeLog("[WARN]", "Unable to register process in ServerWatchdog", color.Warn, now)
		if err != nil {
			printError(now, "Unable to save notification in file", err)
		}
	}
	return false
}

func cleanOldFiles() {
	if fileOptions == nil || fileOptions.DaysToKeep == 0 {
		return
	}

	lowestTime := time.Now().UTC().AddDate(0, 0, -(int(fileOptions.DaysToKeep)))

	files, err := ioutil.ReadDir(fileOptions.Folder)
	if err == nil {
		appNameLC := strings.ToLower(appName)

		for _, f := range files {
			var nameLC = strings.ToLower(f.Name())

			if (!f.IsDir()) && strings.HasSuffix(nameLC, ".log") && strings.HasPrefix(nameLC, appNameLC) {
				if f.ModTime().Before(lowestTime) {
					_ = os.Remove(fileOptions.Folder + f.Name())
				}
			}
		}
	}
	return
}

func getCurrentTime() time.Time {
	now := time.Now()
	if fileOptions != nil && fileOptions.UseLocalTime {
		now = now.UTC()
	}
	return now
}

func printError(now time.Time, msg string, err error) {
	if consoleLogger != nil {
		(*consoleLogger).Error(now, fmt.Sprintf("%v. [%v]", msg, err.Error()))
	} else {
		color.Print(now.Format("2006-01-02 15:04:05") + " ")
		color.Error.Print("[ERROR]")
		color.Print(" - %v. [%v]\n", msg, err.Error())
	}
	return
}
