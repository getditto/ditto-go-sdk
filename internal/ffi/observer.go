package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"

// Forward declarations for Go callbacks
extern void goStoreObserverCallback(void* context, dittoffi_query_result_t* result, ArcDynFn0_void_t signal_next);
extern void goStoreObserverFree(void* context);

// Helper to create the callback struct
static BoxDynFnMut2_void_dittoffi_query_result_ptr_ArcDynFn0_void_t create_store_observer_callback(void* context) {
    BoxDynFnMut2_void_dittoffi_query_result_ptr_ArcDynFn0_void_t cb;
    cb.env_ptr = context;
    cb.call = goStoreObserverCallback;
    cb.free = goStoreObserverFree;
    return cb;
}

// Helper to call signal_next function
static void call_signal_next(ArcDynFn0_void_t signal_next) {
    if (signal_next.call != NULL) {
        signal_next.call(signal_next.env_ptr);
    }
}
*/
import "C"
import (
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
)

// StoreObserverCallbackContext holds the Go callback and context with safe memory management.
// This struct ensures callbacks remain valid during FFI calls through:
// - runtime.Pinner to prevent GC from moving/collecting the context
// - Atomic active flag to prevent use-after-free
// - Finalizer for automatic cleanup if not explicitly unregistered
type StoreObserverCallbackContext struct {
	pinner                 *runtime.Pinner
	callback               func(*QueryResultHandle)
	callbackWithSignalNext func(*QueryResultHandle, func())
	id                     uintptr
	active                 atomic.Bool // atomic flag for active state
	autoSignal             bool        // If true, automatically call signal_next after callback
}

var (
	// Thread-safe observer management using sync.Map for lock-free reads.
	// This eliminates mutex contention in the common case (callback invocation).
	storeObserverContexts sync.Map // map[uintptr]*StoreObserverCallbackContext

	// Atomic counter ensures unique IDs without race conditions
	nextStoreObserverID atomic.Uint64 // atomic counter
)

// RegisterStoreObserverCallback registers a Go callback in storeObserverContexts and returns its ID.
// Safety features:
// - Atomic ID generation prevents ID conflicts
// - runtime.Pinner prevents GC interference during FFI calls
// - sync.Map storage allows concurrent access without locks
// - Finalizer ensures cleanup even if UnregisterObserverCallback is not called
func RegisterStoreObserverCallback(callback func(*QueryResultHandle)) uintptr {
	// Atomic increment ensures unique IDs even with concurrent registrations
	id := uintptr(nextStoreObserverID.Add(1))

	ctx := &StoreObserverCallbackContext{
		pinner:     &runtime.Pinner{},
		callback:   callback,
		id:         id,
		autoSignal: true, // Simple callbacks auto-signal
	}
	ctx.active.Store(true)

	// Pin the context to prevent GC from moving or collecting it
	// This is CRITICAL for FFI safety - the C code holds a pointer to this context
	ctx.pinner.Pin(ctx)

	// Store in thread-safe map for lock-free retrieval during callbacks
	storeObserverContexts.Store(id, ctx)

	// Set finalizer for cleanup - acts as safety net if user forgets to unregister
	runtime.SetFinalizer(ctx, (*StoreObserverCallbackContext).finalize)

	return id
}

// RegisterStoreObserverCallbackWithSignalNext registers a callback that controls signal_next
func RegisterStoreObserverCallbackWithSignalNext(callback func(*QueryResultHandle, func())) uintptr {
	// Atomic increment ensures unique IDs even with concurrent registrations
	id := uintptr(nextStoreObserverID.Add(1))

	ctx := &StoreObserverCallbackContext{
		pinner:                 &runtime.Pinner{},
		callbackWithSignalNext: callback,
		id:                     id,
		autoSignal:             false, // Callback controls signal timing
	}
	ctx.active.Store(true)

	// Pin the context to prevent GC from moving or collecting it
	// This is CRITICAL for FFI safety - the C code holds a pointer to this context
	ctx.pinner.Pin(ctx)

	// Store in thread-safe map for lock-free retrieval during callbacks
	storeObserverContexts.Store(id, ctx)

	// Set finalizer for cleanup - acts as safety net if user forgets to unregister
	runtime.SetFinalizer(ctx, (*StoreObserverCallbackContext).finalize)

	return id
}

