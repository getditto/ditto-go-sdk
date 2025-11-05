package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"
*/
import "C"
import (
	"fmt"
)

// FFI error code to description mapping based on dittoffi.h
// These descriptions come from the comments in the header file
var ffiErrorDescriptions = map[int]string{
	int(C.DITTOFFI_ERROR_CODE_ACTIVATION_LICENSE_TOKEN_EXPIRED):              "The license is valid but expired",
	int(C.DITTOFFI_ERROR_CODE_ACTIVATION_LICENSE_TOKEN_INVALID):              "The license signature failed verification. This could be due to incorrectly encoded/truncated data or could indicate tampering",
	int(C.DITTOFFI_ERROR_CODE_ACTIVATION_LICENSE_UNSUPPORTED_FUTURE_VERSION): "The provided license data was from a future version of Ditto and is incompatible with this version",
	int(C.DITTOFFI_ERROR_CODE_ACTIVATION_NOT_ACTIVATED):                      "The operation failed because the Ditto instance is not yet activated, which is achieved by setting a valid license token",
	int(C.DITTOFFI_ERROR_CODE_ACTIVATION_UNNECESSARY):                        "Activation (via setting a valid license token) is unnecessary for the active identity type",
	int(C.DITTOFFI_ERROR_CODE_AUTHENTICATION_EXPIRATION_HANDLER_MISSING):     "The operation failed because an authentication expiration handler has not yet been set",
	int(C.DITTOFFI_ERROR_CODE_BASE64_INVALID):                                "Invalid input provided for base64 decoding",
	int(C.DITTOFFI_ERROR_CODE_CBOR_INVALID):                                  "Invalid CBOR-encoded input",
	int(C.DITTOFFI_ERROR_CODE_CBOR_UNSUPPORTED):                              "Unsupported CBOR type",
	int(C.DITTOFFI_ERROR_CODE_CRDT):                                          "CRDT operation error",
	int(C.DITTOFFI_ERROR_CODE_DIFFER_IDENTITY_KEY_PATH_INVALID):              "Invalid identity key path for differ",
	int(C.DITTOFFI_ERROR_CODE_DQL_EVALUATION_ERROR):                          "DQL query execution failed in flight",
	int(C.DITTOFFI_ERROR_CODE_DQL_INVALID_QUERY_ARGS):                        "Invalid CBOR-encoded query arguments",
	int(C.DITTOFFI_ERROR_CODE_DQL_QUERY_COMPILATION):                         "Failed to compile the given query. For more information on Ditto's query language see: https://ditto.com/link/dql-guide",
	int(C.DITTOFFI_ERROR_CODE_DQL_UNSUPPORTED):                               "Unsupported features were used in a DQL statement or query",
	int(C.DITTOFFI_ERROR_CODE_ENCRYPTION_EXTRANEOUS_PASSPHRASE_GIVEN):        "Unexpected passphrase provided for the currently unencrypted store",
	int(C.DITTOFFI_ERROR_CODE_ENCRYPTION_PASSPHRASE_INVALID):                 "Incorrect passphrase provided for the currently encrypted store",
	int(C.DITTOFFI_ERROR_CODE_ENCRYPTION_PASSPHRASE_NOT_GIVEN):               "Missing passphrase for the currently encrypted store",
	int(C.DITTOFFI_ERROR_CODE_JS_FLOATING_STORE_OPERATION):                   "Javascript only. Missing `await` on outstanding store operation",
	int(C.DITTOFFI_ERROR_CODE_IO_ALREADY_EXISTS):                             "An I/O operation failed because the specified entity (such as a file) already exists",
	int(C.DITTOFFI_ERROR_CODE_IO_NOT_FOUND):                                  "An I/O operation failed because the specified entity (such as a file) was not found",
	int(C.DITTOFFI_ERROR_CODE_IO_OPERATION_FAILED):                           "An I/O operation failed",
	int(C.DITTOFFI_ERROR_CODE_IO_PERMISSION_DENIED):                          "An I/O operation failed because the necessary privileges to complete it were not present",
	int(C.DITTOFFI_ERROR_CODE_LOCKED_DITTO_WORKING_DIRECTORY):                "Outstanding usage of ditto's working directory detected when trying to instantiate a new `Ditto`, which would have led to concurrent usage of the backing database files",
	int(C.DITTOFFI_ERROR_CODE_PARAMETER_QUERY):                               "A query to alter or retrieve a system parameter (ALTER SYSTEM or SHOW) failed",
	int(C.DITTOFFI_ERROR_CODE_STORE_DATABASE):                                "Store database error",
	int(C.DITTOFFI_ERROR_CODE_STORE_DOCUMENT_ID):                             "Found an invalid document id",
	int(C.DITTOFFI_ERROR_CODE_STORE_DOCUMENT_NOT_FOUND):                      "The requested document could not be found",
	int(C.DITTOFFI_ERROR_CODE_STORE_QUERY):                                   "Store query error",
	int(C.DITTOFFI_ERROR_CODE_STORE_TRANSACTION_READ_ONLY):                   "A mutating DQL query was attempted using a read-only transaction",
	int(C.DITTOFFI_ERROR_CODE_TRANSPORT):                                     "Error from the transport layer",
	int(C.DITTOFFI_ERROR_CODE_UNSUPPORTED):                                   "Feature is not (yet?) supported (on this platform?). See the documentation of the feature for more info",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_DEPTH_LIMIT_EXCEEDED):               "Exceeded a depth limit",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_CBOR):                       "Invalid CBOR provided",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_JSON):                       "Invalid JSON provided",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_TRANSPORT_CONFIG):           "Invalid TransportConfig",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_DITTO_CONFIG):               "Invalid DittoConfig",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_NOT_A_MAP):                          "The value was not a map",
	int(C.DITTOFFI_ERROR_CODE_VALIDATION_SIZE_LIMIT_EXCEEDED):                "Exceeded the size limit",
	int(C.DITTOFFI_ERROR_CODE_UNKNOWN):                                       "An unknown error occurred",
	int(C.DITTOFFI_ERROR_CODE_INTERNAL):                                      "Some not-yet-categorized error occurred",
}

