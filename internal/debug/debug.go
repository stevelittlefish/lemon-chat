package debug

import "log"

var Enabled bool

func Log(format string, args ...any) {
	if Enabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}