// UnregisterStoreObserverCallback removes a callback from the registry.
// Safety features:
// - LoadAndDelete is atomic, preventing race conditions
// - Proper cleanup unpins memory and clears resources
// - Safe to call multiple times (idempotent)
func UnregisterStoreObserverCallback(id uintptr) {
	// Atomic load and delete prevents races
	if ctx, loaded := storeObserverContexts.LoadAndDelete(id); loaded {
		if safeCtx, ok := ctx.(*StoreObserverCallbackContext); ok {
			safeCtx.cleanup()
		}
	}
}

// GetStoreObserverContext retrieves a store observer callback context.
// Safety features:
// - Returns nil if context is inactive (prevents use-after-free)
// - Atomic active check ensures context validity
// - Thread-safe retrieval through sync.Map
func GetStoreObserverContext(id uintptr) *StoreObserverCallbackContext {
	if ctx, exists := storeObserverContexts.Load(id); exists {
		if safeCtx, ok := ctx.(*StoreObserverCallbackContext); ok {
			// Only return if still active to prevent use-after-cleanup
			if safeCtx.active.Load() {
				return safeCtx
			}
		}
	}
	return nil
}

// cleanup safely cleans up the context
func (ctx *StoreObserverCallbackContext) cleanup() {
	if ctx.active.CompareAndSwap(true, false) {
		if ctx.pinner != nil {
			ctx.pinner.Unpin()
		}
		runtime.SetFinalizer(ctx, nil)
	}
}

// finalize is called by the garbage collector
func (ctx *StoreObserverCallbackContext) finalize() {
	ctx.cleanup()
}

// goStoreObserverCallback is called from C when an observer receives new data.
// This is an FFI boundary function that MUST NOT panic to avoid crashing the process.
// Safety features:
// - Panic recovery prevents process crashes
// - Nil checks prevent segfaults
// - Safe context retrieval prevents use-after-free
// - Always signals next to prevent deadlocks
//
//export goStoreObserverCallback
func goStoreObserverCallback(contextPtr unsafe.Pointer, resultPtr *C.dittoffi_query_result_t, signalNext C.ArcDynFn0_void_t) {
	// CRITICAL: Panic recovery at FFI boundary
	// Any panic here would crash the entire process since we're called from C
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in observer callback: %v\n%s", r, debug.Stack())
			// Still try to signal next to prevent deadlock in C code
			if signalNext.call != nil {
				C.call_signal_next(signalNext)
			}
		}
	}()

	if contextPtr == nil {
		return
	}

	// Extract the callback ID directly from the context pointer
	// This is safe because we control the pointer value passed from RegisterStoreObserverCallback
	id := uintptr(contextPtr)

	// Safe retrieval with active check to prevent use-after-free
	ctx := GetStoreObserverContext(id)
	if ctx == nil || (ctx.callback == nil && ctx.callbackWithSignalNext == nil) {
		return
	}

	// Create a QueryResultHandle from the C pointer
	handle := &QueryResultHandle{ptr: resultPtr}

	// Call the appropriate callback based on whether we have signal control
	if ctx.callbackWithSignalNext != nil {
		// Create a Go function that calls the C signal_next
		signalFunc := func() {
			if signalNext.call != nil {
				C.call_signal_next(signalNext)
			}
		}
		// Pass both the result and the signal function to the callback
		ctx.callbackWithSignalNext(handle, signalFunc)
	} else if ctx.callback != nil {
		// Call the simple callback
		ctx.callback(handle)
		// Auto-signal if configured (for simple observers)
		if ctx.autoSignal && signalNext.call != nil {
			C.call_signal_next(signalNext)
		}
	}
}

// goStoreObserverFree is called from C when an observer is being destroyed.
// This ensures proper cleanup of Go resources associated with the observer.
// Safety features:
// - Panic recovery prevents crashes during cleanup
// - Safe unregistration handles concurrent access
//
//export goStoreObserverFree
func goStoreObserverFree(contextPtr unsafe.Pointer) {
	// CRITICAL: Panic recovery during cleanup
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in observer free callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	// Extract the callback ID and unregister it
	id := uintptr(contextPtr)
	UnregisterStoreObserverCallback(id)
}

