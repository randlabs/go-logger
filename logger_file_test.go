package go_logger_test

import (
	"os"
	"path/filepath"
	"testing"

	logger "github.com/randlabs/go-logger/v2"
)

//------------------------------------------------------------------------------

func TestFileLog(t *testing.T) {
	if dir, err := filepath.Abs(filepath.FromSlash("./testdata/logs")); err == nil {
		_ = os.RemoveAll(dir)
	}

	lg, err := logger.Create(logger.Options{
		Console: logger.ConsoleOptions{
			Disable: true,
		},
		File: &logger.FileOptions{
			Prefix:     "Test",
			Directory:  "./testdata/logs",
			DaysToKeep: 7,
		},
		Level:        logger.LogLevelDebug,
		DebugLevel:   1,
		UseLocalTime: false,
	})
	if err != nil {
		t.Errorf("unable to initialize. [%v]", err)
		return
	}
	defer lg.Destroy()

	printTestMessages(lg)
}
