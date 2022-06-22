package syslog

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

// Logger is the object that controls file logging.
type Logger struct {
	conn                  net.Conn
	lastWasError          int32
	appName               string
	serverAddress         string
	useTcp                bool
	tlsConfig             *tls.Config
	useRFC5424            bool
	hostname              string
	pid                   int
	errorHandler          ErrorHandler
	queuedMessagesMtx     sync.Mutex
	queuedMessagesList    *list.List
	maxMessageQueueSize   uint
	wakeUpWorkerCh        chan struct{}
	stopWorkerCh          chan struct{}
	rundownProt           *rp.RundownProtection
}

// Options specifies the sys logger settings to use when it is created.
type Options struct {
	// Application name to use. Defaults to the binary name.
	AppName               string `json:"appName,omitempty"`

	// Syslog server host name.
	Host                  string `json:"host,omitempty"`

	// Syslog server port. Defaults to 514, 1468 or 6514 depending on the network protocol used.
	Port                  uint16 `json:"port,omitempty"`

	// Use TCP instead of UDP.
	UseTcp                bool `json:"useTcp,omitempty"`

	// Uses a secure connection. Implies TCP.
	UseTls                bool `json:"useTls,omitempty"`

	// Send messages in the new RFC 5424 format instead of the original RFC 3164 specification.
	UseRFC5424            bool `json:"useRFC5424,omitempty"`

	// Set the maximum amount of messages to keep in memory if connection to the server is lost.
	MaxMessageQueueSize   uint `json:"queueSize,omitempty"`

	// TLSConfig optionally provides a TLS configuration for use.
	TlsConfig             *tls.Config

	// A callback to call if an internal error is encountered.
	ErrorHandler          ErrorHandler
}

// ErrorHandler is a callback to call if an internal error must be notified.
type ErrorHandler func(message string)

//------------------------------------------------------------------------------

// Create creates a new syslog logger.
func Create(options Options) (*Logger, error) {
	if len(options.AppName) == 0 {
		var err error

		// If no application name was given, use the base name of the executable.
		options.AppName, err = os.Executable()
		if err != nil {
			return nil, err
		}
		options.AppName = filepath.Base(options.AppName)

		extLen := len(filepath.Ext(options.AppName))
		if len(options.AppName) > extLen {
			options.AppName = options.AppName[:(len(options.AppName) - extLen)]
		}
	}

	// Create Syslog logger
	logger := &Logger{
		appName:             options.AppName,
		useTcp:              options.UseTcp,
		useRFC5424:          options.UseRFC5424,
		pid:                 os.Getpid(),
		errorHandler:        options.ErrorHandler,
		queuedMessagesMtx:   sync.Mutex{},
		queuedMessagesList:  list.New(),
		maxMessageQueueSize: options.MaxMessageQueueSize,
		wakeUpWorkerCh:      make(chan struct{}, 1),
		stopWorkerCh:        make(chan struct{}),
		rundownProt:         rp.Create(),
	}

	if options.MaxMessageQueueSize == 0 {
		logger.maxMessageQueueSize = defaultMaxMessageQueueSize
	}

	if options.UseTls {
		if options.TlsConfig != nil {
			logger.tlsConfig = options.TlsConfig.Clone()
		} else {
			logger.tlsConfig = &tls.Config{
				MinVersion: 2,
			}
		}
	}

	// Set the server host
	if len(options.Host) > 0 {
		logger.serverAddress = options.Host
	} else {
		logger.serverAddress = "127.0.0.1"
	}

	// Set the server port
	port := options.Port
	if options.Port == 0 {
		if options.UseTcp {
			if options.UseTls {
				port = 6514
			} else {
				port = 1468
			}
		} else {
			port = 514
		}
	}
	logger.serverAddress += ":" + strconv.Itoa(int(port))

	// Set the client host name
	logger.hostname, _ = os.Hostname()

	// Create a background messenger worker
	go logger.messengerWorker()

	// Done
	return logger, nil
}

// Destroy shuts down the syslog logger.
func (logger *Logger) Destroy() {
	// If using TCP, stop worker messenger
	close(logger.stopWorkerCh)
	logger.rundownProt.Wait()

	// Flush queued messages
	logger.flushQueuedMessages()

	// Disconnect from the network
	logger.disconnect()

	// Cleanup
	close(logger.wakeUpWorkerCh)
}

// LogError sends an error message to the syslog server.
func (logger *Logger) LogError(now time.Time, msg string, isJSON bool) {
	logger.writeString(facilityUser, severityError, now, msg, isJSON)
}

// LogWarning sends a warning message to the syslog server.
func (logger *Logger) LogWarning(now time.Time, msg string, isJSON bool) {
	logger.writeString(facilityUser, severityWarning, now, msg, isJSON)
}

// LogInfo sends an information message to the syslog server.
func (logger *Logger) LogInfo(now time.Time, msg string, isJSON bool) {
	logger.writeString(facilityUser, severityInformational, now, msg, isJSON)
}

