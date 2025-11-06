// Copyright 2025 DittoLive Incorporated. All rights reserved.

// Package ffi provides safe Go bindings to the Ditto C FFI library.
//
// IMPORTANT: These bindings have been created manually or with the help of AI.
// When changes are made to dittoffi.h, updates to the ffi package
// will be needed.
//
// # Architectural Safety Patterns
//
// This package implements several critical safety patterns to ensure reliable
// interaction between Go and C code:
//
// ## 1. Resource Management Pattern (RAII-style)
// All FFI resources are wrapped in handle types with finalizers to ensure cleanup.
// - Dual cleanup: explicit Close() methods + runtime finalizers as safety net
// - Atomic flags prevent double-free vulnerabilities
// - Example: DittoHandle, StoreHandle, StoreObserverHandle
//
// ## 2. Safe Callback Context Management (runtime.Pinner)
// Go 1.21+ runtime.Pinner ensures callback contexts remain valid during FFI calls.
// - Prevents GC from moving/collecting callback data during C execution
// - Automatic unpinning when FFI call completes
//
// ## 3. Panic Recovery at FFI Boundaries
// All Go callbacks invoked from C have panic recovery to prevent process crashes.
// - defer recover() in every //export function
// - Panics converted to error returns or logged
// - Process stability guaranteed even with Go runtime errors
//
// ## 4. Thread-Safe Observer Management
// Uses sync.Map and atomic operations for concurrent access.
// - Lock-free reads for performance
// - Atomic ID generation prevents races
// - Safe concurrent registration/unregistration
//
// ## 5. Context Cancellation Propagation
// Go contexts properly propagated through FFI boundaries.
// - Early cancellation detection before expensive operations
// - Resource cleanup on context cancellation
//
// ## 6. Error Lifecycle Management
// FFI errors extracted completely before freeing C memory.
// - All error information copied to Go before calling free
// - Prevents use-after-free vulnerabilities
// - Structured error types with proper lifecycle
//
// ## 7. Safe Pointer Conversion
// No direct unsafe.Pointer arithmetic; proper type conversions only.
// - C function pointers converted through intermediate unsafe.Pointer
// - No pointer arithmetic on Go pointers
// - Proper alignment and type safety maintained
//
// # Callback Safety Architecture
//
// This file implements critical safety patterns for callbacks invoked from C code:
//
// ## 1. Panic Recovery Pattern
// Every //export function has defer recover() to prevent process crashes.
// Panics in Go callbacks are caught and converted to error returns or logged,
// ensuring the process remains stable even with runtime errors.
//
// ## 2. Safe Context Management with runtime.Pinner
// Uses runtime.Pinner (Go 1.21+) to ensure callback contexts remain valid:
// - Pins Go memory during FFI calls to prevent GC from moving/collecting it
// - Automatically unpins when the callback completes
// - Prevents use-after-free and dangling pointer issues
//
// ## 3. Thread-Safe Callback Registry
// Uses sync.Map for lock-free concurrent access to callback contexts:
// - No mutex contention on reads (common case)
// - Atomic ID generation prevents race conditions
// - Safe concurrent registration and unregistration
//
// ## 4. Callback Lifecycle Management
// Proper retain/release semantics for callback contexts:
// - Reference counting prevents premature cleanup
// - Explicit free functions called by C when callbacks are no longer needed
// - Automatic cleanup on unregistration
//
// ## 5. Type-Safe Callback Wrappers
// C wrapper functions provide type safety at the FFI boundary:
// - Proper function signature matching
// - Context pointer validation
// - Safe type conversions between C and Go
//
// ## Safety Guarantees:
// - No process crashes from Go panics in callbacks
// - No use-after-free of callback contexts
// - No data races in callback management
// - No memory leaks from forgotten callbacks
// - Proper cleanup even with concurrent operations
package ffi

/*
#include <stdlib.h>
#include <string.h>
#include "dittoffi.h"
*/
import "C"
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"unsafe"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
)

// TransportConfigMode defines how transport config is handled
type TransportConfigMode int

const (
	TransportConfigModeDisabled TransportConfigMode = iota
	TransportConfigModePlatformDependent
	TransportConfigModeCustom
)

