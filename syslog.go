package go_logger

import (
	"container/list"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	rp "github.com/randlabs/rundown-protection"
)

//------------------------------------------------------------------------------

const (
	severityError         = 3
	severityWarning       = 4
	severityInformational = 6
	severityDebug         = 7

	facilityUser = 1

	defaultMaxMessageQueueSize = 1024

	flushTimeout = 5 * time.Second
)

//------------------------------------------------------------------------------

// SysLogOptions specifies the syslog settings to use when it is created.
type SysLogOptions struct {
	// Application name to use. Defaults to the binary name.
	AppName string `json:"appName,omitempty"`

	// Syslog server host name.
	Host string `json:"host,omitempty"`

	// Syslog server port. Defaults to 514, 1468 or 6514 depending on the network protocol used.
	Port uint16 `json:"port,omitempty"`

	// Use TCP instead of UDP.
	UseTcp bool `json:"useTcp,omitempty"`

	// Uses a secure connection. Implies TCP.
	UseTls bool `json:"useTls,omitempty"`

	// Send messages in the new RFC 5424 format instead of the original RFC 3164 specification.
	UseRFC5424 bool `json:"useRFC5424,omitempty"`

	// Set the maximum amount of messages to keep in memory if connection to the server is lost.
	MaxMessageQueueSize uint `json:"queueSize,omitempty"`

	// Set the initial logging level to use.
	Level *LogLevel `json:"level,omitempty"`

	// Set the initial logging level for debug output to use.
	DebugLevel *uint `json:"debugLevel,omitempty"`

	// TLSConfig optionally provides a TLS configuration for use.
	TlsConfig *tls.Config
}

type syslogAdapter struct {
	conn                  net.Conn
	lastWasError          int32
	appName               string
	serverAddress         string
	useTcp                bool
	tlsConfig             *tls.Config
	useRFC5424            bool
	hostname              string
	pid                   int
	queuedMessagesMtx     sync.Mutex
	queuedMessagesList    *list.List
	maxMessageQueueSize   uint
	wakeUpWorkerCh        chan struct{}
	stopWorkerCh          chan struct{}
	rundownProt           *rp.RundownProtection
	globals               globalOptions
}

//------------------------------------------------------------------------------

func createSysLogAdapter(opts SysLogOptions, glbOpts globalOptions) (internalLogger, error) {
	if len(opts.AppName) == 0 {
		var err error

		// If no application name was given, use the base name of the executable.
		opts.AppName, err = os.Executable()
		if err != nil {
			return nil, err
		}
		opts.AppName = filepath.Base(opts.AppName)

		extLen := len(filepath.Ext(opts.AppName))
		if len(opts.AppName) > extLen {
			opts.AppName = opts.AppName[:(len(opts.AppName) - extLen)]
		}
	}

	// Create Syslog adapter
	lg := &syslogAdapter{
		appName:             opts.AppName,
		useTcp:              opts.UseTcp,
		useRFC5424:          opts.UseRFC5424,
		pid:                 os.Getpid(),
		queuedMessagesMtx:   sync.Mutex{},
		queuedMessagesList:  list.New(),
		maxMessageQueueSize: opts.MaxMessageQueueSize,
		wakeUpWorkerCh:      make(chan struct{}, 1),
		stopWorkerCh:        make(chan struct{}),
		rundownProt:         rp.Create(),
		globals:             glbOpts,
	}

	// Set output level based on globals or overrides
	if opts.Level != nil {
		lg.globals.Level = *opts.Level
		lg.globals.DebugLevel = 1
	}
	if opts.DebugLevel != nil {
		lg.globals.DebugLevel = *opts.DebugLevel
	}

	if opts.MaxMessageQueueSize == 0 {
		lg.maxMessageQueueSize = defaultMaxMessageQueueSize
	}

	if opts.UseTls {
		if opts.TlsConfig != nil {
			lg.tlsConfig = opts.TlsConfig.Clone()
		} else {
			lg.tlsConfig = &tls.Config{
				MinVersion: 2,
			}
		}
	}

	// Set the server host
	if len(opts.Host) > 0 {
		lg.serverAddress = opts.Host
	} else {
		lg.serverAddress = "127.0.0.1"
	}

	// Set the server port
	port := opts.Port
	if opts.Port == 0 {
		if opts.UseTcp {
			if opts.UseTls {
				port = 6514
			} else {
				port = 1468
			}
		} else {
			port = 514
		}
	}
	lg.serverAddress += ":" + strconv.Itoa(int(port))

	// Set the client host name
	lg.hostname, _ = os.Hostname()

	// Create a background messenger worker
	go lg.messengerWorker()

	// Done
	return lg, nil
}

func (lg *syslogAdapter) class() string {
	return "syslog"
}

func (lg *syslogAdapter) destroy() {
	// If using TCP, stop worker messenger
	close(lg.stopWorkerCh)

	lg.rundownProt.Wait()

	// Flush queued messages
	lg.flushQueuedMessages()

	// Disconnect from the network
	lg.disconnect()

	// Cleanup
	close(lg.wakeUpWorkerCh)

	lg.wakeUpWorkerCh = nil
	lg.stopWorkerCh = nil
}

func (lg *syslogAdapter) setLevel(level LogLevel, debugLevel uint) {
	lg.globals.Level = level
	lg.globals.DebugLevel = debugLevel
}

func (lg *syslogAdapter) logError(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelError {
		lg.writeString(facilityUser, severityError, now, msg, raw)
	}
}