// CreateStoreObserverCallback creates a C callback struct for an observer
func CreateStoreObserverCallback(callbackID uintptr) C.BoxDynFnMut2_void_dittoffi_query_result_ptr_ArcDynFn0_void_t {
	// Pass the callback ID directly as unsafe.Pointer
	// This works because uintptr can be safely cast to unsafe.Pointer for callback context
	return C.create_store_observer_callback(unsafe.Pointer(callbackID))
}

// StoreRegisterObserverThrows registers a store observer
func StoreRegisterObserverThrows(ditto *DittoHandle, query string, args map[string]any, callback func(*QueryResultHandle)) (*StoreObserverHandle, error) {
	ffiDebugTrace("StoreRegisterObserverThrows called")

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

	// Register the callback and get its ID
	callbackID := RegisterStoreObserverCallback(callback)

	// Create the C callback struct
	cCallback := CreateStoreObserverCallback(callbackID)

	// Register the observer
	result := C.dittoffi_store_register_observer_throws(
		ditto.load(),
		cQuery,
		argsSlice,
		cCallback,
	)

	// Check for error
	if result.error != nil {
		UnregisterStoreObserverCallback(callbackID)
		return nil, errorFromFFIError(result.error)
	}

	if result.success == nil {
		UnregisterStoreObserverCallback(callbackID)
		return nil, fmt.Errorf("observer registration returned nil")
	}

	return &StoreObserverHandle{
		ptr:        result.success,
		callbackID: callbackID,
	}, nil
}

// StoreRegisterObserverWithSignalNextThrows registers a store observer with signal control
func StoreRegisterObserverWithSignalNextThrows(ditto *DittoHandle, query string, args map[string]any, callback func(*QueryResultHandle, func())) (*StoreObserverHandle, error) {
	ffiDebugTrace("StoreRegisterObserverWithSignalNextThrows called")

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

	// Register the callback with signal control and get its ID
	callbackID := RegisterStoreObserverCallbackWithSignalNext(callback)

	// Create the C callback struct
	cCallback := CreateStoreObserverCallback(callbackID)

	// Register the observer (same FFI function, different callback behavior)
	result := C.dittoffi_store_register_observer_throws(
		ditto.load(),
		cQuery,
		argsSlice,
		cCallback,
	)

	// Check for error
	if result.error != nil {
		UnregisterStoreObserverCallback(callbackID)
		return nil, errorFromFFIError(result.error)
	}

	if result.success == nil {
		UnregisterStoreObserverCallback(callbackID)
		return nil, fmt.Errorf("observer registration returned nil")
	}

	return &StoreObserverHandle{
		ptr:        result.success,
		callbackID: callbackID,
	}, nil
}

// CancelStoreObserver cancels a store observer
func CancelStoreObserver(handle *StoreObserverHandle) {
	ffiDebugTrace("CancelStoreObserver called")
	if handle != nil && handle.ptr != nil {
		C.dittoffi_store_observer_cancel(handle.ptr)
	}
}

// FreeStoreObserver frees an observer handle
func FreeStoreObserver(handle *StoreObserverHandle) {
	ffiDebugTrace("FreeStoreObserver called")
	if handle != nil && handle.ptr != nil {
		C.dittoffi_store_observer_free(handle.ptr)
		handle.ptr = nil
	}
}

// StoreObservers returns all currently active store observers
func StoreObservers(ditto *DittoHandle) ([]*StoreObserverHandle, error) {
	ffiDebugTrace("StoreObservers called")

	// Call FFI function to get observers
	vec := C.dittoffi_store_observers(ditto.load())
	defer C.dittoffi_store_observers_free_sparse(vec)

	// Convert C array to Go slice
	observerCount := int(vec.len)
	if observerCount == 0 {
		return []*StoreObserverHandle{}, nil
	}

	observers := make([]*StoreObserverHandle, observerCount)

	// Cast the C array pointer to a slice we can index
	cObservers := unsafe.Slice(vec.ptr, observerCount)

	for i := 0; i < observerCount; i++ {
		if cObservers[i] != nil {
			observers[i] = &StoreObserverHandle{
				ptr: cObservers[i],
				// Note: We don't have the callback ID for these observers
				// since they were registered elsewhere
				callbackID: 0,
			}
		}
	}

	return observers, nil
}