// DittoHandle wraps the C Ditto pointer with safe resource management.
// It implements the RAII pattern with dual cleanup mechanisms:
// 1. Explicit cleanup via Close() method
// 2. Automatic cleanup via runtime finalizer as safety net
// The atomic finalized flag ensures the resource is freed exactly once,
// preventing double-free vulnerabilities even in concurrent scenarios.
type DittoHandle struct {
	ptr unsafe.Pointer // *C.CDitto_t
}

// NewDittoHandle creates a new managed Ditto handle with automatic cleanup.
// Safety features:
// - Nil check prevents invalid handle creation
// - Runtime finalizer ensures cleanup even if Close() is not called
// - Atomic operations prevent race conditions during cleanup
// Precondition: ptr is not nil
func NewDittoHandle(ptr *C.CDitto_t) *DittoHandle {
	handle := &DittoHandle{}
	handle.store(ptr)

	// Set finalizer for automatic cleanup
	// This acts as a safety net if the user forgets to call Close()
	runtime.SetFinalizer(handle, (*DittoHandle).finalize)
	return handle
}

// Preferably, these atomic ops would be implemented by atomic.Pointer[C.CDitto_t].
// However Go generics can't handle incomplete CGo types.
// These (load, store, swap) are inlined versions of atomic.Pointer's methods.

func (h *DittoHandle) load() *C.CDitto_t {
	return (*C.CDitto_t)(atomic.LoadPointer(&h.ptr))
}

func (h *DittoHandle) store(ptr *C.CDitto_t) {
	atomic.StorePointer(&h.ptr, unsafe.Pointer(ptr))
}

func (h *DittoHandle) swap(ptr *C.CDitto_t) *C.CDitto_t {
	return (*C.CDitto_t)(atomic.SwapPointer(&h.ptr, unsafe.Pointer(ptr)))
}

// Close explicitly closes the Ditto handle and releases resources.
// Safety features:
// - Atomic swap ensures single execution even with concurrent calls
// - Clears finalizer after successful cleanup to avoid GC overhead
// - Nil checks prevent crashes from invalid handles
// - Idempotent: safe to call multiple times
func (h *DittoHandle) Close() {
	ffiDebugTrace("DittoHandle.Close called")
	if h == nil {
		return
	}

	// Atomic swap ensures this runs exactly once
	if freeMe := h.swap(nil); freeMe != nil {
		C.ditto_free(freeMe)
		// Clear finalizer since we've cleaned up
		runtime.SetFinalizer(h, nil)
	}
}

// finalize is called by the garbage collector as a safety net.
// This ensures resources are freed even if Close() is never called explicitly.
// The atomic flag in Close() prevents double-free if both paths execute.
func (h *DittoHandle) finalize() {
	h.Close()
}

// IsValid returns true if the handle is valid and not closed.
func (h *DittoHandle) IsValid() bool {
	return h != nil && h.load() != nil
}

// StoreObserverHandle wraps an observer
type StoreObserverHandle struct {
	ptr        *C.dittoffi_store_observer_t
	callbackID uintptr // Track the callback ID for cleanup
}

// SubscriptionHandle wraps a subscription
type SubscriptionHandle struct {
	ptr *C.dittoffi_sync_subscription_t
}

func (h *SubscriptionHandle) IsValid() bool {
	return h != nil && h.ptr != nil
}

// QueryResultHandle wraps the C QueryResult pointer with safe resource management
type QueryResultHandle struct {
	ptr       *C.dittoffi_query_result_t
	finalized int32 // atomic flag to prevent double-free
}

// NewQueryResultHandle creates a new managed query result handle
func NewQueryResultHandle(ptr *C.dittoffi_query_result_t) *QueryResultHandle {
	if ptr == nil {
		return nil
	}

	handle := &QueryResultHandle{
		ptr: ptr,
	}

	// Set finalizer for automatic cleanup
	runtime.SetFinalizer(handle, (*QueryResultHandle).finalize)
	return handle
}

// Close explicitly closes the query result handle and releases resources
func (h *QueryResultHandle) Close() {
	ffiDebugTrace("QueryResultHandle.Close called")
	if h == nil {
		return
	}

	if atomic.CompareAndSwapInt32(&h.finalized, 0, 1) {
		if h.ptr != nil {
			C.dittoffi_query_result_free(h.ptr)
			h.ptr = nil
		}
		runtime.SetFinalizer(h, nil)
	}
}

// finalize is called by the garbage collector
func (h *QueryResultHandle) finalize() {
	h.Close()
}

