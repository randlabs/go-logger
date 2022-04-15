package util

import (
	"strings"
	"time"
)

//------------------------------------------------------------------------------

func AddPayloadToJSON(s string, now *time.Time, level string) string {
	payload := make([]string, 0)
	if now != nil {
		payload = append(payload, `"timestamp":"`+now.Format("2006-01-02 15:04:05.000")+`"`)
	}
	if len(level) > 0 {
		payload = append(payload, `"level":"` + level + `"`)
	}

	if len(payload) == 0 {
		return s
	}

	// Embed additional payload
	sep := ""
	if len(s) != 2 || s[1] != '}' {
		sep = "," // Add the comma separator if not an empty json object
	}

	// Return modified string
	return s[:1] + strings.Join(payload, ",") + sep + s[1:]
}
