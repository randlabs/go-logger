package go_logger

import (
	"encoding/json"
	"reflect"
	"time"
)

//------------------------------------------------------------------------------

type globalOptions struct {
	// Set the initial logging level to use.
	Level LogLevel

	// Set the initial logging level for debug output to use.
	DebugLevel uint

	// A callback to call if an internal error is encountered.
	ErrorHandler ErrorHandler
}

//------------------------------------------------------------------------------

func (logger *Logger) getTimestamp() time.Time {
	now := time.Now()
	if !logger.useLocalTime {
		now = now.UTC()
	}
	return now
}

func (logger *Logger) parseObj(obj interface{}) (msg string, isJSON bool, ok bool) {
	// Quick check for strings, structs or pointer to strings or structs
	refObj := reflect.ValueOf(obj)
	switch refObj.Kind() {
	case reflect.Ptr:
		if !refObj.IsNil() {
			switch refObj.Elem().Kind() {
			case reflect.String:
				msg = *(obj.(*string))
				ok = true

			case reflect.Struct:
				// Marshal struct
				b, err := json.Marshal(obj)
				if err == nil {
					msg = string(b)
					isJSON = true
					ok = true
				}
			}
		}

	case reflect.String:
		msg = obj.(string)
		ok = true

	case reflect.Struct:
		// Marshal struct
		b, err := json.Marshal(obj)
		if err == nil {
			msg = string(b)
			isJSON = true
			ok = true
		}
	}

	// Done
	return
}