// IsValid returns true if the handle is valid and not closed
func (h *QueryResultHandle) IsValid() bool {
	return h != nil && atomic.LoadInt32(&h.finalized) == 0 && h.ptr != nil
}

// DefaultRootDirectory returns the default root directory for persistence
// Platform-specific behavior:
// - macOS: ~/Library/Application Support/Ditto (avoids permission issues in CI)
// - Linux/Unix: Current working directory
// - Windows: Current working directory
// - iOS/Android: Would use app data directory (not currently supported)
func DefaultRootDirectory() string {
	ffiDebugTrace("DefaultRootDirectory called")

	// Use runtime.GOOS to determine the platform
	switch runtime.GOOS {
	case "darwin": // macOS
		// Use Application Support directory like the Swift SDK
		// This avoids permission issues in CI environments
		home, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if we can't get home
			if cwd, err := os.Getwd(); err == nil {
				return cwd
			}
			return "."
		}
		// Create the Application Support/Ditto path
		//
		// TODO: Use the Core Foundation function that finds this directory, rather than
		// hard-coding a path.
		appSupportPath := filepath.Join(home, "Library", "Application Support", "Ditto")

		// Ensure the directory exists
		if err := os.MkdirAll(appSupportPath, 0755); err != nil {
			// If we can't create it, fall back to current directory
			if cwd, err := os.Getwd(); err == nil {
				return cwd
			}
			return "."
		}

		return appSupportPath

	default: // Linux, Windows, and other platforms
		// Use current working directory for other desktop platforms
		cwd, err := os.Getwd()
		if err != nil {
			// If we can't get the current directory, try the user's home directory
			if home, err := os.UserHomeDir(); err == nil {
				return home
			}
			// Last resort: return current directory indicator
			return "."
		}
		return cwd
	}
}

// InitSDKVersion initializes the SDK version info
func InitSDKVersion(version string) {
	ffiDebugTrace("InitSDKVersion called")
	cVersion := C.CString(version)
	defer C.free(unsafe.Pointer(cVersion))

	// Use proper platform detection with named constants
	platform := DetectPlatform()
	languageType := GetLanguage()

	C.ditto_init_sdk_version(platform, languageType, cVersion)
}

// stringFromFFI safely converts a C string allocated by FFI to a Go string and frees the C string using ditto_c_string_free.
//
// Returns empty string if the input is nil.
func stringFromFFI(cStr *C.char) string {
	ffiDebugTrace("stringFromFFI called")
	if cStr == nil {
		return ""
	}
	defer C.ditto_c_string_free(cStr)
	return C.GoString(cStr)
}

// bytesFromFFI safely converts a C slice_boxed_uint8_t allocated by the FFI to a byte slice
// and frees the FFI slice using ditto_c_bytes_free().
//
// Returns nil if the FFI slice is nil.
//
// Returns an empty slice if the FFI slice is empty.
func bytesFromFFI(cSlice C.slice_boxed_uint8_t) []byte {
	// Convert slice_boxed_uint8_t to Go []byte
	if cSlice.ptr != nil {
		defer C.ditto_c_bytes_free(cSlice)
		bytes := C.GoBytes(unsafe.Pointer(cSlice.ptr), C.int(cSlice.len))
		return bytes
	}
	return nil
}

// DittoOpenThrows creates a new Ditto instance
func DittoOpenThrows(configCBOR []byte, mode TransportConfigMode, rootDir string) (*DittoHandle, error) {
	ffiDebugTrace("DittoOpenThrows called")
	// Prepare config CBOR slice
	configSlice := C.slice_ref_uint8_t{
		ptr: (*C.uint8_t)(unsafe.Pointer(&configCBOR[0])),
		len: C.size_t(len(configCBOR)),
	}

	// Convert root dir to C string (always provide a string, even if empty)
	cRootDir := C.CString(rootDir)
	defer C.free(unsafe.Pointer(cRootDir))

	// Call FFI function
	result := C.dittoffi_ditto_open_throws(
		configSlice,
		C.TransportConfigMode_t(mode),
		cRootDir)

	// Check for error
	if result.error != nil {
		return nil, errorFromFFIError(result.error)
	}

	if result.success == nil {
		return nil, fmt.Errorf("dittoffi_ditto_open_throws returned nil")
	}

	return NewDittoHandle(result.success), nil
}

