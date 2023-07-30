package go_logger

import (
	"time"
)

type internalLogger interface {
	class() string

	destroy()

	//NOTE: Called within an exclusive lock
	setLevel(level LogLevel, debugLevel uint)

	//NOTE: Called within a shared lock
	logError(now time.Time, msg string, raw bool)
	logWarning(now time.Time, msg string, raw bool)
	logInfo(now time.Time, msg string, raw bool)
	logDebug(level uint, now time.Time, msg string, raw bool)
}
