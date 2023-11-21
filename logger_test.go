package go_logger_test

import (
	"testing"

	logger "github.com/randlabs/go-logger/v2"
)

//------------------------------------------------------------------------------

func TestDefault(t *testing.T) {
	printTestMessages(logger.Default())
}

func TestLevelOverride(t *testing.T) {
	lg, err := logger.Create(logger.Options{
		Console: logger.ConsoleOptions{
			Level: logger.WithLevel(logger.LogLevelError),
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

//------------------------------------------------------------------------------
// Private methods

type JsonMessage struct {
	Message string `json:"message"`
}

func printTestMessages(l *logger.Logger) {
	l.Error("This is an error message sample")
	l.Warning("This is a warning message sample")
	l.Info("This is an information message sample")
	l.Debug(1, "This is a debug message sample at level 1 which should be printed")
	l.Debug(2, "This is a debug message sample at level 2 which should NOT be printed")

	l.Error(JsonMessage{
		Message: "This is an error message sample",
	})
	l.Warning(JsonMessage{
		Message: "This is a warning message sample",
	})
	l.Info(JsonMessage{
		Message: "This is an information message sample",
	})
	l.Debug(1, JsonMessage{
		Message: "This is a debug message sample at level 1 which should be printed",
	})
	l.Debug(2, JsonMessage{
		Message: "This is a debug message sample at level 2 which should NOT be printed",
	})
}