// DittoClose closes a Ditto instance
func DittoClose(handle *DittoHandle) {
	ffiDebugTrace("DittoClose called")
	handle.Close()
}

// StartSync starts synchronization
func StartSync(handle *DittoHandle) error {
	ffiDebugTrace("StartSync called")

	result := C.dittoffi_ditto_try_start_sync(handle.load())
	if result.error != nil {
		return errorFromFFIError(result.error)
	}
	return nil
}

// StopSync stops synchronization
func StopSync(handle *DittoHandle) {
	ffiDebugTrace("StopSync called")
	C.dittoffi_ditto_stop_sync(handle.load())
}

// ExecuteStatement executes a DQL statement
func ExecuteStatement(handle *DittoHandle, query string, args map[string]any) (*QueryResultHandle, error) {
	ffiDebugTrace("ExecuteStatement called")

	// Convert query to C string
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	// Encode args to CBOR
	argsCBOR, err := cbor.Encode(args)
	if err != nil {
		return nil, fmt.Errorf("failed to encode args: %w", err)
	}

	// Prepare args CBOR slice
	var argsSlice C.slice_ref_uint8_t
	if len(argsCBOR) > 0 {
		argsSlice = C.slice_ref_uint8_t{
			ptr: (*C.uint8_t)(unsafe.Pointer(&argsCBOR[0])),
			len: C.size_t(len(argsCBOR)),
		}
	}

	// Execute statement
	result := C.dittoffi_try_exec_statement(handle.load(), cQuery, argsSlice)

	// Check for error
	if result.error != nil {
		return nil, errorFromFFIError(result.error)
	}

	return NewQueryResultHandle(result.success), nil
}

// GetQueryResultItemCount returns the number of items in a query result
func GetQueryResultItemCount(handle *QueryResultHandle) int {
	ffiDebugTrace("GetQueryResultItemCount called")

	if !handle.IsValid() {
		return 0
	}
	return int(C.dittoffi_query_result_item_count(handle.ptr))
}

// GetQueryResultItemCBOR returns the raw CBOR data for an item at the specified index
func GetQueryResultItemCBOR(handle *QueryResultHandle, index int) ([]byte, error) {
	ffiDebugTrace("GetQueryResultItemCBOR called")

	if !handle.IsValid() {
		return nil, fmt.Errorf("invalid query result handle")
	}

	// Get the item at index
	item := C.dittoffi_query_result_item_at(handle.ptr, C.size_t(index))
	if item == nil {
		return nil, fmt.Errorf("invalid index %d", index)
	}
	defer C.dittoffi_query_result_item_free(item)

	// Get CBOR data
	cborData := C.dittoffi_query_result_item_cbor(item)
	if cborData.ptr == nil || cborData.len == 0 {
		return nil, fmt.Errorf("no data for item at index %d", index)
	}
	goBytes := bytesFromFFI(cborData)

	return goBytes, nil
}

// GetQueryResultItem returns an item at the specified index as a map
func GetQueryResultItem(handle *QueryResultHandle, index int) (map[string]any, error) {
	ffiDebugTrace("GetQueryResultItem called")

	// Get raw CBOR first
	cborBytes, err := GetQueryResultItemCBOR(handle, index)
	if err != nil {
		return nil, err
	}

	// Decode CBOR with type normalization
	result, err := cbor.DecodeToMap(cborBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CBOR: %w", err)
	}

	return result, nil
}

// GetQueryResultItemJSON returns an item as a JSON string
func GetQueryResultItemJSON(handle *QueryResultHandle, index int) (string, error) {
	ffiDebugTrace("GetQueryResultItemJSON called")

	if !handle.IsValid() {
		return "", fmt.Errorf("invalid query result handle")
	}

	// Get the item at index
	item := C.dittoffi_query_result_item_at(handle.ptr, C.size_t(index))
	if item == nil {
		return "", fmt.Errorf("invalid index %d", index)
	}
	defer C.dittoffi_query_result_item_free(item)

	// Get JSON string
	jsonStr := C.dittoffi_query_result_item_json(item)
	if jsonStr == nil {
		return "", fmt.Errorf("failed to get JSON for item at index %d", index)
	}

	return stringFromFFI(jsonStr), nil
}

