package syslog_logger

//goland:noinspection ALL
import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	consolelogger "github.com/randlabs/go-logger/console"
)

//------------------------------------------------------------------------------

const (
	severityError = 3
	severityWarning = 4
	severityInformational = 6
	// severityDebug = 7

	facilityUser = 1
)

//------------------------------------------------------------------------------

// SysLogOptions ...
type SysLogOptions struct {
	Host                   string `json:"host,omitempty"`
	Port                   uint16 `json:"port,omitempty"`
	UseTcp                 bool `json:"useTcp,omitempty"`
	UseTls                 bool `json:"useTls,omitempty"`
	UseRFC3164             bool `json:"useRFC3164,omitempty"`
	SendInfoNotifications  bool `json:"sendInfoNotifications,omitempty"`
}

// SysLogLogger ...
type SysLogLogger struct {
	mtx                   sync.Mutex
	conn                  net.Conn
	lastWasError          int32
	appName               string
	host                  string
	port                  uint16
	useTcp                bool
	useTls                bool
	useRFC3164            bool
	sendInfoNotifications bool
	hostname              string
	pid                   int
}

//------------------------------------------------------------------------------

func CreateSysLogLogger(appName string, options *SysLogOptions) (*SysLogLogger, error) {
	syslogLogger := &SysLogLogger{}
	syslogLogger.appName = appName

	if len(options.Host) > 0 {
		syslogLogger.host = options.Host
	} else {
		syslogLogger.host = "127.0.0.1"
	}

	if options.Port != 0 {
		syslogLogger.port = options.Port
	} else {
		if options.UseTcp {
			if options.UseTls {
				syslogLogger.port = 6514
			} else {
				syslogLogger.port = 1468
			}
		} else {
			syslogLogger.port = 514
		}
	}

	syslogLogger.useTcp = options.UseTcp
	syslogLogger.useTls = options.UseTls
	syslogLogger.useRFC3164 = options.UseRFC3164
	syslogLogger.sendInfoNotifications = options.SendInfoNotifications

	syslogLogger.hostname, _ = os.Hostname()
	syslogLogger.pid = os.Getpid()

	//done
	return syslogLogger, nil
}

func (logger *SysLogLogger) Shutdown() {
	logger.mtx.Lock()

	logger.disconnect()

	logger.mtx.Unlock()
}

// Error ...
func (logger *SysLogLogger) Error(now time.Time, msg string) {
	logger.writeString(facilityUser, severityError, now, msg)
}

// Warn ...
func (logger *SysLogLogger) Warn(now time.Time, msg string) {
	logger.writeString(facilityUser, severityWarning, now, msg)
}

// Info ...
func (logger *SysLogLogger) Info(now time.Time, msg string) {
	if logger.sendInfoNotifications {
		logger.writeString(facilityUser, severityInformational, now, msg)
	}
}

//------------------------------------------------------------------------------
// Private methods

func (logger *SysLogLogger) writeString(facility int, severity int, now time.Time, msg string) {
	var formattedMessage string

	priority := (facility * 8) + severity

	if logger.useTcp {
		if !strings.HasSuffix(msg, "\n") {
			msg = msg + "\n"
		}
	} else {
		if strings.HasSuffix(msg, "\n") {
			msg = msg[:len(msg)-1]
		}
	}

	if logger.useRFC3164 {
		formattedMessage = "<" + strconv.Itoa(priority) + ">" + now.Format("Jan _2 15:04:05") + " " +
			logger.hostname + " " + msg
	} else {
		formattedMessage = "<" + strconv.Itoa(priority) + ">1 " + now.Format("2006-02-01T15:04:05Z") + " " +
			logger.hostname + " " + logger.appName + " " + strconv.Itoa(logger.pid) + " - - " + msg
	}

	err := logger.writeBytes([]byte(formattedMessage))

	if err == nil {
		atomic.StoreInt32(&logger.lastWasError, 0)
	} else {
		if atomic.CompareAndSwapInt32(&logger.lastWasError, 0, 1) {
			consolelogger.Error(now, fmt.Sprintf("Unable to deliver notification to SysLog [%v]", err))
		}
	}
}

func (logger *SysLogLogger) connect() error {
	var err error

	logger.disconnect()

	address := logger.host + ":" + strconv.Itoa(int(logger.port))
	if logger.useTcp {
		if logger.useTls {
			tlsConfig := &tls.Config{
				//RootCAs: roots,
			}
			logger.conn, err = tls.Dial("tcp", address, tlsConfig)
		} else {
			logger.conn, err = net.Dial("tcp", address)
		}
	} else {
		logger.conn, err = net.Dial("udp", address)
	}

	return err
}

func (logger *SysLogLogger) disconnect() {
	if logger.conn != nil {
		_ = logger.conn.Close()
		logger.conn = nil
	}
}

func (logger *SysLogLogger) writeBytes(b []byte) error {
	var err error

	logger.mtx.Lock()

	if logger.conn != nil {
		_, err := logger.conn.Write(b)
		if err == nil {
			logger.mtx.Unlock()
			return nil
		}
	}

	err = logger.connect()
	if err == nil {
		_, err := logger.conn.Write(b)
		if err != nil {
			logger.disconnect()
		}
	}

	logger.mtx.Unlock()

	return err
}
