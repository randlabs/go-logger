# go-logger

Yet another Go logger library.

## How to use

1. Import the library

```golang
import (
        gologger "github.com/randlabs/go-logger"
)
```

2. Then use `gologger.Create` to create a logger object with desired options.
3. Optionally, you can also use the default logger which outputs only to console by accesing `gologger.Default()`.

## Logger options:

The `Options` struct accepts several modifiers that affects the logger behavior:

| Field            | Meaning                                                 |
|------------------|---------------------------------------------------------|
| `DisableConsole` | Disable console output.                                 |
| `FileLog`        | Enable file logging. Optional. Details below.           |
| `SysLog`         | Enable SysLog logging. Optional. Details below.         |
| `Level`          | Set the initial logging level to use.                   |
| `DebugLevel`     | Set the initial logging level for debug output to use.  |
| `UseLocalTime`   | Use the local computer time instead of UTC.             |
| `ErrorHandler`   | A callback to call if an internal error is encountered. |

The `FileOptions` struct accepts the following parameters:

| Field        | Meaning                                                                     |
|--------------|-----------------------------------------------------------------------------|
| `Prefix`     | Filename prefix to use when a file is created. Defaults to the binary name. |
| `Directory`  | Destination directory to store log files.                                   |
| `DaysToKeep` | Amount of days to keep old logs,                                            |

And the `SysLogOptions` struct accepts the following parameters:


| Field                 | Meaning                                                                                   |
|-----------------------|-------------------------------------------------------------------------------------------|
| `AppName`             | Application name to use. Defaults to the binary name.                                     |
| `Host`                | Syslog server host name.                                                                  |
| `Port`                | Syslog server port. Defaults to 514, 1468 or 6514 depending on the network protocol used. |
| `UseTcp`              | Use TCP instead of UDP.                                                                   |
| `UseTls`              | Uses a secure connection. Implies TCP.                                                    |
| `UseRFC5424`          | Send messages in the new RFC 5424 format instead of the original RFC 3164 specification.  |
| `MaxMessageQueueSize` | Set the maximum amount of messages to keep in memory if connection to the server is lost. |
| `TlsConfig`           | An optional pointer to a `tls.Config` object to provide the TLS configuration for use.    |

## Example

```golang
package example

import (
    "fmt"
    "os"
    "path/filepath"

    gologger "github.com/randlabs/go-logger"
)

// Define a custom JSON message. Timestamp and level will be automatically added by the logger.
type JsonMessage struct {
    Message string `json:"message"`
}

func main() {
    // Create the logger
    logger, err := gologger.Create(gologger.Options{
        FileLog: &gologger.FileOptions{
            Directory:  "./logs",
            DaysToKeep: 7,
        },
        Level:        gologger.LogLevelDebug,
        DebugLevel:   1,
    })
    if err != nil {
        // Use default logger to send the error
        gologger.Default().Error(fmt.Sprintf("unable to initialize. [%v]", err))
        return
    }
    // Defer logger shut down
    defer logger.Destroy()

    // Send some logs using the plain text format 
    logger.Error("This is an error message sample")
    logger.Warning("This is a warning message sample")
    logger.Info("This is an information message sample")
    logger.Debug(1, "This is a debug message sample at level 1 which should be printed")
    logger.Debug(2, "This is a debug message sample at level 2 which should NOT be printed")

    // Send some other logs using the JSON format 
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
```

## License
See `LICENSE` file for details.