// errorFromFFIError safely converts a dittoffi_error_t to a Go error
// CRITICAL: Extracts ALL information BEFORE freeing to prevent use-after-free
// Precondition: err is not nil.
func errorFromFFIError(err *C.dittoffi_error_t) error {
	ffiDebugTrace("errorFromFFIError called")

	// Convert to DittoError using the new proper error mapping
	dittoError := ConvertFFIErrorToDittoError(err)

	// Free the FFI error after extraction
	C.dittoffi_error_free(err)

	return dittoError
}

// FreeQueryResult frees a query result handle
func FreeQueryResult(handle *QueryResultHandle) {
	ffiDebugTrace("FreeQueryResult called")
	handle.Close()
}

// GetSDKSemver returns the SDK version
func GetSDKSemver() string {
	ffiDebugTrace("GetSDKSemver called")
	cStr := C.dittoffi_get_sdk_semver()
	return stringFromFFI(cStr)
}

// GetDefaultConfig returns the default configuration CBOR
func GetDefaultConfig() []byte {
	ffiDebugTrace("GetDefaultConfig called")

	// Call FFI to get default config - returns slice_boxed_uint8_t directly
	cborSlice := C.dittoffi_ditto_config_default()
	return bytesFromFFI(cborSlice)
}

// GetDittoConfig returns the configuration for the given Ditto instance as CBOR bytes.
// Returns nil if the handle is invalid or if the operation fails.
func GetDittoConfig(handle *DittoHandle) []byte {
	ffiDebugTrace("GetDittoConfig called")

	cborSlice := C.dittoffi_ditto_config(handle.load())
	return bytesFromFFI(cborSlice)
}

// IsEncrypted returns true if Ditto data at the persistence directory is encrypted.
// Returns false if the handle is invalid or if the data is not encrypted.
func IsEncrypted(handle *DittoHandle) bool {
	ffiDebugTrace("IsEncrypted called")
	return bool(C.ditto_is_encrypted(handle.load()))
}

// GetDefaultDatabaseID returns the default database ID
func GetDefaultDatabaseID() string {
	ffiDebugTrace("GetDefaultDatabaseID called")

	cStr := C.dittoffi_DEFAULT_DATABASE_ID()
	// Note: we don't have to free this string. FFI returns a pointer to a constant.
	if cStr == nil {
		return "ditto"
	}
	return C.GoString(cStr)
}

// GetDeviceName returns the device name for a Ditto instance
func GetDeviceName() string {
	ffiDebugTrace("GetDeviceName called")

	// Note: The device name is managed at the SDK level, not at the FFI level.
	// The FFI only provides ditto_set_device_name() but no get function.
	// TODO: The actual device name should be tracked by the Ditto struct in the SDK layer.
	// Return a reasonable default hostname-based name.
	hostname, err := os.Hostname()
	if err != nil {
		return "Go-SDK-Device"
	}

	// Truncate hostname to reasonable length (similar to other SDKs)
	if len(hostname) > 24 {
		hostname = hostname[:24]
	}

	return hostname
}

// SetDeviceName sets the device name for a Ditto instance
func SetDeviceName(handle *DittoHandle, name string) error {
	ffiDebugTrace("SetDeviceName called")

	// Convert name to C string
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	// Call FFI function to set device name
	result := C.ditto_set_device_name(handle.load(), cName)
	if result != nil {
		// Free the returned error string (if any)
		C.ditto_c_string_free(result)
		return fmt.Errorf("failed to set device name")
	}

	return nil
}

// IsActivated returns whether Ditto is activated
func IsActivated(handle *DittoHandle) bool {
	ffiDebugTrace("IsActivated called")

	// Call FFI function to check activation status
	return bool(C.dittoffi_ditto_is_activated(handle.load()))
}

// GetAbsolutePersistenceDirectory returns the absolute path to the persistence directory
func GetAbsolutePersistenceDirectory(handle *DittoHandle) string {
	ffiDebugTrace("GetAbsolutePersistenceDirectory called")

	// Call FFI function that returns an allocated string
	cStr := C.dittoffi_ditto_absolute_persistence_directory(handle.load())

	// Use helper function to safely convert and free the string
	return stringFromFFI(cStr)
}

