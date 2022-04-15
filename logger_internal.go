package go_logger

import (
	"encoding/json"
	"reflect"
	"time"
)

//------------------------------------------------------------------------------

// NOTE: This function is only called if the logger has an error handler callback.
func (logger *Logger) forwardLogError(message string) {
	logger.errorHandler(message)
}

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
