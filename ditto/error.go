// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"fmt"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

var (
	ErrDittoClosed = &DittoError{Code: ffi.ErrorCodeInternal, Message: "Ditto instance is closed"} // TODO: should this even be a DittoError?
)

// TODO: This DittoError (as opposed to ffi.DittoError) is mostly used to wrap ffi.DittoError
// In addition, ErrDittoClosed (now extracted from its duplicates) is used in many places when the ditto instance is closed.
// Otherwise, there are only a handful of constructed DittoError-s with Code=ffi.ErrorCodeInternal and a unique message.
// Outside of DittoError, there are many (41) uses of fmt.Errorf() with unique messages.

// DittoError represents an error from the Ditto SDK.
//
// All errors that are returned by the Ditto SDK are wrapped as a DittoError.
// This type wraps multiple different types of error that each have an associated
// reason. You can access more specific information about an error by checking
// the error's Code value.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type DittoError struct {
	// Code is the FFI error code from dittoffi_error_code_t enumeration.
	Code int

	// Message provides a human-readable description of the error.
	Message string

	// Err is the underlying error that caused this DittoError, if any.
	// This allows for error wrapping and unwrapping using the standard
	// errors.Unwrap function.
	Err error
}

// Error implements the error interface for DittoError.
//
// Returns a formatted string containing both the error code and message
// in the format: "DittoError(code): message"
// If there's an underlying error, it includes that information as well.
func (e *DittoError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("DittoError(%d): %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("DittoError(%d): %s", e.Code, e.Message)
}

// Unwrap implements the errors.Unwrap interface for DittoError.
//
// This allows for error chain inspection using errors.Unwrap, errors.As,
// and errors.Is with wrapped errors.
//
// Example:
//
//	err := someOperation()
//	var pathErr *os.PathError
//	if errors.As(err, &pathErr) {
//		// Handle path error
//	}
func (e *DittoError) Unwrap() error {
	return e.Err
}

// Is implements the errors.Is interface for DittoError.
//
// This allows for error comparison using errors.Is, matching errors
// by their code rather than exact instance or message. This is useful
// for checking against common error conditions even when the message
// might differ.
//
// Example:
//
//	err := dittoInstance.Sync().Start()
//	if errors.Is(err, ErrDittoClosed) {
//		// Handle closed instance
//	}
func (e *DittoError) Is(target error) bool {
	if target == nil || e == nil {
		return false
	}

	// Check if target is a DittoError and compare codes
	if targetErr, ok := target.(*DittoError); ok {
		return e.Code == targetErr.Code
	}

	return false
}

// convertFFIError converts an FFI DittoError to the public DittoError type.
// This function handles the interface between the internal FFI layer and the public API.
func convertFFIError(ffiErr error) *DittoError {
	// Handle FFI DittoError by copying its fields
	if ffiDittoErr, ok := ffiErr.(interface {
		GetFFIErrorCode() int
		GetMessage() string
	}); ok {
		return &DittoError{
			Code:    ffiDittoErr.GetFFIErrorCode(),
			Message: ffiDittoErr.GetMessage(),
		}
	}

	// Handle standard error interface
	return &DittoError{Code: ffi.ErrorCodeInternal, Message: "FFI operation failed", Err: ffiErr}
}