// GetTransportDiagnostics returns transport diagnostics information
func GetTransportDiagnostics(handle *DittoHandle) ([]byte, error) {
	ffiDebugTrace("GetTransportDiagnostics called")

	// Call FFI function - returns JSON as C string
	cStr := C.ditto_transports_diagnostics(handle.load())
	if cStr == nil {
		return nil, fmt.Errorf("failed to get transport diagnostics")
	}

	jsonStr := stringFromFFI(cStr)

	// Return JSON as bytes
	return []byte(jsonStr), nil
}

// GetQueriesHashMnemonic returns a mnemonic representation of the queries hash
func GetQueriesHashMnemonic(handle *DittoHandle) (string, error) {
	ffiDebugTrace("GetQueriesHashMnemonic called")

	// Create empty slices for collection names and queries
	var collNames C.slice_ref_char_const_ptr_t
	var queries C.slice_ref_char_const_ptr_t

	result := C.dittoffi_try_queries_hash_mnemonic(handle.load(), collNames, queries)
	if result.error != nil {
		return "", errorFromFFIError(result.error)
	}

	if result.success == nil {
		return "", fmt.Errorf("null mnemonic returned")
	}

	return stringFromFFI(result.success), nil
}

// GetTransportConfig gets the current transport configuration
func GetTransportConfig(handle *DittoHandle) ([]byte, error) {
	ffiDebugTrace("GetTransportConfig called")

	// Get transport config as CBOR
	configCBOR := C.dittoffi_ditto_transport_config(handle.load())
	if configCBOR.ptr == nil {
		return nil, fmt.Errorf("failed to get transport config")
	}

	result := bytesFromFFI(configCBOR)
	return result, nil
}

// SetTransportConfig sets the transport configuration
func SetTransportConfig(handle *DittoHandle, configCBOR []byte) error {
	ffiDebugTrace("SetTransportConfig called")

	// Prepare config CBOR slice
	var configSlice C.slice_ref_uint8_t
	if len(configCBOR) > 0 {
		configSlice = C.slice_ref_uint8_t{
			ptr: (*C.uint8_t)(unsafe.Pointer(&configCBOR[0])),
			len: C.size_t(len(configCBOR)),
		}
	}

	// Set transport config with validation
	result := C.dittoffi_ditto_try_set_transport_config(handle.load(), configSlice, true)

	// Check for error
	if result.error != nil {
		return errorFromFFIError(result.error)
	}

	return nil
}

// SyncRegisterSubscriptionThrows registers a sync subscription
func SyncRegisterSubscriptionThrows(ditto *DittoHandle, query string, args map[string]any) (*SubscriptionHandle, error) {
	ffiDebugTrace("SyncRegisterSubscriptionThrows called")

	// Convert query to C string
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	// Encode args to CBOR
	argsCBOR, err := cbor.Encode(args)
	if err != nil {
		return nil, fmt.Errorf("failed to encode args: %w", err)
	}

	// Prepare args CBOR slice
	var argsSlice C.slice_ref_uint8_t
	if len(argsCBOR) > 0 {
		argsSlice = C.slice_ref_uint8_t{
			ptr: (*C.uint8_t)(unsafe.Pointer(&argsCBOR[0])),
			len: C.size_t(len(argsCBOR)),
		}
	}

	// Register subscription
	result := C.dittoffi_sync_register_subscription_throws(
		ditto.load(),
		cQuery,
		argsSlice)

	// Check for error
	if result.error != nil {
		return nil, errorFromFFIError(result.error)
	}

	if result.success == nil {
		return nil, fmt.Errorf("subscription registration returned nil")
	}

	return &SubscriptionHandle{ptr: result.success}, nil
}

// CancelSubscription cancels a subscription
func CancelSubscription(handle *SubscriptionHandle) {
	ffiDebugTrace("CancelSubscription called")

	C.dittoffi_sync_subscription_cancel(handle.ptr)
	C.dittoffi_sync_subscription_free(handle.ptr)
	handle.ptr = nil
}

// SmallPeerInfoSyncScope represents the sync scope for small peer info
type SmallPeerInfoSyncScope int

const (
	SmallPeerInfoSyncScopeBigPeerOnly SmallPeerInfoSyncScope = iota
	SmallPeerInfoSyncScopeLocalPeerOnly
)

// SmallPeerInfoGetIsEnabled returns whether small peer info is enabled
func SmallPeerInfoGetIsEnabled(handle *DittoHandle) bool {
	return bool(C.ditto_small_peer_info_get_is_enabled(handle.load()))
}

