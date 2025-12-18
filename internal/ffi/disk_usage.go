// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"

// Forward declarations for Go callbacks
extern void goDiskUsageCallback(void* context, slice_ref_uint8_t data);
extern void goDiskUsageRetain(void* context);
extern void goDiskUsageRelease(void* context);

// Helper to register disk usage callback
static DiskUsageObserver_t *register_disk_usage_callback(
    CDitto_t const * ditto,
    FsComponent_t component,
    uintptr_t ctx
) {
	return ditto_register_disk_usage_callback(
		ditto,
		component,
		(void *)ctx,
		goDiskUsageRetain,
		goDiskUsageRelease,
		goDiskUsageCallback
	);
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

// DiskUsageHandle represents a disk usage observer handle
type DiskUsageHandle struct {
	HandleCleaner[diskUsageHandleFreer]
	callbackID uintptr
}

type diskUsageHandleFreer struct {
	ptr *C.DiskUsageObserver_t
}

func (d diskUsageHandleFreer) free() {
	C.ditto_release_disk_usage_callback(d.ptr)
}

// DiskUsageContext holds the Go callback and context for disk usage
type DiskUsageContext struct {
	callback func([]byte)
	id       uintptr
}

var (
	// Thread-safe disk usage management
	diskUsageContexts sync.Map      // map[uintptr]*DiskUsageContext
	nextDiskUsageID   atomic.Uint64 // atomic counter
)

// RegisterDiskUsageCallback registers a callback for disk usage changes
func RegisterDiskUsageCallback(handle *DittoHandle, component FsComponent, callback func([]byte)) (*DiskUsageHandle, error) {
	// Register the Go callback and get its ID
	callbackID := registerDiskUsageCallbackGo(callback)

	// Pass the callback ID directly as unsafe.Pointer
	// This works because uintptr can be safely cast to unsafe.Pointer for callback context
	callbackIDPtr := C.uintptr_t(callbackID)

	observerPtr := C.register_disk_usage_callback(
		handle.load(),
		C.FsComponent_t(component),
		callbackIDPtr,
	)

	if observerPtr == nil {
		// Clean up the registered callback
		unregisterDiskUsageCallbackGo(callbackID)
		return nil, fmt.Errorf("failed to register disk usage callback")
	}

	d := &DiskUsageHandle{}
	d.Initialize(diskUsageHandleFreer{ptr: observerPtr})
	d.callbackID = callbackID
	return d, nil
}

// ReleaseDiskUsageCallback releases a disk usage observer
func ReleaseDiskUsageCallback(handle *DiskUsageHandle) {
	handle.Free()
	// Unregister the Go callback
	if handle.callbackID != 0 {
		unregisterDiskUsageCallbackGo(handle.callbackID)
		handle.callbackID = 0
	}
}

// registerDiskUsageCallbackGo registers a Go callback for disk usage and returns its ID
func registerDiskUsageCallbackGo(callback func([]byte)) uintptr {
	id := uintptr(nextDiskUsageID.Add(1))

	ctx := &DiskUsageContext{
		callback: callback,
		id:       id,
	}

	diskUsageContexts.Store(id, ctx)
	return id
}

// unregisterDiskUsageCallbackGo removes a disk usage callback from the registry
func unregisterDiskUsageCallbackGo(id uintptr) {
	diskUsageContexts.Delete(id)
}

// GetDiskUsageContext retrieves a disk usage callback context
func GetDiskUsageContext(id uintptr) *DiskUsageContext {
	if ctx, exists := diskUsageContexts.Load(id); exists {
		if diskCtx, ok := ctx.(*DiskUsageContext); ok {
			return diskCtx
		}
	}
	return nil
}

//export goDiskUsageCallback
func goDiskUsageCallback(contextPtr unsafe.Pointer, data C.slice_ref_uint8_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in disk usage callback: %v\n%s", r, debug.Stack())
		}
	}()

	// Extract the callback ID from the context pointer
	id := uintptr(contextPtr)

	ctx := GetDiskUsageContext(id)
	if ctx == nil || ctx.callback == nil {
		return
	}

	// Convert C slice to Go byte slice
	goData := C.GoBytes(unsafe.Pointer(data.ptr), C.int(data.len))

	// Call the Go callback
	ctx.callback(goData)
}

//export goDiskUsageRetain
func goDiskUsageRetain(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in disk usage retain callback: %v\n%s", r, debug.Stack())
		}
	}()

	// No-op for now - Go manages memory
}

//export goDiskUsageRelease
func goDiskUsageRelease(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in disk usage release callback: %v\n%s", r, debug.Stack())
		}
	}()

	// Extract the callback ID and unregister it
	id := uintptr(contextPtr)
	unregisterDiskUsageCallbackGo(id)
}

// FsComponent represents filesystem components for disk usage
type FsComponent uint8

const (
	FsComponentRoot        FsComponent = C.FS_COMPONENT_ROOT
	FsComponentStore       FsComponent = C.FS_COMPONENT_STORE
	FsComponentAuth        FsComponent = C.FS_COMPONENT_AUTH
	FsComponentReplication FsComponent = C.FS_COMPONENT_REPLICATION
	FsComponentAttachment  FsComponent = C.FS_COMPONENT_ATTACHMENT
)

// GetDiskUsage returns disk usage information for the specified component
func GetDiskUsage(handle *DittoHandle, component FsComponent) ([]byte, error) {
	result := C.ditto_disk_usage(handle.load(), C.FsComponent_t(component))
	if result.ptr == nil {
		return nil, fmt.Errorf("failed to get disk usage")
	}

	cborData := bytesFromFFI(result)
	return cborData, nil
}
