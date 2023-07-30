package go_logger

import (
	"fmt"
	"time"
)

//------------------------------------------------------------------------------

func addPayloadToJSON(s string, now time.Time, level string) string {
	payload := fmt.Sprintf(`"timestamp":"%v","level":"%v"`, now.Format("2006-01-02 15:04:05.000"), level)

	// Embed additional payload
	sep := ""
	if len(s) != 2 || s[1] != '}' {
		sep = "," // Add the comma separator if not an empty json object
	}

	// Return modified string
	return s[:1] + payload + sep + s[1:]
}