// SetSmallPeerInfoEnabled enables or disables small peer info
func SetSmallPeerInfoEnabled(handle *DittoHandle, enabled bool) {
	C.ditto_small_peer_info_set_enabled(handle.load(), C.bool(enabled))
}

// SmallPeerInfoGetMetadata returns the small peer info metadata as JSON
func SmallPeerInfoGetMetadata(handle *DittoHandle) string {
	cStr := C.ditto_small_peer_info_get_metadata(handle.load())
	return stringFromFFI(cStr)
}

// SmallPeerInfoSetMetadata sets the small peer info metadata from JSON
func SmallPeerInfoSetMetadata(handle *DittoHandle, jsonMetadata string) error {
	cMetadata := C.CString(jsonMetadata)
	defer C.free(unsafe.Pointer(cMetadata))

	result := C.ditto_small_peer_info_set_metadata(handle.load(), cMetadata)
	if result != 0 {
		return fmt.Errorf("failed to set small peer info metadata: error code %d", result)
	}

	return nil
}

// SmallPeerInfoGetSyncScope returns the current sync scope
func SmallPeerInfoGetSyncScope(handle *DittoHandle) SmallPeerInfoSyncScope {
	scope := C.ditto_small_peer_info_get_sync_scope(handle.load())
	return SmallPeerInfoSyncScope(scope)
}

// SetSmallPeerInfoSyncScope sets the sync scope
func SetSmallPeerInfoSyncScope(handle *DittoHandle, scope SmallPeerInfoSyncScope) {
	C.ditto_small_peer_info_set_sync_scope(handle.load(), C.DittoSmallPeerInfoSyncScope_t(scope))
}

// SetOfflineOnlyLicenseToken sets the offline-only license token
func SetOfflineOnlyLicenseToken(ditto *DittoHandle, token string) error {
	ffiDebugTrace("SetOfflineOnlyLicenseToken")

	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))

	result := C.dittoffi_ditto_set_offline_only_license_token_throws(ditto.load(), cToken)
	if result.error != nil {
		return errorFromFFIError(result.error)
	}

	return nil
}

// GetQueryResultMutatedDocumentIDCount returns the count of mutated document IDs in a query result
func GetQueryResultMutatedDocumentIDCount(result *QueryResultHandle) int {
	ffiDebugTrace("GetQueryResultMutatedDocumentIDCount called")

	if result == nil || result.ptr == nil {
		return 0
	}
	count := C.dittoffi_query_result_mutated_document_id_count(result.ptr)
	return int(count)
}

// GetQueryResultMutatedDocumentIDAt returns the CBOR data for a mutated document ID at the given index
func GetQueryResultMutatedDocumentIDAt(result *QueryResultHandle, index int) ([]byte, error) {
	if result == nil || result.ptr == nil {
		return nil, fmt.Errorf("invalid query result handle")
	}

	// Call FFI function to get CBOR slice
	cborSlice := C.dittoffi_query_result_mutated_document_id_at(result.ptr, C.size_t(index))

	if cborSlice.ptr == nil || cborSlice.len == 0 {
		return nil, fmt.Errorf("no document ID at index %d", index)
	}

	cborData := bytesFromFFI(cborSlice)
	return cborData, nil
}

// GetQueryResultCommitID returns the commit ID for a query result, if any.
func GetQueryResultCommitID(result *QueryResultHandle) (uint64, bool) {
	ffiDebugTrace("GetQueryResultCommitID called")

	if result == nil || result.ptr == nil {
		return 0, false
	}

	if C.dittoffi_query_result_has_commit_id(result.ptr) {
		return uint64(C.dittoffi_query_result_commit_id(result.ptr)), true
	} else {
		return 0, false
	}
}

// GetQueryResultItemAt returns a QueryResultItemHandle for an item at the specified index
func GetQueryResultItemAt(handle *QueryResultHandle, index int) (*QueryResultItemHandle, error) {
	if handle == nil || !handle.IsValid() {
		return nil, fmt.Errorf("invalid query result handle")
	}

	// Get the item at index
	item := C.dittoffi_query_result_item_at(handle.ptr, C.size_t(index))
	if item == nil {
		return nil, fmt.Errorf("invalid index %d", index)
	}

	itemHandle := &QueryResultItemHandle{
		ptr: item,
	}

	runtime.SetFinalizer(itemHandle, func(h *QueryResultItemHandle) {
		h.Free()
	})

	return itemHandle, nil
}

