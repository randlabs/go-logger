package console_logger

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/gookit/color"
)

//------------------------------------------------------------------------------

var mtx sync.Mutex

//------------------------------------------------------------------------------

// Error ...
func Error(now time.Time, msg string) {
	print(os.Stderr, color.Error, now, "[ERROR]", msg)
}

// Warn ...
func Warn(now time.Time, msg string) {
	print(os.Stderr, color.Warn, now, "[WARN]", msg)
}

// Info ...
func Info(now time.Time, msg string) {
	print(os.Stdout, color.Info, now, "[INFO]", msg)
}

// Debug ...
func Debug(now time.Time, msg string) {
	print(os.Stdout, color.Debug, now, "[DEBUG]", msg)
}

//------------------------------------------------------------------------------
// Private methods

func print(w io.Writer, theme *color.Theme, now time.Time, title string, msg string) {
	mtx.Lock()

	color.SetOutput(w)

	color.Printf("%v ", now.Format("2006-01-02 15:04:05"))
	theme.Print(title)
	color.Printf(" %v\n", msg)

	color.ResetOutput()

	mtx.Unlock()
}