// DittoError represents an error from the Ditto SDK with FFI error codes.
// This is defined here to avoid circular imports with the ditto package.
type DittoError struct {
	// Code is the FFI error code from dittoffi_error_code_t
	Code int
	// Message provides a human-readable description of the error
	Message string
}

// Error implements the error interface for DittoError.
func (e *DittoError) Error() string {
	return fmt.Sprintf("DittoError(%d): %s", e.Code, e.Message)
}

// ConvertFFIErrorToDittoError converts an FFI error to a DittoError with proper error code and message.
// It extracts the error code using C.dittoffi_error_code() and gets the message using C.dittoffi_error_description().
// If no message is provided by the FFI, it uses the default description from ffiErrorDescriptions.
// Precondition: ffiError is not nil.
func ConvertFFIErrorToDittoError(ffiError *C.dittoffi_error_t) *DittoError {
	// Extract error code using the FFI function
	errorCode := int(C.dittoffi_error_code(ffiError))

	// Extract error message using the FFI function
	var message string
	if msgPtr := C.dittoffi_error_description(ffiError); msgPtr != nil {
		message = C.GoString(msgPtr)
		C.ditto_c_string_free(msgPtr)
	}

	// If no message was provided by FFI, use our default description
	if message == "" {
		if defaultMsg, exists := ffiErrorDescriptions[errorCode]; exists {
			message = defaultMsg
		} else {
			message = fmt.Sprintf("FFI error code %d", errorCode)
		}
	}

	return &DittoError{
		Code:    errorCode,
		Message: message,
	}
}

// GetFFIErrorCode returns the error code for the given FFI error
func (e *DittoError) GetFFIErrorCode() int {
	return e.Code
}

// GetMessage returns the error message for the given FFI error
func (e *DittoError) GetMessage() string {
	return e.Message
}

// GetFFIErrorDescription returns the default description for an FFI error code
func GetFFIErrorDescription(errorCode int) string {
	if desc, exists := ffiErrorDescriptions[errorCode]; exists {
		return desc
	}
	return fmt.Sprintf("Unknown FFI error code %d", errorCode)
}
