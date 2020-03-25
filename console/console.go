package console

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gookit/color"
)

//------------------------------------------------------------------------------

const (
	classError int = 0
	classWarn      = 1
	classInfo      = 2
	classDebug     = 3
)

//------------------------------------------------------------------------------

var m sync.Mutex

//------------------------------------------------------------------------------

// Error ...
func Error(format string, a ...interface{}) {
	printCommon("", classError, getTimestamp(), fmt.Sprintf(format, a...))
	return
}

// Warn ...
func Warn(format string, a ...interface{}) {
	printCommon("", classWarn, getTimestamp(), fmt.Sprintf(format, a...))
	return
}

// Info ...
func Info(format string, a ...interface{}) {
	printCommon("", classInfo, getTimestamp(), fmt.Sprintf(format, a...))
	return
}

// Debug ...
func Debug(format string, a ...interface{}) {
	printCommon("", classDebug, getTimestamp(), fmt.Sprintf(format, a...))
	return
}

// LogError ...
func LogError(title string, timestamp string, msg string) {
	printCommon(title, classError, timestamp, msg)
	return
}

// LogWarn ...
func LogWarn(title string, timestamp string, msg string) {
	printCommon(title, classWarn, timestamp, msg)
	return
}

// LogInfo ...
func LogInfo(title string, timestamp string, msg string) {
	printCommon(title, classInfo, timestamp, msg)
	return
}

// LogDebug ...
func LogDebug(title string, timestamp string, msg string) {
	printCommon(title, classDebug, timestamp, msg)
	return
}

//------------------------------------------------------------------------------

func printCommon(title string, cls int, timestamp string, msg string) {
	m.Lock()
	defer m.Unlock()

	if cls == classInfo || cls == classDebug {
		color.SetOutput(os.Stdout)
	} else {
		color.SetOutput(os.Stderr)
	}

	color.Printf("%v ", timestamp)

	switch cls {
	case classError:
		color.Error.Print("[ERROR]")
	case classWarn:
		color.Warn.Print("[WARN]")
	case classInfo:
		color.Info.Print("[INFO]")
	case classDebug:
		color.Debug.Print("[DEBUG]")
	}

	if len(title) > 0 {
		color.Printf(" %v", title)
	}

	color.Printf(" - %v\n", msg)

	color.ResetOutput()
	return
}

func getTimestamp() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}
