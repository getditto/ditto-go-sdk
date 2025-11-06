// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"

// Forward declarations for Go callbacks
extern void goLogExportCallback(void* context, dittoffi_result_uint64_t result);
extern void goLogExportFree(void* context);
extern void goLogCallback(CLogLevel_t level, char* message);

// Helper to create log export callback struct
static continuation_dittoffi_result_uint64_t create_log_export_callback(void* context) {
	continuation_dittoffi_result_uint64_t cb;
    cb.env_ptr = context;
    cb.call = goLogExportCallback;
    cb.free = goLogExportFree;
	return cb;
}

// Helper to set the logger callback function
static void logger_enable_custom_log_cb() {
	ditto_logger_set_custom_log_cb(goLogCallback);
}

// Helper to unset the logger callback function
static void logger_disable_custom_log_cb() {
	ditto_logger_set_custom_log_cb(NULL);
}
*/
import "C"
import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"
)

// LogExport callback management
var (
	// Thread-safe log export callback management
	logExportCallbacks      sync.Map      // map[uintptr]func(uint64, error)
	nextLogExportCallbackID atomic.Uint64 // atomic counter
)

// RegisterLogExportCallback registers a log export callback and returns its ID
func RegisterLogExportCallback(callback func(uint64, error)) uintptr {
	id := uintptr(nextLogExportCallbackID.Add(1))
	logExportCallbacks.Store(id, callback)
	return id
}

// UnregisterLogExportCallback unregisters a log export callback
func UnregisterLogExportCallback(id uintptr) {
	logExportCallbacks.Delete(id)
}

// GetLogExportCallback retrieves a log export callback by ID
func GetLogExportCallback(id uintptr) func(uint64, error) {
	if callback, exists := logExportCallbacks.Load(id); exists {
		if cb, ok := callback.(func(uint64, error)); ok {
			return cb
		}
	}
	return nil
}

//export goLogExportCallback
func goLogExportCallback(contextPtr unsafe.Pointer, result C.dittoffi_result_uint64_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in log export callback: %v\n%s", r, debug.Stack())
		}
	}()

	id := uintptr(contextPtr)
	callback := GetLogExportCallback(id)
	if callback == nil {
		return
	}

	// Check if there's an error
	if result.error != nil {
		// Convert C error to Go error
		// Note: errorFromFFIError consumes the error, so we can't reuse it
		err := errorFromFFIError(result.error)
		callback(0, err)
	} else {
		// Success - return the bytes written
		callback(uint64(result.success), nil)
	}
}

//export goLogExportFree
func goLogExportFree(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in log export free callback: %v\n%s", r, debug.Stack())
		}
	}()

	// Extract the callback ID and unregister it
	id := uintptr(contextPtr)
	UnregisterLogExportCallback(id)
}

// CreateLogExportCallback creates a C callback struct for exporting logs
func CreateLogExportCallback(callbackID uintptr) C.continuation_dittoffi_result_uint64_t {
	return C.create_log_export_callback(unsafe.Pointer(callbackID))
}

// Logger callback management
var (
	loggerCallbackMu sync.RWMutex
	loggerCallback   func(LogLevel, string)
)

// SetCustomLogCallback sets a custom callback for log messages
func SetCustomLogCallback(callback func(LogLevel, string)) {
	ffiDebugTrace("SetCustomLogCallback called")

	// Set the callback in our registry
	setLoggerCallback(callback)

	// Register the callback with the FFI layer
	if callback != nil {
		C.logger_enable_custom_log_cb()
	} else {
		C.logger_disable_custom_log_cb()
	}
}

// setLoggerCallback sets the global logger callback
func setLoggerCallback(callback func(LogLevel, string)) {
	loggerCallbackMu.Lock()
	defer loggerCallbackMu.Unlock()
	loggerCallback = callback
}

//export goLogCallback
func goLogCallback(level C.CLogLevel_t, message *C.char) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in log callback: %v\n%s", r, debug.Stack())
		}
	}()

	// Get the current callback
	loggerCallbackMu.RLock()
	defer loggerCallbackMu.RUnlock()
	if loggerCallback == nil {
		// No callback registered, nothing to do
		return
	}

	// Convert C log level to Go log level
	goLevel := LogLevel(level)

	// Convert C string to Go string
	goMessage := C.GoString(message)

	// Call the Go callback
	loggerCallback(goLevel, goMessage)
}

// LogLevel represents the log level
type LogLevel C.CLogLevel_t