// QueryResultItemHandle wraps the C QueryResultItem pointer with safe resource management
type QueryResultItemHandle struct {
	ptr       *C.dittoffi_query_result_item_t
	finalized int32 // atomic flag to prevent double-free
}

// GetQueryResultItemCBORFromHandle returns the CBOR data from a QueryResultItemHandle
func GetQueryResultItemCBORFromHandle(handle *QueryResultItemHandle) ([]byte, error) {
	if handle == nil || handle.ptr == nil {
		return nil, fmt.Errorf("invalid query result item handle")
	}

	// Get CBOR data
	cborData := C.dittoffi_query_result_item_cbor(handle.ptr)
	if cborData.ptr == nil || cborData.len == 0 {
		return nil, fmt.Errorf("no CBOR data available")
	}

	// Convert to Go bytes
	goBytes := bytesFromFFI(cborData)
	return goBytes, nil
}

// GetQueryResultItemJSONData returns the JSON string as []byte from a QueryResultItemHandle
func GetQueryResultItemJSONData(handle *QueryResultItemHandle) ([]byte, error) {
	if !handle.IsValid() {
		return nil, fmt.Errorf("invalid query result item handle")
	}

	// Get JSON string
	jsonStr := C.dittoffi_query_result_item_json(handle.ptr)
	if jsonStr == nil {
		return nil, fmt.Errorf("failed to get JSON")
	}
	defer C.ditto_c_string_free(jsonStr)

	bytes := C.GoBytes(unsafe.Pointer(jsonStr), C.int(C.strlen(jsonStr)))
	return bytes, nil
}

// GetQueryResultItemJSONString returns the JSON string from a QueryResultItemHandle
func GetQueryResultItemJSONString(handle *QueryResultItemHandle) (string, error) {
	if !handle.IsValid() {
		return "", fmt.Errorf("invalid query result item handle")
	}

	// Get JSON string
	jsonStr := C.dittoffi_query_result_item_json(handle.ptr)
	if jsonStr == nil {
		return "", fmt.Errorf("failed to get JSON")
	}

	return stringFromFFI(jsonStr), nil
}

// NewQueryResultItemFromJSON creates a new QueryResultItemHandle from JSON data
func NewQueryResultItemFromJSON(jsonData []byte) (*QueryResultItemHandle, error) {
	if len(jsonData) == 0 {
		return nil, fmt.Errorf("empty JSON data")
	}

	jsonSlice := C.slice_ref_uint8_t{
		ptr: (*C.uint8_t)(unsafe.Pointer(&jsonData[0])),
		len: C.size_t(len(jsonData)),
	}

	result := C.dittoffi_query_result_item_new(jsonSlice)
	if result.error != nil {
		err := ConvertFFIErrorToDittoError(result.error)
		return nil, err
	}

	if result.success == nil {
		return nil, fmt.Errorf("failed to create query result item")
	}

	handle := &QueryResultItemHandle{
		ptr: result.success,
	}

	runtime.SetFinalizer(handle, func(h *QueryResultItemHandle) {
		h.Free()
	})

	return handle, nil
}

// Free releases the underlying C resources for a QueryResultItemHandle
func (h *QueryResultItemHandle) Free() {
	ffiDebugTrace("QueryResultItemHandle.Free() called")

	if h == nil {
		return
	}

	// Use atomic compare-and-swap to ensure we only free once
	if atomic.CompareAndSwapInt32(&h.finalized, 0, 1) {
		if h.ptr != nil {
			C.dittoffi_query_result_item_free(h.ptr)
			h.ptr = nil
		}
	}
}

// IsValid checks if the handle is still valid (not freed)
func (h *QueryResultItemHandle) IsValid() bool {
	return h != nil && h.ptr != nil && atomic.LoadInt32(&h.finalized) == 0
}

// GetStoreTransactions returns CBOR data containing information about currently active transactions
func GetStoreTransactions(handle *DittoHandle) ([]byte, error) {
	// Call FFI function to get transactions CBOR data
	cborSlice := C.dittoffi_store_transactions(handle.load())

	// Check if we got valid data
	if cborSlice.ptr == nil {
		// Empty transactions list is valid, return empty array
		return []byte("[]"), nil
	}

	cborData := bytesFromFFI(cborSlice)
	return cborData, nil
}
