// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

// See the package comment in bindings.go for notes about callback safety

/*
#include <stdlib.h>
#include "dittoffi.h"

// Forward declarations for Go callbacks
extern void goAttachmentOnComplete(void* context, AttachmentHandle_t* handle);
extern void goAttachmentOnProgress(void* context, uint64_t downloaded, uint64_t total);
extern void goAttachmentOnDeleted(void* context);
extern void goAttachmentRetain(void* context);
extern void goAttachmentRelease(void* context);

// Helper to resolve attachment and register callbacks
static CancelTokenResult_t resolve_attachment(
    CDitto_t const * ditto,
    slice_ref_uint8_t id,
    uintptr_t ctx
) {
	return ditto_resolve_attachment(
		ditto,
		id,
		(void *)ctx,
		goAttachmentRetain,
		goAttachmentRelease,
		goAttachmentOnComplete,
		goAttachmentOnProgress,
		goAttachmentOnDeleted
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

// ResolveAttachment starts fetching an attachment
func ResolveAttachment(ditto *DittoHandle, attachmentID []byte, callbackID uintptr) (*CancelTokenResult, error) {
	if len(attachmentID) == 0 {
		return nil, fmt.Errorf("attachment ID cannot be empty")
	}

	// Convert Go slice to C slice
	idSlice := C.slice_ref_uint8_t{
		ptr: (*C.uint8_t)(unsafe.Pointer(&attachmentID[0])),
		len: C.size_t(len(attachmentID)),
	}

	var result C.CancelTokenResult_t
	result = C.resolve_attachment(
		ditto.load(),
		idSlice,
		C.uintptr_t(callbackID), // ctx
	)

	return &CancelTokenResult{
		StatusCode:  int(result.status_code),
		CancelToken: uintptr(result.cancel_token),
	}, nil
}

// AttachmentCallbackContext holds the callbacks for attachment fetching
type AttachmentCallbackContext struct {
	onComplete func(*AttachmentHandle)
	onProgress func(downloaded, total uint64)
	onDeleted  func()
	id         uintptr
}

var (
	// Thread-safe attachment callback management
	attachmentContexts sync.Map      // map[uintptr]*AttachmentCallbackContext
	nextAttachmentID   atomic.Uint64 // atomic counter
)

// RegisterAttachmentCallbacks registers attachment fetch callbacks and returns an ID
func RegisterAttachmentCallbacks(onComplete func(*AttachmentHandle), onProgress func(uint64, uint64), onDeleted func()) uintptr {
	id := uintptr(nextAttachmentID.Add(1))

	ctx := &AttachmentCallbackContext{
		onComplete: onComplete,
		onProgress: onProgress,
		onDeleted:  onDeleted,
		id:         id,
	}

	attachmentContexts.Store(id, ctx)
	return id
}

// UnregisterAttachmentCallbacks removes attachment callbacks
func UnregisterAttachmentCallbacks(id uintptr) {
	attachmentContexts.Delete(id)
}

// GetAttachmentContext retrieves attachment callback context
func GetAttachmentContext(id uintptr) *AttachmentCallbackContext {
	if ctx, exists := attachmentContexts.Load(id); exists {
		if attachCtx, ok := ctx.(*AttachmentCallbackContext); ok {
			return attachCtx
		}
	}
	return nil
}

//export goAttachmentOnComplete
func goAttachmentOnComplete(contextPtr unsafe.Pointer, handlePtr *C.AttachmentHandle_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in attachment complete callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)
	ctx := GetAttachmentContext(id)
	if ctx == nil || ctx.onComplete == nil {
		return
	}

	// Create an AttachmentHandle from the C pointer
	handle := &AttachmentHandle{}
	handle.Initialize(attachmentHandleFreer{ptr: handlePtr})
	ctx.onComplete(handle)
}

//export goAttachmentOnProgress
func goAttachmentOnProgress(contextPtr unsafe.Pointer, downloaded, total C.uint64_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in attachment progress callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)
	ctx := GetAttachmentContext(id)
	if ctx == nil || ctx.onProgress == nil {
		return
	}

	ctx.onProgress(uint64(downloaded), uint64(total))
}

//export goAttachmentOnDeleted
func goAttachmentOnDeleted(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in attachment deleted callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)
	ctx := GetAttachmentContext(id)
	if ctx == nil || ctx.onDeleted == nil {
		return
	}

	ctx.onDeleted()
}

//export goAttachmentRetain
func goAttachmentRetain(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in attachment retain callback: %v\n%s", r, debug.Stack())
		}
	}()

	// Nothing to do - we manage the context lifetime ourselves
}

//export goAttachmentRelease
func goAttachmentRelease(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in attachment release callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	// Unregister the callbacks when released
	id := uintptr(contextPtr)
	UnregisterAttachmentCallbacks(id)
}

// AttachmentHandle represents an attachment
type AttachmentHandle struct {
	HandleCleaner[attachmentHandleFreer]
}

type attachmentHandleFreer struct {
	ptr *C.AttachmentHandle_t
}

func (a attachmentHandleFreer) free() {
	C.ditto_free_attachment_handle(a.ptr)
}

func (h *AttachmentHandle) IsValid() bool {
	return h != nil && h.inner.ptr != nil
}

// Attachment represents attachment data
type Attachment struct {
	ID     []byte
	Length uint64
	Handle *AttachmentHandle
}

// NewAttachmentFromBytes creates an attachment from bytes
func NewAttachmentFromBytes(ditto *DittoHandle, data []byte) (*Attachment, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("attachment data cannot be empty")
	}

	var cattachment C.CAttachment_t
	dataSlice := C.slice_ref_uint8_t{
		ptr: (*C.uint8_t)(unsafe.Pointer(&data[0])),
		len: C.size_t(len(data)),
	}

	result := C.ditto_new_attachment_from_bytes(ditto.load(), dataSlice, &cattachment)
	if result != 0 {
		return nil, fmt.Errorf("failed to create attachment from bytes: error code %d", result)
	}

	idBytes := bytesFromFFI(cattachment.id)

	handle := &AttachmentHandle{}
	handle.Initialize(attachmentHandleFreer{ptr: cattachment.handle})
	attachment := &Attachment{
		ID:     idBytes,
		Length: uint64(cattachment.len),
		Handle: handle,
	}

	return attachment, nil
}

// NewAttachmentFromFile creates an attachment from a file
func NewAttachmentFromFile(ditto *DittoHandle, path string, copyFile bool) (*Attachment, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	fileOp := C.AttachmentFileOperation_t(C.ATTACHMENT_FILE_OPERATION_COPY)
	if !copyFile {
		fileOp = C.AttachmentFileOperation_t(C.ATTACHMENT_FILE_OPERATION_MOVE)
	}

	var cattachment C.CAttachment_t
	result := C.ditto_new_attachment_from_file(ditto.load(), cPath, fileOp, &cattachment)

	switch result {
	case 0:
		// Success
	case 2:
		return nil, fmt.Errorf("file not found: %s", path)
	case 3:
		return nil, fmt.Errorf("permission denied: %s", path)
	default:
		return nil, fmt.Errorf("failed to create attachment from file: error code %d", result)
	}

	idBytes := bytesFromFFI(cattachment.id)

	handle := &AttachmentHandle{}
	handle.Initialize(attachmentHandleFreer{ptr: cattachment.handle})
	attachment := &Attachment{
		ID:     idBytes,
		Length: uint64(cattachment.len),
		Handle: handle,
	}

	return attachment, nil
}

// GetAttachmentStatus gets the status of an attachment
func GetAttachmentStatus(ditto *DittoHandle, id []byte) (*AttachmentHandle, error) {
	idSlice := C.slice_ref_uint8_t{
		ptr: (*C.uint8_t)(unsafe.Pointer(&id[0])),
		len: C.size_t(len(id)),
	}

	result := C.ditto_get_attachment_status(ditto.load(), idSlice)
	if result.status_code != 0 {
		return nil, fmt.Errorf("failed to get attachment status: error code %d", result.status_code)
	}

	if result.handle == nil {
		return nil, fmt.Errorf("attachment handle is nil")
	}
	handle := &AttachmentHandle{}
	handle.Initialize(attachmentHandleFreer{ptr: result.handle})
	return handle, nil
}

// AttachmentFetchCallback represents callback functions for attachment fetching
type AttachmentFetchCallback struct {
	OnComplete func(handle *AttachmentHandle)
	OnProgress func(downloaded, total uint64)
	OnDeleted  func()
}

// CancelTokenResult represents the result of resolve attachment operation
type CancelTokenResult struct {
	StatusCode  int
	CancelToken uintptr
}

// GetCompleteAttachmentPath gets the path of a completed attachment
func GetCompleteAttachmentPath(ditto *DittoHandle, handle *AttachmentHandle) (string, error) {
	if !handle.IsValid() {
		return "", fmt.Errorf("invalid attachment handle")
	}

	// Call FFI function to get path
	cStr := C.ditto_get_complete_attachment_path(ditto.load(), handle.inner.ptr)
	if cStr == nil {
		return "", fmt.Errorf("attachment path not available")
	}

	// Use helper function to safely convert and free the string
	return stringFromFFI(cStr), nil
}

// GetCompleteAttachmentData gets the data of a completed attachment
func GetCompleteAttachmentData(ditto *DittoHandle, handle *AttachmentHandle) ([]byte, error) {
	if !handle.IsValid() {
		return nil, fmt.Errorf("invalid attachment handle")
	}

	// Call FFI function to get data
	result := C.ditto_get_complete_attachment_data(ditto.load(), handle.inner.ptr)

	// Check for errors
	if result.status != 0 {
		return nil, fmt.Errorf("failed to get attachment data: status %d", result.status)
	}

	// Check if we got valid data
	if result.data.ptr == nil || result.data.len == 0 {
		return nil, fmt.Errorf("attachment data not available")
	}

	// Copy the data to Go slice
	data := bytesFromFFI(result.data)

	return data, nil
}
