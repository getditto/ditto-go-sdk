package ditto

import (
	"fmt"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// enableSDKDebugTrace must be set true to enable internal debugging trace log messages in the ditto package.
//
// Internal trace messages are logged at Debug level, so you need to set the log level
// to that or higher to see them.
//
// This is not exposed in the public API, and it is not intended for general use.
// The set of trace log messages and their formats are subject to change, so don't rely on them.
//
// DO NOT COMMIT A CHANGE OF THIS VALUE TO true
const enableSDKDebugTrace = false

// sdkDebugTrace writes a debug-level log message if enableSDKDebugTrace is true.
// Otherwise, it is a no-op.
func sdkDebugTrace(message string) {
	if enableSDKDebugTrace {
		LogDebug(message)
	}
}

// sdkDebugTraceF formats and writes a debug-level log message if enableSDKDebugTrace is true.
// Otherwise, it is a no-op.
func sdkDebugTraceF(format string, args ...any) {
	if enableSDKDebugTrace {
		LogDebugF(format, args...)
	}
}

// LogLevel represents the different log levels supported by the Ditto logger.
type LogLevel ffi.LogLevel

const (
	// LogLevelError logs only errors
	LogLevelError = LogLevel(ffi.LogLevelError)

	// LogLevelWarning logs warnings and above
	LogLevelWarning = LogLevel(ffi.LogLevelWarning)

	// LogLevelInfo logs info and above
	LogLevelInfo = LogLevel(ffi.LogLevelInfo)

	// LogLevelDebug logs debug and above
	LogLevelDebug = LogLevel(ffi.LogLevelDebug)

	// LogLevelVerbose logs everything
	LogLevelVerbose = LogLevel(ffi.LogLevelVerbose)
)

// String returns the string representation of the LogLevel
func (l LogLevel) String() string {
	switch l {
	case LogLevelError:
		return "Error"
	case LogLevelWarning:
		return "Warning"
	case LogLevelInfo:
		return "Info"
	case LogLevelDebug:
		return "Debug"
	case LogLevelVerbose:
		return "Verbose"
	default:
		return "Unknown"
	}
}

// SetMinimumLogLevel sets the minimum log level at which logs will be logged, provided the logger is enabled.
//
// For example if this is set to DittoLogLevel.warning, then only logs that are logged with the LogLevelWarning or
// LogLevelError log levels will be shown.
//
// Logs exported through ExportLog are not affected by this setting and include all logs at LogLevelDebug and above.
func SetMinimumLogLevel(level LogLevel) {
	ffiLevel := ffi.LogLevel(level)
	ffi.SetMinimumLogLevel(ffiLevel)
}

// GetMinimumLogLevel returns the minimum log level at which logs will be logged, provided the logger is enabled.
//
// For example if this is set to DittoLogLevel.warning, then only logs that are logged with the LogLevelWarning or
// LogLevelError log levels will be shown.
//
// Logs exported through ExportLog are not affected by this setting and include all logs at LogLevelDebug and above.
func GetMinimumLogLevel() LogLevel {
	ffiLevel := ffi.GetLogLevel()
	return LogLevel(ffiLevel)
}

// SetCustomLogCallback sets a custom callback for log messages.
// The callback receives the log level and message from the Ditto core.
//
// When a custom callback is set, it will receive all log messages that pass
// through the Ditto logging system, regardless of the current log level setting.
// The callback is called on the thread that generates the log message.
//
// Parameters:
//   - callback: A function that receives log messages. If nil, removes the current callback.
func SetCustomLogCallback(callback func(level LogLevel, message string)) {
	// Register the callback with the FFI layer
	if callback != nil {
		// Wrap the callback to convert between LogLevel types
		ffiCallback := func(level ffi.LogLevel, message string) {
			dittoLevel := LogLevel(level)
			callback(dittoLevel, message)
		}
		ffi.SetCustomLogCallback(ffiCallback)
	} else {
		ffi.SetCustomLogCallback(nil)
	}
}

// LogInfo logs an info message
func LogInfo(message string) {
	logMessage(LogLevelInfo, message)
}

// LogDebug logs a debug message
func LogDebug(message string) {
	logMessage(LogLevelDebug, message)
}

// LogWarning logs a warning message
func LogWarning(message string) {
	logMessage(LogLevelWarning, message)
}

// LogError logs an error message
func LogError(message string) {
	logMessage(LogLevelError, message)
}

// LogVerbose logs a verbose message
func LogVerbose(message string) {
	logMessage(LogLevelVerbose, message)
}

// logMessage is the internal function to log messages
func logMessage(level LogLevel, message string) {
	// Use FFI logging which will trigger custom callback if set
	// Let the FFI layer handle log level filtering
	ffiLevel := ffi.LogLevel(level)
	ffi.Log(ffiLevel, message)
}

// SetLogFile registers a file path where logs will be written to, whenever Ditto wants to issue a log
// (in addition to emitting the log to the console).
//
// If the given path is the empty string, then the current logging file, if any, is unregistered.
// Otherwise, the file path must be within an already existing directory
func SetLogFile(logFilePath string) error {
	return ffi.SetLogFile(logFilePath)
}

// SetLoggerEnabled enables or disables logging.
//
// Logs exported through ExportLog are not affected by this setting and will also include logs emitted while enabled is false.
func SetLoggerEnabled(enabled bool) {
	ffi.SetLoggerEnabled(enabled)
}

// IsLoggerEnabled returns whether logging is enabled.
//
// Logs exported through ExportLog are not affected by this setting and will also include logs emitted while enabled is false
func IsLoggerEnabled() bool {
	return ffi.GetLoggerEnabled()
}

// SetEmojiLogLevelHeadingsEnabled enables or disables whether emoji are used as the log level indicator in logs.
func SetEmojiLogLevelHeadingsEnabled(enabled bool) {
	ffi.SetEmojiLogLevelHeadingsEnabled(enabled)
}

// IsEmojiLogLevelHeadingsEnabled returns whether emoji are used as the log level indicator in logs.
func IsEmojiLogLevelHeadingsEnabled() bool {
	return ffi.GetEmojiLogLevelHeadingsEnabled()
}

// LogInfoF logs a formatted info message
func LogInfoF(format string, args ...any) {
	LogInfo(fmt.Sprintf(format, args...))
}

// LogDebugF logs a formatted debug message
func LogDebugF(format string, args ...any) {
	LogDebug(fmt.Sprintf(format, args...))
}

// LogWarningF logs a formatted warning message
func LogWarningF(format string, args ...any) {
	LogWarning(fmt.Sprintf(format, args...))
}

// LogErrorF logs a formatted error message
func LogErrorF(format string, args ...any) {
	LogError(fmt.Sprintf(format, args...))
}

// LogVerboseF logs a formatted verbose message
func LogVerboseF(format string, args ...any) {
	LogVerbose(fmt.Sprintf(format, args...))
}

// ExportLog exports collected logs to a compressed file.
//
// Ditto collects a limited amount of diagnostic logs in the background,
// which can be exported to a compressed file for analysis. This includes
// logs from all Ditto instances created in the same process.
//
// The exported file is a gzip-compressed JSON Lines file (`.jsonl.gz`),
// with one JSON object per line representing a log entry. The logs are
// ordered from oldest to newest.
//
// Parameters:
//   - path: The filesystem path where the log file should be written. The file must not
//     already exist, and the containing directory must exist. It is recommended
//     to use the `.jsonl.gz` extension, though not required.
//
// Returns:
//   - The number of bytes written to disk
//   - An error if the export fails (e.g., file already exists, permission denied)
//
// Note: This function works independently of the current log level setting
// and exports all logs at Debug level and above. Logs are limited to 15 MB
// and a maximum age of 3 days.
func ExportLog(path string) (uint64, error) {
	sdkDebugTraceF("gosdk: ExportLog(); path: %s", path)
	return ffi.LoggerExportToFileBlocking(path)
}
