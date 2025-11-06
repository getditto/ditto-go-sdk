// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"

// Forward declarations for Go callbacks
extern void goPresenceCallback(void* context, slice_boxed_uint8_t data);
extern void goPresenceFree(void* context);
extern void goConnectionRequestHandler(void* context, dittoffi_connection_request_t* request);
extern void goConnectionHandlerRetain(void* context);
extern void goConnectionHandlerRelease(void* context);

// Connection request handler wrappers
static void connection_handler_on_connecting(Erased_t const* context, dittoffi_connection_request_t* request) {
    goConnectionRequestHandler((void*)context, request);
}

static Erased_t* connection_handler_retain(Erased_t const* context) {
    goConnectionHandlerRetain((void*)context);
    return (Erased_t*)context;
}

static void connection_handler_release(Erased_t* context) {
    goConnectionHandlerRelease((void*)context);
}

// Helper to create connection request handler vtable
static VirtualPtr__Erased_ptr_FfiConnectionRequestHandlerVTable_t create_connection_handler_vtable(void* context) {
    VirtualPtr__Erased_ptr_FfiConnectionRequestHandlerVTable_t handler;
    handler.ptr = (Erased_t*)context;
    handler.vtable.on_connecting = connection_handler_on_connecting;
    handler.vtable.retain_vptr = connection_handler_retain;
    handler.vtable.release_vptr = connection_handler_release;
    return handler;
}

// Helper to create presence observer callback struct
static BoxDynFnMut1_void_slice_boxed_uint8_t create_presence_observer_callback(void* context) {
	BoxDynFnMut1_void_slice_boxed_uint8_t cb;
	cb.env_ptr = context;
	cb.call = goPresenceCallback;
	cb.free = goPresenceFree;
	return cb;
}
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Presence callback management
var (
	// Thread-safe presence callback management
	presenceCallbacks      sync.Map      // map[uintptr]func([]byte)
	nextPresenceCallbackID atomic.Uint64 // atomic counter
)

// RegisterPresenceCallbackGo registers a Go presence callback and returns its ID
func RegisterPresenceCallbackGo(callback func([]byte)) uintptr {
	id := uintptr(nextPresenceCallbackID.Add(1))
	presenceCallbacks.Store(id, callback)
	return id
}

// UnregisterPresenceCallback unregisters a presence callback
func UnregisterPresenceCallback(id uintptr) {
	presenceCallbacks.Delete(id)
}

// GetPresenceCallback retrieves a presence callback by ID
func GetPresenceCallback(id uintptr) func([]byte) {
	if callback, exists := presenceCallbacks.Load(id); exists {
		if cb, ok := callback.(func([]byte)); ok {
			return cb
		}
	}
	return nil
}

// PresenceObserverHandle represents a presence observer
type PresenceObserverHandle struct {
	ptr        *C.dittoffi_presence_observer_t
	callbackID uintptr
}

// GetPresenceGraph returns the current presence graph as JSON
func GetPresenceGraph(handle *DittoHandle) ([]byte, error) {
	ffiDebugTrace("GetPresenceGraph called")

	result := C.dittoffi_presence_graph(handle.load())
	if result.ptr == nil {
		return nil, fmt.Errorf("failed to get presence graph")
	}

	data := bytesFromFFI(result)
	return data, nil
}

// SetPeerMetadataJSON sets the peer metadata from JSON
func SetPeerMetadataJSON(handle *DittoHandle, jsonData []byte) error {
	ffiDebugTrace("SetPeerMetadataJSON called")

	dataSlice := C.slice_ref_uint8_t{
		ptr: (*C.uint8_t)(C.CBytes(jsonData)),
		len: C.size_t(len(jsonData)),
	}
	defer C.free(unsafe.Pointer(dataSlice.ptr))

	result := C.dittoffi_presence_set_peer_metadata_json_throws(handle.load(), dataSlice)
	if result.error != nil {
		return errorFromFFIError(result.error)
	}

	return nil
}

// GetPeerMetadataJSON returns the peer metadata as JSON
func GetPeerMetadataJSON(handle *DittoHandle) ([]byte, error) {
	ffiDebugTrace("GetPeerMetadataJSON called")

	result := C.dittoffi_presence_peer_metadata_json(handle.load())
	if result.ptr == nil {
		return nil, fmt.Errorf("failed to get peer metadata")
	}

	data := bytesFromFFI(result)
	return data, nil
}

