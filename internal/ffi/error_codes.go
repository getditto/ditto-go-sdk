package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"
*/
import "C"

// FFI Error Code Constants
// These constants are exported from the internal/ffi package to allow
// the ditto package to use them for error mappings.

const (
	// Activation-related errors
	ErrorCodeActivationLicenseTokenExpired             = int(C.DITTOFFI_ERROR_CODE_ACTIVATION_LICENSE_TOKEN_EXPIRED)
	ErrorCodeActivationLicenseTokenInvalid             = int(C.DITTOFFI_ERROR_CODE_ACTIVATION_LICENSE_TOKEN_INVALID)
	ErrorCodeActivationLicenseUnsupportedFutureVersion = int(C.DITTOFFI_ERROR_CODE_ACTIVATION_LICENSE_UNSUPPORTED_FUTURE_VERSION)
	ErrorCodeActivationNotActivated                    = int(C.DITTOFFI_ERROR_CODE_ACTIVATION_NOT_ACTIVATED)
	ErrorCodeActivationUnnecessary                     = int(C.DITTOFFI_ERROR_CODE_ACTIVATION_UNNECESSARY)

	// Authentication-related errors
	ErrorCodeAuthenticationExpirationHandlerMissing = int(C.DITTOFFI_ERROR_CODE_AUTHENTICATION_EXPIRATION_HANDLER_MISSING)

	// Encoding-related errors
	ErrorCodeBase64Invalid   = int(C.DITTOFFI_ERROR_CODE_BASE64_INVALID)
	ErrorCodeCborInvalid     = int(C.DITTOFFI_ERROR_CODE_CBOR_INVALID)
	ErrorCodeCborUnsupported = int(C.DITTOFFI_ERROR_CODE_CBOR_UNSUPPORTED)

	// CRDT and differ errors
	ErrorCodeCrdt                         = int(C.DITTOFFI_ERROR_CODE_CRDT)
	ErrorCodeDifferIdentityKeyPathInvalid = int(C.DITTOFFI_ERROR_CODE_DIFFER_IDENTITY_KEY_PATH_INVALID)

	// DQL-related errors
	ErrorCodeDqlEvaluationError  = int(C.DITTOFFI_ERROR_CODE_DQL_EVALUATION_ERROR)
	ErrorCodeDqlInvalidQueryArgs = int(C.DITTOFFI_ERROR_CODE_DQL_INVALID_QUERY_ARGS)
	ErrorCodeDqlQueryCompilation = int(C.DITTOFFI_ERROR_CODE_DQL_QUERY_COMPILATION)
	ErrorCodeDqlUnsupported      = int(C.DITTOFFI_ERROR_CODE_DQL_UNSUPPORTED)

	// Encryption-related errors
	ErrorCodeEncryptionExtraneousPassphraseGiven = int(C.DITTOFFI_ERROR_CODE_ENCRYPTION_EXTRANEOUS_PASSPHRASE_GIVEN)
	ErrorCodeEncryptionPassphraseInvalid         = int(C.DITTOFFI_ERROR_CODE_ENCRYPTION_PASSPHRASE_INVALID)
	ErrorCodeEncryptionPassphraseNotGiven        = int(C.DITTOFFI_ERROR_CODE_ENCRYPTION_PASSPHRASE_NOT_GIVEN)

	// JavaScript-specific errors
	ErrorCodeJsFloatingStoreOperation = int(C.DITTOFFI_ERROR_CODE_JS_FLOATING_STORE_OPERATION)

	// I/O-related errors
	ErrorCodeIoAlreadyExists    = int(C.DITTOFFI_ERROR_CODE_IO_ALREADY_EXISTS)
	ErrorCodeIoNotFound         = int(C.DITTOFFI_ERROR_CODE_IO_NOT_FOUND)
	ErrorCodeIoOperationFailed  = int(C.DITTOFFI_ERROR_CODE_IO_OPERATION_FAILED)
	ErrorCodeIoPermissionDenied = int(C.DITTOFFI_ERROR_CODE_IO_PERMISSION_DENIED)

	// Directory and resource errors
	ErrorCodeLockedDittoWorkingDirectory = int(C.DITTOFFI_ERROR_CODE_LOCKED_DITTO_WORKING_DIRECTORY)

	// Parameter query errors
	ErrorCodeParameterQuery = int(C.DITTOFFI_ERROR_CODE_PARAMETER_QUERY)

	// Store-related errors
	ErrorCodeStoreDatabase            = int(C.DITTOFFI_ERROR_CODE_STORE_DATABASE)
	ErrorCodeStoreDocumentId          = int(C.DITTOFFI_ERROR_CODE_STORE_DOCUMENT_ID)
	ErrorCodeStoreDocumentNotFound    = int(C.DITTOFFI_ERROR_CODE_STORE_DOCUMENT_NOT_FOUND)
	ErrorCodeStoreQuery               = int(C.DITTOFFI_ERROR_CODE_STORE_QUERY)
	ErrorCodeStoreTransactionReadOnly = int(C.DITTOFFI_ERROR_CODE_STORE_TRANSACTION_READ_ONLY)

	// Transport-related errors
	ErrorCodeTransport = int(C.DITTOFFI_ERROR_CODE_TRANSPORT)

	// Feature support errors
	ErrorCodeUnsupported = int(C.DITTOFFI_ERROR_CODE_UNSUPPORTED)

	// Validation-related errors
	ErrorCodeValidationDepthLimitExceeded     = int(C.DITTOFFI_ERROR_CODE_VALIDATION_DEPTH_LIMIT_EXCEEDED)
	ErrorCodeValidationInvalidCbor            = int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_CBOR)
	ErrorCodeValidationInvalidJson            = int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_JSON)
	ErrorCodeValidationInvalidTransportConfig = int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_TRANSPORT_CONFIG)
	ErrorCodeValidationInvalidDittoConfig     = int(C.DITTOFFI_ERROR_CODE_VALIDATION_INVALID_DITTO_CONFIG)
	ErrorCodeValidationNotAMap                = int(C.DITTOFFI_ERROR_CODE_VALIDATION_NOT_A_MAP)
	ErrorCodeValidationSizeLimitExceeded      = int(C.DITTOFFI_ERROR_CODE_VALIDATION_SIZE_LIMIT_EXCEEDED)

	// Generic errors
	ErrorCodeUnknown  = int(C.DITTOFFI_ERROR_CODE_UNKNOWN)
	ErrorCodeInternal = int(C.DITTOFFI_ERROR_CODE_INTERNAL)
)
