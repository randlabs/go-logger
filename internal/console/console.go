package console

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/gookit/color"
	"github.com/randlabs/go-logger/internal/util"
)

//------------------------------------------------------------------------------

var mtx = sync.Mutex{}

//------------------------------------------------------------------------------

// LogError prints an error message on screen.
func LogError(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logPrint(os.Stderr, color.Error, now, "ERROR", msg)
	} else {
		logPrintJSON(os.Stderr, now, "error", msg)
	}
}

// LogWarning prints a warning message on screen.
func LogWarning(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logPrint(os.Stderr, color.Warn, now, "WARNING", msg)
	} else {
		logPrintJSON(os.Stderr, now, "warning", msg)
	}
}

// LogInfo prints an information message on screen.
func LogInfo(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logPrint(os.Stdout, color.Info, now, "INFO", msg)
	} else {
		logPrintJSON(os.Stdout, now, "info", msg)
	}
}

// LogDebug prints a debug message on screen.
func LogDebug(now time.Time, msg string, isJSON bool) {
	if !isJSON {
		logPrint(os.Stdout, color.Debug, now, "DEBUG", msg)
	} else {
		logPrintJSON(os.Stdout, now, "debug", msg)
	}
}

//------------------------------------------------------------------------------

func logPrint(w io.Writer, theme *color.Theme, now time.Time, level string, msg string) {
	// Lock console access
	mtx.Lock()

	// Print the message prefixed with the timestamp and level
	color.Fprintf(w, "%v %v %v\n", now.Format("2006-01-02 15:04:05.000"), theme.Sprintf("[%v]", level), msg)

	// Unlock console access
	mtx.Unlock()
}

func logPrintJSON(w io.Writer, now time.Time, level string, msg string) {
	// Lock console access
	mtx.Lock()

	// Print the message with extra payload
	color.Fprintf(w, "%v\n", util.AddPayloadToJSON(msg, &now, level))

	// Unlock console access
	mtx.Unlock()
}