// RegisterPresenceObserver registers a callback for presence changes
func RegisterPresenceObserver(handle *DittoHandle, callback func([]byte)) (*PresenceObserverHandle, error) {
	ffiDebugTrace("RegisterPresenceObserver called")

	// Register the Go callback and get its ID
	callbackID := RegisterPresenceCallbackGo(callback)

	// Pass the callback ID directly as unsafe.Pointer
	// This works because uintptr can be safely cast to unsafe.Pointer for callback context
	callbackIDPtr := unsafe.Pointer(callbackID)

	cCallback := C.create_presence_observer_callback(callbackIDPtr)

	result := C.dittoffi_presence_register_observer_throws(handle.load(), cCallback)
	if result.error != nil {
		UnregisterPresenceCallback(callbackID)
		return nil, errorFromFFIError(result.error)
	}

	return &PresenceObserverHandle{
		ptr:        result.success,
		callbackID: callbackID,
	}, nil
}

// CancelPresenceObserver cancels a presence observer
func CancelPresenceObserver(handle *PresenceObserverHandle) {
	ffiDebugTrace("CancelPresenceObserver called")

	if handle != nil && handle.ptr != nil {
		C.dittoffi_presence_observer_cancel(handle.ptr)
		C.dittoffi_presence_observer_free(handle.ptr)
		handle.ptr = nil

		// Unregister the Go callback
		if handle.callbackID != 0 {
			UnregisterPresenceCallback(handle.callbackID)
			handle.callbackID = 0
		}
	}
}

//export goPresenceCallback
func goPresenceCallback(contextPtr unsafe.Pointer, data C.slice_boxed_uint8_t) {
	ffiDebugTrace("goPresenceCallback called")
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in presence callback: %v\n%s", r, debug.Stack())
		}
	}()

	defer C.ditto_c_bytes_free(data)

	id := uintptr(contextPtr)
	callback := GetPresenceCallback(id)
	if callback == nil {
		ffiDebugTrace("gosdk: ERROR goPresenceCallback unable to find Go callback")
		return
	}

	// Convert C data to Go bytes and call the Go callback
	goData := C.GoBytes(unsafe.Pointer(data.ptr), C.int(data.len))
	callback(goData)
}

//export goPresenceFree
func goPresenceFree(contextPtr unsafe.Pointer) {
	ffiDebugTrace("goPresenceFree called")

	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in presence free callback: %v\n%s", r, debug.Stack())
		}
	}()

	// Extract the callback ID and unregister it
	id := uintptr(contextPtr)
	UnregisterPresenceCallback(id)
}

// SetConnectionRequestHandler sets a handler for connection requests
func SetConnectionRequestHandler(ditto *DittoHandle, handlerID uintptr) {
	// Create the vtable for the handler
	vtable := C.create_connection_handler_vtable(unsafe.Pointer(handlerID))

	// Call the FFI function
	C.dittoffi_presence_set_connection_request_handler(ditto.load(), vtable)
}

// ConnectionRequestHandlerContext holds the handler for connection requests
type ConnectionRequestHandlerContext struct {
	handler ConnectionRequestHandler
	id      uintptr
}

// ConnectionRequestHandler is the interface for handling connection requests
type ConnectionRequestHandler interface {
	HandleConnectionRequest(request *ConnectionRequestWrapper) ConnectionRequestAuthorization
}

// ConnectionRequestWrapper wraps a C connection request
type ConnectionRequestWrapper struct {
	ptr *C.dittoffi_connection_request_t
}

// ConnectionRequestAuthorization represents the authorization decision
type ConnectionRequestAuthorization int

const (
	ConnectionRequestAuthorizationDeny  ConnectionRequestAuthorization = C.DITTOFFI_CONNECTION_REQUEST_AUTHORIZATION_DENY
	ConnectionRequestAuthorizationAllow ConnectionRequestAuthorization = C.DITTOFFI_CONNECTION_REQUEST_AUTHORIZATION_ALLOW
)

var (
	// Thread-safe connection handler management
	connectionHandlerContexts sync.Map      // map[uintptr]*ConnectionRequestHandlerContext
	nextConnectionHandlerID   atomic.Uint64 // atomic counter
)

// RegisterConnectionRequestHandler registers a connection request handler
func RegisterConnectionRequestHandler(handler ConnectionRequestHandler) uintptr {
	id := uintptr(nextConnectionHandlerID.Add(1))

	ctx := &ConnectionRequestHandlerContext{
		handler: handler,
		id:      id,
	}

	connectionHandlerContexts.Store(id, ctx)
	return id
}

// UnregisterConnectionRequestHandler removes a connection request handler
func UnregisterConnectionRequestHandler(id uintptr) {
	connectionHandlerContexts.Delete(id)
}

