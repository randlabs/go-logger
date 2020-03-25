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
var logOptions FileOptions
var fileMutex sync.Mutex
var logFd *os.File = nil
var dayOfFile int
var swc *swcgo.ServerWatcherClient

//------------------------------------------------------------------------------

type Options struct {
	FileOpts *FileOptions               `json:"file,omitempty"`
	ServerWatchdog *swcgo.ClientOptions `json:"serverWatchdog,omitempty"`
}

type FileOptions struct {
	DaysToKeep uint   `json:"daysToKeep,omitempty"`
	Folder     string `json:"folder,omitempty"`
	UseLocalTime bool `json:"useLocalTime,omitempty"`
}

//------------------------------------------------------------------------------

// Initialize...
func Initialize(_appName string, options Options, baseFolder string) error {
	var err error

	if len(_appName) == 0 {
		return errors.New("invalid application name")
	}
	appName = _appName

	logOptions.Folder = "logs"
	if options.FileOpts != nil {
		if len(options.FileOpts.Folder) > 0 {
			logOptions.Folder = options.FileOpts.Folder
		}

		logOptions.DaysToKeep = options.FileOpts.DaysToKeep

		logOptions.UseLocalTime = options.FileOpts.UseLocalTime
	}

	if !filepath.IsAbs(logOptions.Folder) {
		logOptions.Folder = filepath.Join(baseFolder, logOptions.Folder)
	}
	logOptions.Folder = filepath.Clean(logOptions.Folder)
	if !strings.HasSuffix(logOptions.Folder, string(filepath.Separator)) {
		logOptions.Folder += string(filepath.Separator)
	}

	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}

	if options.ServerWatchdog != nil {
		swc, err = swcgo.Create(*options.ServerWatchdog)
		if err != nil {
			return  err
		}

		err = swc.ProcessWatch(os.Getpid(), appName, "error", "")
		if err != nil {
			return  err
		}
	}

	cleanOldFiles()

	return nil
}

func Finalize() {
	if swc != nil {
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

	color.Print(now.Format("2006-01-02 15:04:05") + " ")
	theme.Print(title)
	color.Print(" - "+ msg + "\n")

	err = nil

	fileMutex.Lock()
	if logFd == nil || now.Day() != dayOfFile {
		var filename string

		if logFd != nil {
			_ = logFd.Sync()
			_ = logFd.Close()
			logFd = nil
		}

		cleanOldFiles()

		_ = os.MkdirAll(logOptions.Folder, 0755)

		filename = logOptions.Folder + strings.ToLower(appName) + "." + now.Format("2006-01-02") + ".log"

		logFd, err = os.OpenFile(filename, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0644)
		if err == nil {
			dayOfFile = now.Day()
		}
	}
	if err == nil {
		_, err = logFd.WriteString("[" + now.Format("2006-01-02 15:04:05") + "] " + title + " - " + msg + newLine)
	}
	defer fileMutex.Unlock()

	return err
}

func cleanOldFiles() {
	var files []os.FileInfo
	var lowestTime time.Time
	var err error

	if logOptions.DaysToKeep == 0 {
		return
	}

	lowestTime = time.Now().UTC().AddDate(0, 0, -(int(logOptions.DaysToKeep)))

	files, err = ioutil.ReadDir(logOptions.Folder)
	if err == nil {
		appNameLC := strings.ToLower(appName)

		for _, f := range files {
			var nameLC = strings.ToLower(f.Name())

			if (!f.IsDir()) && strings.HasSuffix(nameLC, ".log") && strings.HasPrefix(nameLC, appNameLC) {
				if f.ModTime().Before(lowestTime) {
					_ = os.Remove(logOptions.Folder + f.Name())
				}
			}
		}
	}
	return
}

func getCurrentTime() time.Time {
	now := time.Now()
	if logOptions.UseLocalTime {
		now = now.UTC()
	}
	return now
}

func printError(now time.Time, msg string, err error) {
	color.Print(now.Format("2006-01-02 15:04:05") + " ")
	color.Error.Print("[ERROR]")
	color.Print(" - %v. [%v]\n", msg, err)
	return
}
