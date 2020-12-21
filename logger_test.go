package go_logger

import (
	"testing"

	filelogger "github.com/randlabs/go-logger/file_logger"
	sysloglogger "github.com/randlabs/go-logger/syslog_logger"
)

//------------------------------------------------------------------------------

func TestFileLog(t *testing.T) {
	options := Options{
		AppName: "logger_test",
		FileLog: &filelogger.FileLogOptions{
			Directory:  "./logs",
			DaysToKeep: 7,
		},
		DebugLevel: 1,
		UseLocalTime: false,
	}

	err := Initialize(options)
	if err != nil {
		t.Errorf("unable to initialize. [%v]", err)
		return
	}

	printTestMessages()

	Finalize()
}

func TestSysLog(t *testing.T) {
	options := Options{
		AppName: "logger_test",
		SysLog: &sysloglogger.SysLogOptions{
			Host:                  "localhost",
			Port:                  514,
			UseTcp:                true,
			UseRFC3164:            false,
			SendInfoNotifications: false,
		},
		DebugLevel: 1,
		UseLocalTime: false,
	}

	err := Initialize(options)
	if err != nil {
		t.Errorf("unable to initialize. [%v]", err)
		return
	}

	printTestMessages()

	Finalize()
}

//------------------------------------------------------------------------------
// Private methods

func printTestMessages() {
	Error("This is an error message sample")
	Warn("This is a warning message sample")
	Info("This is an information message sample")
	Debug(1, "This is a debug message sample at level 1 which should be printed")
	Debug(2, "This is a debug message sample at level 2 which should NOT be printed")
}
