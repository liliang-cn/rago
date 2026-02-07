package log

import (
	"fmt"
	"os"
	"sync"
)

var (
	debugEnabled bool
	mu           sync.RWMutex
)

// SetDebug enables or disables debug logging
func SetDebug(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	debugEnabled = enabled
}

// IsDebug returns whether debug logging is enabled
func IsDebug() bool {
	mu.RLock()
	defer mu.RUnlock()
	return debugEnabled
}

// Debug logs a debug message only if debug mode is enabled
func Debug(format string, args ...interface{}) {
	if IsDebug() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// DebugWithPrefix logs a debug message with a custom prefix
func DebugWithPrefix(prefix, format string, args ...interface{}) {
	if IsDebug() {
		fmt.Fprintf(os.Stderr, "["+prefix+"] "+format+"\n", args...)
	}
}

// Info logs an info message (always shown)
func Info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

// Warn logs a warning message (always shown)
func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// Error logs an error message (always shown)
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
