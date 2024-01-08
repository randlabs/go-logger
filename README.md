# go-logger

Yet another Go logger library.

### Version 2

The new version of this library implements some breaking changes:

* The initialization options contains some changes. Once the logger is initialized, the rest of the code should be
  compatible with the `v1` version.
* Added per-target logging level. Now you can optionally specify a different logging level for console and file logging.
  For example, you can just send limited logs to a third-party provider to avoid high costs and keep a detailed local
  file log for debugging purposes. 
* `timestamp` and `level` fields are always added to a JSON log. DO NOT include them when the input is a struct.
* Verbosity level for the default logger is now `Info` instead of `Warning`.

## How to use

1. Import the library

```golang
import (
    logger "github.com/randlabs/go-logger/v2"
)
```

2. Then use `logger.Create` to create a logger object with desired options.
3. Optionally, you can also use the default logger which outputs only to console by accessing `logger.Default()`.

## Logger options:

The `Options` struct accepts several modifiers that affects the logger behavior:

| Field          | Meaning                                                 |
|----------------|---------------------------------------------------------|
| `Console`      | Establishes some options for the console output.        |
| `File`         | Enable file logging. Optional. Details below.           |
| `SysLog`       | Enable SysLog logging. Optional. Details below.         |
| `Level`        | Set the initial logging level to use.                   |
| `DebugLevel`   | Set the initial logging level for debug output to use.  |
| `UseLocalTime` | Use the local computer time instead of UTC.             |
| `ErrorHandler` | A callback to call if an internal error is encountered. |

#### ConsoleOptions:

| Field        | Meaning                                                               |
|--------------|-----------------------------------------------------------------------|
| `Disable`    | Disabled console output.                                              |
| `Level`      | Optional logging level to use in the console output.                  |
| `DebugLevel` | Optional logging level for debug output to use in the console output. |

#### FileOptions:

| Field        | Meaning                                                                     |
|--------------|-----------------------------------------------------------------------------|
| `Prefix`     | Filename prefix to use when a file is created. Defaults to the binary name. |
| `Directory`  | Destination directory to store log files.                                   |
| `DaysToKeep` | Amount of days to keep old logs.                                            |
| `Level`      | Optional logging level to use in the file output.                           |
| `DebugLevel` | Optional logging level for debug output to use in the file output.          |

#### SysLogOptions:

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
| `Level`               | Optional logging level to use in the syslog output.                                       |
| `DebugLevel`          | Optional logging level for debug output to use in the syslog output.                      |

## Example

```golang
package example

import (
    "fmt"

    logger "github.com/randlabs/go-logger/v2"
)

// Define a custom JSON message. Timestamp and level will be automatically added by the logger.
type JsonMessage struct {
    Message string `json:"message"`
}

func main() {
    // Create the logger
    lg, err := logger.Create(logger.Options{
        File: &logger.FileOptions{
            Directory:  "./logs",
            DaysToKeep: 7,
        },
        Level:        logger.LogLevelDebug,
        DebugLevel:   1,
    })
    if err != nil {
        // Use default logger to send the error
        logger.Default().Error(fmt.Sprintf("unable to initialize. [%v]", err))
        return
    }
    // Defer logger shut down
    defer lg.Destroy()

    // Send some logs using the plain text format 
    lg.Error("This is an error message sample")
    lg.Warning("This is a warning message sample")
    lg.Info("This is an information message sample")
    lg.Debug(1, "This is a debug message sample at level 1 which should be printed")
    lg.Debug(2, "This is a debug message sample at level 2 which should NOT be printed")

    // Send some other logs using the JSON format 
    lg.Error(JsonMessage{
        Message: "This is an error message sample",
    })
    lg.Warning(JsonMessage{
        Message: "This is a warning message sample",
    })
    lg.Info(JsonMessage{
        Message: "This is an information message sample",
    })
    lg.Debug(1, JsonMessage{
        Message: "This is a debug message sample at level 1 which should be printed",
    })
    lg.Debug(2, JsonMessage{
        Message: "This is a debug message sample at level 2 which should NOT be printed",
    })
}
```

## License

See [LICENSE](/LICENSE) file for details.
