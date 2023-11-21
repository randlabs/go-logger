package go_logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gookit/color"
)

//------------------------------------------------------------------------------

// ConsoleOptions specifies the console logger settings to use when it is created.
type ConsoleOptions struct {
	// Disable console output.
	Disable bool `json:"disable,omitempty"`

	// Set the initial logging level to use.
	Level *LogLevel `json:"level,omitempty"`

	// Set the initial logging level for debug output to use.
	DebugLevel *uint `json:"debugLevel,omitempty"`
}

type consoleAdapter struct {
	themedLevels [4]string
	globals      globalOptions
}

//------------------------------------------------------------------------------

var consoleMtx = sync.Mutex{}

//------------------------------------------------------------------------------

func createConsoleAdapter(opts ConsoleOptions, glbOpts globalOptions) internalLogger {
	// Create console adapter
	lg := &consoleAdapter{
		globals: glbOpts,
	}

	if color.IsSupportColor() {
		lg.themedLevels[0] = color.New(color.OpBlink, color.FgLightWhite, color.BgRed).Sprintf("[ERROR]")
		lg.themedLevels[1] = color.New(color.FgLightYellow).Sprintf("[WARN]")
		lg.themedLevels[2] = color.New(color.FgLightGreen).Sprintf("[INFO]")
		lg.themedLevels[3] = color.New(color.FgCyan).Sprintf("[DEBUG]")
	} else {
		lg.themedLevels[0] = "[ERROR]"
		lg.themedLevels[1] = "[WARN]"
		lg.themedLevels[2] = "[INFO]"
		lg.themedLevels[3] = "[DEBUG]"
	}

	// Set output level based on globals or overrides
	if opts.Level != nil {
		lg.globals.Level = *opts.Level
		lg.globals.DebugLevel = 1
	}
	if opts.DebugLevel != nil {
		lg.globals.DebugLevel = *opts.DebugLevel
	}

	// Done
	return lg
}

func (lg *consoleAdapter) class() string {
	return "console"
}

func (lg *consoleAdapter) destroy() {
	// Do nothing
}

func (lg *consoleAdapter) setLevel(level LogLevel, debugLevel uint) {
	lg.globals.Level = level
	lg.globals.DebugLevel = debugLevel
}

func (lg *consoleAdapter) logError(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelError {
		if !raw {
			consolePrint(os.Stderr, now, lg.themedLevels[0], msg)
		} else {
			consolePrintRAW(os.Stderr, msg)
		}
	}
}

func (lg *consoleAdapter) logWarning(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelWarning {
		if !raw {
			consolePrint(os.Stderr, now, lg.themedLevels[1], msg)
		} else {
			consolePrintRAW(os.Stderr, msg)
		}
	}
}

func (lg *consoleAdapter) logInfo(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelInfo {
		if !raw {
			consolePrint(os.Stdout, now, lg.themedLevels[2], msg)
		} else {
			consolePrintRAW(os.Stdout, msg)
		}
	}
}

func (lg *consoleAdapter) logDebug(level uint, now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelDebug && lg.globals.DebugLevel >= level {
		if !raw {
			consolePrint(os.Stdout, now, lg.themedLevels[3], msg)
		} else {
			consolePrintRAW(os.Stdout, msg)
		}
	}
}

func consolePrint(w io.Writer, now time.Time, themedLevel string, msg string) {
	// Lock console access
	consoleMtx.Lock()

	// Print the message prefixed with the timestamp and level
	_, _ = fmt.Fprintf(w, "%v %v %v\n", now.Format("2006-01-02 15:04:05.000"), themedLevel, msg)

	// Unlock console access
	consoleMtx.Unlock()
}

func consolePrintRAW(w io.Writer, msg string) {
	// Lock console access
	consoleMtx.Lock()

	// Print the message with extra payload
	_, _ = fmt.Fprintf(w, "%v\n", msg)

	// Unlock console access
	consoleMtx.Unlock()
}