// LogDebug sends a debug message to the syslog server.
func (logger *Logger) LogDebug(now time.Time, msg string, isJSON bool) {
	logger.writeString(facilityUser, severityDebug, now, msg, isJSON)
}

//------------------------------------------------------------------------------
// Private methods

func (logger *Logger) writeString(facility int, severity int, now time.Time, msg string, _ bool) {
	var formattedMessage string

	// Establish priority
	priority := (facility * 8) + severity

	// Remove or add new line depending on the transport protocol
	if logger.useTcp {
		if !strings.HasSuffix(msg, "\n") {
			msg = msg + "\n"
		}
	} else {
		msg = strings.TrimSuffix(msg, "\n")
	}

	// Format the message
	// NOTE: We don't need to care here about the message type because level and timestamp are in separate fields.
	if !logger.useRFC5424 {
		formattedMessage = "<" + strconv.Itoa(priority) + ">" + now.Format("Jan _2 15:04:05") + " " +
			logger.hostname + " " + msg
	} else {
		formattedMessage = "<" + strconv.Itoa(priority) + ">1 " + now.Format("2006-02-01T15:04:05Z") + " " +
			logger.hostname + " " + logger.appName + " " + strconv.Itoa(logger.pid) + " - - " + msg
	}

	// Queue the message
	logger.queuedMessagesMtx.Lock()

	if uint(logger.queuedMessagesList.Len()) > logger.maxMessageQueueSize {
		elem := logger.queuedMessagesList.Front()
		if elem != nil {
			logger.queuedMessagesList.Remove(elem)
		}
	}
	logger.queuedMessagesList.PushBack(formattedMessage)

	logger.queuedMessagesMtx.Unlock()

	// If the worker is sleeping it will get the wake-up signal, if not the
	// channel will not be being read and the default case will be selected
	select {
	case logger.wakeUpWorkerCh <- struct{}{}:
	default:
	}
}

// The messenger worker do actual message delivery. The intention of this goroutine, is to
// avoid halting the routine that sends the message if there are network issues.
func (logger *Logger) messengerWorker() {
	for {
		select {
		case <-logger.stopWorkerCh:
			return
		default:
		}

		// Dequeue a message
		logger.queuedMessagesMtx.Lock()
		elem := logger.queuedMessagesList.Front()
		if elem != nil {
			logger.queuedMessagesList.Remove(elem)
		}
		logger.queuedMessagesMtx.Unlock()

		// If no message, check again just in case the signal was buffered
		if elem == nil {
			select {
			case <-logger.stopWorkerCh:
				return
			case <-logger.wakeUpWorkerCh:
			}

			// Dequeue next message
			logger.queuedMessagesMtx.Lock()
			elem = logger.queuedMessagesList.Front()
			if elem != nil {
				logger.queuedMessagesList.Remove(elem)
			}
			logger.queuedMessagesMtx.Unlock()
		}

		// If we have a message, deliver it
		if elem != nil {
			// Try to acquire rundown protection
			if !logger.rundownProt.Acquire() {
				// If we cannot, we are shutting down
				return
			}

			// Send message to server
			err := logger.writeBytes([]byte(elem.Value.(string)))

			// Handle error
			logger.handleError(err)

			// Release rundown protection
			logger.rundownProt.Release()
		}
	}
}

func (logger *Logger) flushQueuedMessages() {
	deadline := time.Now().Add(flushTimeout)

	for time.Now().Before(deadline) {
		// Dequeue next message
		elem := logger.queuedMessagesList.Front()
		if elem == nil {
			break // Reached the end
		}
		logger.queuedMessagesList.Remove(elem)

		// Send message to server
		err := logger.writeBytes([]byte(elem.Value.(string)))
		if err != nil {
			break // Stop on error
		}
	}
}

func (logger *Logger) handleError(err error) {
	if err == nil {
		atomic.StoreInt32(&logger.lastWasError, 0)
	} else {
		if atomic.CompareAndSwapInt32(&logger.lastWasError, 0, 1) && logger.errorHandler != nil {
			logger.errorHandler(fmt.Sprintf("Unable to deliver notification to SysLog [%v]", err))
		}
	}
}

func (logger *Logger) connect() error {
	var err error

	logger.disconnect()

	if logger.useTcp {
		if logger.tlsConfig != nil {
			logger.conn, err = tls.Dial("tcp", logger.serverAddress, logger.tlsConfig)
		} else {
			logger.conn, err = net.Dial("tcp", logger.serverAddress)
		}
	} else {
		logger.conn, err = net.Dial("udp", logger.serverAddress)
	}

	return err
}

func (logger *Logger) disconnect() {
	if logger.conn != nil {
		_ = logger.conn.Close()
		logger.conn = nil
	}
}

func (logger *Logger) writeBytes(b []byte) error {
	var err error

	// Send the message if connected
	if logger.conn != nil {
		_, err = logger.conn.Write(b)
		if err == nil {
			return nil
		}
	}

	// On error or if disconnected, try to connect
	err = logger.connect()
	if err == nil {
		_, err = logger.conn.Write(b)
		if err != nil {
			logger.disconnect()
		}
	}

	// Done
	return err
}
