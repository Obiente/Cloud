package logger

import (
	"log"
	"os"
)

var (
	debugEnabled bool
)

// Init initializes the logger
func Init() {
	debugEnabled = os.Getenv("LOG_LEVEL") == "debug"
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// Debug logs a debug message (only if debug is enabled)
func Debug(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	log.Fatalf("[FATAL] "+format, args...)
}