// GetConnectionRequestHandler retrieves a connection request handler context
func GetConnectionRequestHandler(id uintptr) *ConnectionRequestHandlerContext {
	if ctx, exists := connectionHandlerContexts.Load(id); exists {
		if connCtx, ok := ctx.(*ConnectionRequestHandlerContext); ok {
			return connCtx
		}
	}
	return nil
}

//export goConnectionRequestHandler
func goConnectionRequestHandler(contextPtr unsafe.Pointer, requestPtr *C.dittoffi_connection_request_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in connection request handler: %v\n%s", r, debug.Stack())
			// Deny by default on panic for security
			if requestPtr != nil {
				C.dittoffi_connection_request_authorize(requestPtr, C.DITTOFFI_CONNECTION_REQUEST_AUTHORIZATION_DENY)
			}
		}
	}()

	if contextPtr == nil || requestPtr == nil {
		// Deny by default if no handler or request
		C.dittoffi_connection_request_authorize(requestPtr, C.DITTOFFI_CONNECTION_REQUEST_AUTHORIZATION_DENY)
		return
	}

	id := uintptr(contextPtr)
	ctx := GetConnectionRequestHandler(id)
	if ctx == nil || ctx.handler == nil {
		// Deny if handler not found
		C.dittoffi_connection_request_authorize(requestPtr, C.DITTOFFI_CONNECTION_REQUEST_AUTHORIZATION_DENY)
		return
	}

	// Wrap the request
	request := &ConnectionRequestWrapper{ptr: requestPtr}

	// Call the Go handler
	auth := ctx.handler.HandleConnectionRequest(request)

	// Convert authorization to FFI type
	ffiAuth := C.dittoffi_connection_request_authorization_t(auth)

	// Authorize the request
	C.dittoffi_connection_request_authorize(requestPtr, ffiAuth)
}

//export goConnectionHandlerRetain
func goConnectionHandlerRetain(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in connection handler retain: %v\n%s", r, debug.Stack())
		}
	}()

	// Reference counting is handled by Go's GC
	// We don't need to do anything here
}

//export goConnectionHandlerRelease
func goConnectionHandlerRelease(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in connection handler release: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	// Unregister the handler when released
	id := uintptr(contextPtr)
	UnregisterConnectionRequestHandler(id)
}

// Methods for ConnectionRequestWrapper to extract information

// GetPeerKey returns the peer key string
func (r *ConnectionRequestWrapper) GetPeerKey() string {
	if r.ptr == nil {
		return ""
	}
	cStr := C.dittoffi_connection_request_peer_key_string(r.ptr)
	if cStr == nil {
		return ""
	}
	defer C.ditto_c_string_free(cStr)
	return C.GoString(cStr)
}

// GetPeerMetadata returns the peer metadata as a map
func (r *ConnectionRequestWrapper) GetPeerMetadata() map[string]any {
	if r.ptr == nil {
		return nil
	}

	slice := C.dittoffi_connection_request_peer_metadata_json(r.ptr)
	if slice.ptr == nil || slice.len == 0 {
		return nil
	}

	jsonData := C.GoBytes(unsafe.Pointer(slice.ptr), C.int(slice.len))

	var metadata map[string]any
	if err := json.Unmarshal(jsonData, &metadata); err != nil {
		return nil
	}

	return metadata
}

// GetIdentityServiceMetadata returns the identity service metadata as a map
func (r *ConnectionRequestWrapper) GetIdentityServiceMetadata() map[string]any {
	if r.ptr == nil {
		return nil
	}

	slice := C.dittoffi_connection_request_identity_service_metadata_json(r.ptr)
	if slice.ptr == nil || slice.len == 0 {
		return nil
	}

	jsonData := C.GoBytes(unsafe.Pointer(slice.ptr), C.int(slice.len))

	var metadata map[string]any
	if err := json.Unmarshal(jsonData, &metadata); err != nil {
		return nil
	}

	return metadata
}

// GetConnectionType returns the connection type
func (r *ConnectionRequestWrapper) GetConnectionType() string {
	if r.ptr == nil {
		return "Unknown"
	}

	connType := C.dittoffi_connection_request_connection_type(r.ptr)

	// Convert the connection type enum to string
	switch connType {
	case C.DITTOFFI_CONNECTION_TYPE_BLUETOOTH:
		return "Bluetooth"
	case C.DITTOFFI_CONNECTION_TYPE_ACCESS_POINT:
		return "AccessPoint"
	case C.DITTOFFI_CONNECTION_TYPE_P2_P_WI_FI:
		return "P2PWiFi"
	case C.DITTOFFI_CONNECTION_TYPE_WEB_SOCKET:
		return "WebSocket"
	default:
		return "Unknown"
	}
}