const (
	LogLevelError   LogLevel = C.C_LOG_LEVEL_ERROR
	LogLevelWarning LogLevel = C.C_LOG_LEVEL_WARNING
	LogLevelInfo    LogLevel = C.C_LOG_LEVEL_INFO
	LogLevelDebug   LogLevel = C.C_LOG_LEVEL_DEBUG
	LogLevelVerbose LogLevel = C.C_LOG_LEVEL_VERBOSE
)

// SetMinimumLogLevel sets the minimum log level
func SetMinimumLogLevel(level LogLevel) {
	ffiDebugTraceF("SetMinimumLogLevel: %d", level)
	C.ditto_logger_minimum_log_level(C.CLogLevel_t(level))
}

// GetLogLevel gets the current minimum log level
func GetLogLevel() LogLevel {
	ffiDebugTrace("GetLogLevel called")
	return LogLevel(C.ditto_logger_minimum_log_level_get())
}

// SetLoggerEnabled enables or disables logging
func SetLoggerEnabled(enabled bool) {
	ffiDebugTraceF("SetLoggerEnabled: %v", enabled)
	C.ditto_logger_enabled(C._Bool(enabled))
}

// GetLoggerEnabled returns whether logging is enabled
func GetLoggerEnabled() bool {
	ffiDebugTrace("GetLoggerEnabled called")
	return bool(C.ditto_logger_enabled_get())
}

// SetEmojiLogLevelHeadingsEnabled enables or disables emoji headings in logs
func SetEmojiLogLevelHeadingsEnabled(enabled bool) {
	ffiDebugTraceF("SetEmojiLogLevelHeadingsEnabled: %v", enabled)
	C.ditto_logger_emoji_headings_enabled(C._Bool(enabled))
}

// GetEmojiLogLevelHeadingsEnabled returns whether emoji headings are enabled
func GetEmojiLogLevelHeadingsEnabled() bool {
	ffiDebugTrace("GetEmojiLogLevelHeadingsEnabled called")
	return bool(C.ditto_logger_emoji_headings_enabled_get())
}

// InitLogger initializes the logger
func InitLogger() {
	ffiDebugTrace("InitLogger called")
	C.ditto_logger_init()
}

// SetLogFile sets the log file path
func SetLogFile(path string) error {
	ffiDebugTraceF("SetLogFile: %s", path)

	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	result := C.ditto_logger_set_log_file(cPath)
	if result != 0 {
		return fmt.Errorf("failed to set log file: error code %d", result)
	}

	return nil
}

// LoggerExportToFileAsync exports logs to a file asynchronously
func LoggerExportToFileAsync(destPath string, callback func(uint64, error)) {
	ffiDebugTraceF("LoggerExportToFileAsync: %s", destPath)

	cPath := C.CString(destPath)
	defer C.free(unsafe.Pointer(cPath))

	// Register the callback and get its ID
	callbackID := RegisterLogExportCallback(callback)

	continuation := CreateLogExportCallback(callbackID)

	// Call the FFI function
	C.dittoffi_logger_try_export_to_file_async(cPath, continuation)
}

// LoggerExportToFileBlocking exports logs to a file and blocks until complete
func LoggerExportToFileBlocking(destPath string) (uint64, error) {
	ffiDebugTraceF("LoggerExportToFileBlocking: %s", destPath)

	done := make(chan struct{})
	var bytesWritten uint64
	var exportErr error

	LoggerExportToFileAsync(destPath, func(bytes uint64, err error) {
		bytesWritten = bytes
		exportErr = err
		close(done)
	})

	<-done
	return bytesWritten, exportErr
}

// Log sends a log message through the Ditto logging system
func Log(level LogLevel, message string) {
	ffiDebugTraceF("Log(level=%d): %s", level, message)

	// Convert Go log level to C log level
	var cLevel C.CLogLevel_t
	switch level {
	case LogLevelError:
		cLevel = C.C_LOG_LEVEL_ERROR
	case LogLevelWarning:
		cLevel = C.C_LOG_LEVEL_WARNING
	case LogLevelInfo:
		cLevel = C.C_LOG_LEVEL_INFO
	case LogLevelDebug:
		cLevel = C.C_LOG_LEVEL_DEBUG
	case LogLevelVerbose:
		cLevel = C.C_LOG_LEVEL_VERBOSE
	default:
		cLevel = C.C_LOG_LEVEL_INFO // Default to info if unknown
	}

	// Convert Go string to C string
	cMessage := C.CString(message)
	defer C.free(unsafe.Pointer(cMessage))

	// Call the FFI function
	C.ditto_log(cLevel, cMessage)
}
