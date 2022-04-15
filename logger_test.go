package go_logger_test

import (
	"testing"

	gologger "github.com/randlabs/go-logger"
)

//------------------------------------------------------------------------------

func TestDefault(t *testing.T) {
	printTestMessages(gologger.Default())
}

func TestFileLog(t *testing.T) {
	logger, err := gologger.Create(gologger.Options{
		DisableConsole: true,
		FileLog: &gologger.FileOptions{
			Directory:  "./testdata/logs",
			DaysToKeep: 7,
		},
		Level:        gologger.LogLevelDebug,
		DebugLevel:   1,
		UseLocalTime: false,
	})
	if err != nil {
		t.Errorf("unable to initialize. [%v]", err)
		return
	}

	printTestMessages(logger)

	logger.Destroy()
}

func TestSysLogUDP(t *testing.T) {
	logger, err := gologger.Create(gologger.Options{
		DisableConsole: true,
		SysLog: &gologger.SyslogOptions{
			Host: "127.0.0.1",
			Port: 514,
		},
		Level:        gologger.LogLevelDebug,
		DebugLevel:   1,
		UseLocalTime: false,
	})
	if err != nil {
		t.Errorf("unable to initialize. [%v]", err)
		return
	}

	printTestMessages(logger)

	logger.Destroy()
}

func TestSysLogTCP(t *testing.T) {
	logger, err := gologger.Create(gologger.Options{
		DisableConsole: true,
		SysLog: &gologger.SyslogOptions{
			Host:   "127.0.0.1",
			Port:   1468,
			UseTcp: true,
		},
		Level:        gologger.LogLevelDebug,
		DebugLevel:   1,
		UseLocalTime: false,
	})
	if err != nil {
		t.Errorf("unable to initialize. [%v]", err)
		return
	}

	printTestMessages(logger)

	logger.Destroy()
}

//------------------------------------------------------------------------------
// Private methods

type JsonMessage struct {
	Message string `json:"message"`
}

func printTestMessages(logger *gologger.Logger) {
	logger.Error("This is an error message sample")
	logger.Warning("This is a warning message sample")
	logger.Info("This is an information message sample")
	logger.Debug(1, "This is a debug message sample at level 1 which should be printed")
	logger.Debug(2, "This is a debug message sample at level 2 which should NOT be printed")

	logger.Error(JsonMessage{
		Message: "This is an error message sample",
	})
	logger.Warning(JsonMessage{
		Message: "This is a warning message sample",
	})
	logger.Info(JsonMessage{
		Message: "This is an information message sample",
	})
	logger.Debug(1, JsonMessage{
		Message: "This is a debug message sample at level 1 which should be printed",
	})
	logger.Debug(2, JsonMessage{
		Message: "This is a debug message sample at level 2 which should NOT be printed",
	})
}
