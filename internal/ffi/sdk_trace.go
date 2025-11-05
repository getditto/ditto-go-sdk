package ffi

import (
	"fmt"
	"log"
)

// enableFFIDebugTrace must be set true to enable internal debugging trace log messages in the ditto package.
//
// Internal FFI trace messages are logged using Go's standard log package.
//
// This is not exposed in the public API, and it is not intended for general use.
// The set of trace log messages and their formats are subject to change, so don't rely on them.
//
// DO NOT COMMIT A CHANGE OF THIS VALUE TO true
const enableFFIDebugTrace = false

// ffiDebugTrace writes a debug-level log message if enableFFIDebugTrace is true.
// Otherwise, it is a no-op.
func ffiDebugTrace(message string) {
	if enableFFIDebugTrace {
		log.Printf("gosdk/ffi: %s\n", message)
	}
}

// ffiDebugTraceF formats and writes a debug+level log message if enableFFIDebugTrace is true.
// Otherwise, it is a no-op.
func ffiDebugTraceF(format string, args ...any) {
	if enableFFIDebugTrace {
		message := fmt.Sprintf(format, args...)
		log.Printf("gosdk/ffi: %s\n", message)
	}
}
