package logger

import (
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	currentLevel LogLevel
	initialized  bool
)

// Init initializes the logger with the LOG_LEVEL environment variable
func Init() {
	levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	
	switch levelStr {
	case "debug", "trace":
		currentLevel = LevelDebug
	case "info":
		currentLevel = LevelInfo
	case "warn", "warning":
		currentLevel = LevelWarn
	case "error":
		currentLevel = LevelError
	default:
		// Default to info if not set or invalid
		currentLevel = LevelInfo
	}
	
	initialized = true
}

// shouldLog checks if a log level should be logged based on current level
func shouldLog(level LogLevel) bool {
	if !initialized {
		Init()
	}
	return level >= currentLevel
}

// Debug logs debug messages (only if LOG_LEVEL is debug or trace)
func Debug(format string, v ...interface{}) {
	if shouldLog(LevelDebug) {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs info messages (if LOG_LEVEL is debug, trace, or info)
func Info(format string, v ...interface{}) {
	if shouldLog(LevelInfo) {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warn logs warning messages (if LOG_LEVEL is debug, trace, info, or warn)
func Warn(format string, v ...interface{}) {
	if shouldLog(LevelWarn) {
		log.Printf("[WARN] "+format, v...)
	}
}

// Error logs error messages (always logged)
func Error(format string, v ...interface{}) {
	if shouldLog(LevelError) {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Debugln logs debug messages (only if LOG_LEVEL is debug or trace)
func Debugln(v ...interface{}) {
	if shouldLog(LevelDebug) {
		log.Println(append([]interface{}{"[DEBUG]"}, v...)...)
	}
}

// Infoln logs info messages (if LOG_LEVEL is debug, trace, or info)
func Infoln(v ...interface{}) {
	if shouldLog(LevelInfo) {
		log.Println(append([]interface{}{"[INFO]"}, v...)...)
	}
}

// Warnln logs warning messages (if LOG_LEVEL is debug, trace, info, or warn)
func Warnln(v ...interface{}) {
	if shouldLog(LevelWarn) {
		log.Println(append([]interface{}{"[WARN]"}, v...)...)
	}
}

// Errorln logs error messages (always logged)
func Errorln(v ...interface{}) {
	if shouldLog(LevelError) {
		log.Println(append([]interface{}{"[ERROR]"}, v...)...)
	}
}

// Fatal logs error messages and exits (always logged)
func Fatalf(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}

// GetLevel returns the current log level as a string
func GetLevel() string {
	if !initialized {
		Init()
	}
	switch currentLevel {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	return shouldLog(LevelDebug)
}