func (lg *syslogAdapter) logWarning(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelWarning {
		lg.writeString(facilityUser, severityWarning, now, msg, raw)
	}
}

func (lg *syslogAdapter) logInfo(now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelInfo {
		lg.writeString(facilityUser, severityInformational, now, msg, raw)
	}
}

func (lg *syslogAdapter) logDebug(level uint, now time.Time, msg string, raw bool) {
	if lg.globals.Level >= LogLevelDebug && lg.globals.DebugLevel >= level {
		lg.writeString(facilityUser, severityDebug, now, msg, raw)
	}
}

func (lg *syslogAdapter) writeString(facility int, severity int, now time.Time, msg string, _ bool) {
	var formattedMessage string

	// Establish priority
	priority := (facility * 8) + severity

	// Remove or add new line depending on the transport protocol
	if lg.useTcp {
		if !strings.HasSuffix(msg, "\n") {
			msg = msg + "\n"
		}
	} else {
		msg = strings.TrimSuffix(msg, "\n")
	}

	// Format the message
	// NOTE: We don't need to care here about the message type because level and timestamp are in separate fields.
	if !lg.useRFC5424 {
		formattedMessage = "<" + strconv.Itoa(priority) + ">" + now.Format("Jan _2 15:04:05") + " " +
			lg.hostname + " " + msg
	} else {
		formattedMessage = "<" + strconv.Itoa(priority) + ">1 " + now.Format("2006-02-01T15:04:05Z") + " " +
			lg.hostname + " " + lg.appName + " " + strconv.Itoa(lg.pid) + " - - " + msg
	}

	// Queue the message
	lg.queuedMessagesMtx.Lock()

	if uint(lg.queuedMessagesList.Len()) > lg.maxMessageQueueSize {
		elem := lg.queuedMessagesList.Front()
		if elem != nil {
			lg.queuedMessagesList.Remove(elem)
		}
	}
	lg.queuedMessagesList.PushBack(formattedMessage)

	lg.queuedMessagesMtx.Unlock()

	// If the worker is sleeping it will get the wake-up signal, if not the
	// channel will not be being read and the default case will be selected
	select {
	case lg.wakeUpWorkerCh <- struct{}{}:
	default:
	}
}

// The messenger worker do actual message delivery. The intention of this goroutine, is to
// avoid halting the routine that sends the message if there are network issues.
func (lg *syslogAdapter) messengerWorker() {
	for {
		select {
		case <-lg.stopWorkerCh:
			return
		default:
		}

		// Dequeue a message
		lg.queuedMessagesMtx.Lock()
		elem := lg.queuedMessagesList.Front()
		if elem != nil {
			lg.queuedMessagesList.Remove(elem)
		}
		lg.queuedMessagesMtx.Unlock()

		// If no message, check again just in case the signal was buffered
		if elem == nil {
			select {
			case <-lg.stopWorkerCh:
				return
			case <-lg.wakeUpWorkerCh:
			}

			// Dequeue next message
			lg.queuedMessagesMtx.Lock()
			elem = lg.queuedMessagesList.Front()
			if elem != nil {
				lg.queuedMessagesList.Remove(elem)
			}
			lg.queuedMessagesMtx.Unlock()
		}

		// If we have a message, deliver it
		if elem != nil {
			// Try to acquire rundown protection
			if !lg.rundownProt.Acquire() {
				// If we cannot, we are shutting down
				return
			}

			// Send message to server
			err := lg.writeBytes([]byte(elem.Value.(string)))

			// Handle error
			lg.handleError(err)

			// Release rundown protection
			lg.rundownProt.Release()
		}
	}
}

func (lg *syslogAdapter) flushQueuedMessages() {
	deadline := time.Now().Add(flushTimeout)

	for time.Now().Before(deadline) {
		// Dequeue next message
		elem := lg.queuedMessagesList.Front()
		if elem == nil {
			break // Reached the end
		}
		lg.queuedMessagesList.Remove(elem)

		// Send message to server
		err := lg.writeBytes([]byte(elem.Value.(string)))
		if err != nil {
			break // Stop on error
		}
	}
}

func (lg *syslogAdapter) handleError(err error) {
	if err == nil {
		atomic.StoreInt32(&lg.lastWasError, 0)
	} else {
		if atomic.CompareAndSwapInt32(&lg.lastWasError, 0, 1) && lg.globals.ErrorHandler != nil {
			lg.globals.ErrorHandler(fmt.Sprintf("Unable to deliver notification to SysLog [%v]", err))
		}
	}
}

func (lg *syslogAdapter) connect() error {
	var err error

	lg.disconnect()

	if lg.useTcp {
		if lg.tlsConfig != nil {
			lg.conn, err = tls.Dial("tcp", lg.serverAddress, lg.tlsConfig)
		} else {
			lg.conn, err = net.Dial("tcp", lg.serverAddress)
		}
	} else {
		lg.conn, err = net.Dial("udp", lg.serverAddress)
	}

	return err
}

func (lg *syslogAdapter) disconnect() {
	if lg.conn != nil {
		_ = lg.conn.Close()
		lg.conn = nil
	}
}

func (lg *syslogAdapter) writeBytes(b []byte) error {
	var err error

	// Send the message if connected
	if lg.conn != nil {
		_, err = lg.conn.Write(b)
		if err == nil {
			return nil
		}
	}

	// On error or if disconnected, try to connect
	err = lg.connect()
	if err == nil {
		_, err = lg.conn.Write(b)
		if err != nil {
			lg.disconnect()
		}
	}

	// Done
	return err
}
