package async

import (
	"fmt"
	"os"
	"runtime/debug"
)

// RecoverPanic logs an unexpected panic from background work so one task does
// not terminate the whole process. It returns true when a panic was recovered.
func RecoverPanic(context string) bool {
	if r := recover(); r != nil {
		if context == "" {
			context = "async task"
		}
		fmt.Fprintf(os.Stderr, "recovered panic in %s: %v\n%s", context, r, debug.Stack())
		return true
	}
	return false
}
