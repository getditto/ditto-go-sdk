// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"

// Forward declarations for Go callbacks
extern void goTransactionBeginCallback(void* context, dittoffi_result_dittoffi_transaction_ptr_t result);
extern void goTransactionBeginFree(void* context);
extern void goTransactionExecuteCallback(void* context, dittoffi_result_dittoffi_query_result_ptr_t result);
extern void goTransactionExecuteFree(void* context);
extern void goTransactionCompleteCallback(void* context, dittoffi_result_dittoffi_transaction_completion_action_t result);
extern void goTransactionCompleteFree(void* context);

// Helper to create transaction begin callback struct
static continuation_dittoffi_result_dittoffi_transaction_ptr_t create_transaction_begin_callback(uintptr_t context) {
    continuation_dittoffi_result_dittoffi_transaction_ptr_t cb;
    cb.env_ptr = (void *)context;
    cb.call = goTransactionBeginCallback;
    cb.free = goTransactionBeginFree;
    return cb;
}

// Helper to create transaction execute callback struct
static continuation_dittoffi_result_dittoffi_query_result_ptr_t create_transaction_execute_callback(uintptr_t context) {
    continuation_dittoffi_result_dittoffi_query_result_ptr_t cb;
    cb.env_ptr = (void *)context;
    cb.call = goTransactionExecuteCallback;
    cb.free = goTransactionExecuteFree;
    return cb;
}

// Helper to create transaction complete callback struct
static continuation_dittoffi_result_dittoffi_transaction_completion_action_t create_transaction_complete_callback(uintptr_t context) {
    continuation_dittoffi_result_dittoffi_transaction_completion_action_t cb;
    cb.env_ptr = (void *)context;
    cb.call = goTransactionCompleteCallback;
    cb.free = goTransactionCompleteFree;
    return cb;
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

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
)

// Transaction callback management

// TransactionBeginContext holds the context for a transaction begin callback
type TransactionBeginContext struct {
	callback func(*TransactionHandle, error)
	id       uintptr
}

// TransactionExecuteContext holds the context for a transaction execute callback
type TransactionExecuteContext struct {
	callback func(*QueryResultHandle, error)
	id       uintptr
}

// TransactionCompleteContext holds the context for a transaction complete callback
type TransactionCompleteContext struct {
	callback func(int, error)
	id       uintptr
}

var (
	// Thread-safe transaction callback management
	transactionBeginContexts sync.Map      // map[uintptr]*TransactionBeginContext
	nextTransactionBeginID   atomic.Uint64 // atomic counter

	transactionExecuteContexts sync.Map      // map[uintptr]*TransactionExecuteContext
	nextTransactionExecuteID   atomic.Uint64 // atomic counter

	transactionCompleteContexts sync.Map      // map[uintptr]*TransactionCompleteContext
	nextTransactionCompleteID   atomic.Uint64 // atomic counter
)

// RegisterTransactionBeginCallback registers a transaction begin callback
func RegisterTransactionBeginCallback(callback func(*TransactionHandle, error)) uintptr {
	id := uintptr(nextTransactionBeginID.Add(1))

	ctx := &TransactionBeginContext{
		callback: callback,
		id:       id,
	}

	transactionBeginContexts.Store(id, ctx)
	return id
}

// UnregisterTransactionBeginCallback removes a transaction begin callback
func UnregisterTransactionBeginCallback(id uintptr) {
	transactionBeginContexts.Delete(id)
}

// RegisterTransactionExecuteCallback registers a transaction execute callback
func RegisterTransactionExecuteCallback(callback func(*QueryResultHandle, error)) uintptr {
	id := uintptr(nextTransactionExecuteID.Add(1))

	ctx := &TransactionExecuteContext{
		callback: callback,
		id:       id,
	}

	transactionExecuteContexts.Store(id, ctx)
	return id
}

// UnregisterTransactionExecuteCallback removes a transaction execute callback
func UnregisterTransactionExecuteCallback(id uintptr) {
	transactionExecuteContexts.Delete(id)
}

// RegisterTransactionCompleteCallback registers a transaction complete callback
func RegisterTransactionCompleteCallback(callback func(int, error)) uintptr {
	id := uintptr(nextTransactionCompleteID.Add(1))

	ctx := &TransactionCompleteContext{
		callback: callback,
		id:       id,
	}

	transactionCompleteContexts.Store(id, ctx)
	return id
}

// UnregisterTransactionCompleteCallback removes a transaction complete callback
func UnregisterTransactionCompleteCallback(id uintptr) {
	transactionCompleteContexts.Delete(id)
}

//export goTransactionBeginCallback
func goTransactionBeginCallback(contextPtr unsafe.Pointer, result C.dittoffi_result_dittoffi_transaction_ptr_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transaction begin callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)

	ctx, exists := transactionBeginContexts.Load(id)
	if !exists {
		return
	}
	beginCtx, ok := ctx.(*TransactionBeginContext)
	if !ok || beginCtx.callback == nil {
		return
	}

	// Check for error
	if result.error != nil {
		// CRITICAL: Extract error message BEFORE freeing
		var errMsg string
		if msgPtr := C.dittoffi_error_description(result.error); msgPtr != nil {
			errMsg = C.GoString(msgPtr)
			C.ditto_c_string_free(msgPtr)
		}
		C.dittoffi_error_free(result.error)
		beginCtx.callback(nil, fmt.Errorf("transaction begin failed: %s", errMsg))
		return
	}

	// Success case
	handle := &TransactionHandle{}
	handle.Initialize(transactionHandleFreer{ptr: result.success})
	beginCtx.callback(handle, nil)
}

//export goTransactionBeginFree
func goTransactionBeginFree(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transaction begin free callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)
	UnregisterTransactionBeginCallback(id)
}

//export goTransactionExecuteCallback
func goTransactionExecuteCallback(contextPtr unsafe.Pointer, result C.dittoffi_result_dittoffi_query_result_ptr_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transaction execute callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)

	ctx, exists := transactionExecuteContexts.Load(id)
	if !exists {
		return
	}
	execCtx, ok := ctx.(*TransactionExecuteContext)
	if !ok || execCtx.callback == nil {
		return
	}

	// Check for error
	if result.error != nil {
		// CRITICAL: Extract error message BEFORE freeing
		var errMsg string
		if msgPtr := C.dittoffi_error_description(result.error); msgPtr != nil {
			errMsg = C.GoString(msgPtr)
			C.ditto_c_string_free(msgPtr)
		}
		C.dittoffi_error_free(result.error)
		execCtx.callback(nil, fmt.Errorf("transaction execute failed: %s", errMsg))
		return
	}

	// Success case
	handle := &QueryResultHandle{}
	handle.Initialize(queryResultHandleFreer{ptr: result.success})
	execCtx.callback(handle, nil)
}

//export goTransactionExecuteFree
func goTransactionExecuteFree(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transaction execute free callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)
	UnregisterTransactionExecuteCallback(id)
}

//export goTransactionCompleteCallback
func goTransactionCompleteCallback(contextPtr unsafe.Pointer, result C.dittoffi_result_dittoffi_transaction_completion_action_t) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transaction complete callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)

	ctx, exists := transactionCompleteContexts.Load(id)
	if !exists {
		return
	}
	completeCtx, ok := ctx.(*TransactionCompleteContext)
	if !ok || completeCtx.callback == nil {
		return
	}

	// Check for error
	if result.error != nil {
		// CRITICAL: Extract error message BEFORE freeing
		var errMsg string
		if msgPtr := C.dittoffi_error_description(result.error); msgPtr != nil {
			errMsg = C.GoString(msgPtr)
			C.ditto_c_string_free(msgPtr)
		}
		C.dittoffi_error_free(result.error)
		completeCtx.callback(0, fmt.Errorf("transaction complete failed: %s", errMsg))
		return
	}

	// Success case - convert the completion action to Go int
	completeCtx.callback(int(result.success), nil)
}

//export goTransactionCompleteFree
func goTransactionCompleteFree(contextPtr unsafe.Pointer) {
	// CRITICAL: Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in transaction complete free callback: %v\n%s", r, debug.Stack())
		}
	}()

	if contextPtr == nil {
		return
	}

	id := uintptr(contextPtr)
	UnregisterTransactionCompleteCallback(id)
}

// TransactionHandle wraps the C DQL transaction pointer
type TransactionHandle struct {
	HandleCleaner[transactionHandleFreer]
}

type transactionHandleFreer struct {
	ptr *C.dittoffi_transaction_t
}

func (h transactionHandleFreer) free() {
	C.dittoffi_transaction_free(h.ptr)
}

// TransactionCompletionAction represents how to complete a transaction.
type TransactionCompletionAction int

const (
	TransactionCompletionActionCommit   TransactionCompletionAction = C.DITTOFFI_TRANSACTION_COMPLETION_ACTION_COMMIT
	TransactionCompletionActionRollback TransactionCompletionAction = C.DITTOFFI_TRANSACTION_COMPLETION_ACTION_ROLLBACK
)

// StoreBeginTransactionAsyncThrows begins a new transaction
func StoreBeginTransactionAsyncThrows(handle *DittoHandle, hint *string, isReadOnly bool) (*TransactionHandle, error) {
	var cHint *C.char
	if hint != nil && *hint != "" {
		cHint = C.CString(*hint)
		defer C.free(unsafe.Pointer(cHint))
	}

	type result struct {
		tx  *TransactionHandle
		err error
	}
	resultChan := make(chan result, 1)

	callbackID := RegisterTransactionBeginCallback(func(tx *TransactionHandle, err error) {
		if err != nil {
			resultChan <- result{nil, err}
		} else if tx == nil {
			resultChan <- result{nil, fmt.Errorf("transaction handle is nil")}
		} else {
			resultChan <- result{tx, nil}
		}
	})
	defer UnregisterTransactionBeginCallback(callbackID)

	cCallback := C.create_transaction_begin_callback(C.uintptr_t(callbackID))

	options := C.dittoffi_store_begin_transaction_options_make()
	options.is_read_only = C._Bool(isReadOnly)
	options.hint = cHint

	// Call FFI function
	C.dittoffi_store_begin_transaction_async_throws(
		handle.load(),
		options,
		cCallback)

	// Wait for result
	r := <-resultChan
	return r.tx, r.err
}

// TransactionExecuteAsyncThrows executes a query within a transaction
func TransactionExecuteAsyncThrows(tx *TransactionHandle, query string, args map[string]any) (*QueryResultHandle, error) {
	if tx == nil || tx.inner.ptr == nil {
		return nil, fmt.Errorf("invalid transaction handle")
	}

	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	argsCBOR, err := cbor.Encode(args)
	if err != nil {
		return nil, fmt.Errorf("failed to encode args: %w", err)
	}

	var argsSlice C.slice_ref_uint8_t
	if len(argsCBOR) > 0 {
		argsSlice = C.slice_ref_uint8_t{
			ptr: (*C.uint8_t)(unsafe.Pointer(&argsCBOR[0])),
			len: C.size_t(len(argsCBOR)),
		}
	}

	type result struct {
		result *QueryResultHandle
		err    error
	}
	resultChan := make(chan result, 1)

	callbackID := RegisterTransactionExecuteCallback(func(resultHandle *QueryResultHandle, err error) {
		if err != nil {
			resultChan <- result{nil, err}
		} else if resultHandle == nil {
			resultChan <- result{nil, fmt.Errorf("query result is nil")}
		} else {
			resultChan <- result{resultHandle, nil}
		}
	})
	defer UnregisterTransactionExecuteCallback(callbackID)

	cCallback := C.create_transaction_execute_callback(C.uintptr_t(callbackID))

	C.dittoffi_transaction_execute_async_throws(
		tx.inner.ptr,
		cQuery,
		argsSlice,
		cCallback)

	// Wait for result
	r := <-resultChan
	return r.result, r.err
}

// TransactionCompleteAsyncThrows completes a transaction
func TransactionCompleteAsyncThrows(tx *TransactionHandle, action TransactionCompletionAction) (TransactionCompletionAction, error) {
	if tx == nil || tx.inner.ptr == nil {
		return 0, fmt.Errorf("invalid transaction handle")
	}

	type result struct {
		action TransactionCompletionAction
		err    error
	}

	resultChan := make(chan result, 1)

	callbackID := RegisterTransactionCompleteCallback(func(action int, err error) {
		if err != nil {
			resultChan <- result{TransactionCompletionActionRollback, err}
		} else {
			resultChan <- result{TransactionCompletionAction(action), nil}
		}
	})
	defer UnregisterTransactionCompleteCallback(callbackID)

	cCallback := C.create_transaction_complete_callback(C.uintptr_t(callbackID))

	C.dittoffi_transaction_complete_async_throws(
		tx.inner.ptr,
		C.dittoffi_transaction_completion_action_t(action),
		cCallback)

	// Wait for result
	r := <-resultChan
	return r.action, r.err
}

// TransactionInfo gets transaction metadata
func TransactionInfo(tx *TransactionHandle) ([]byte, error) {
	if tx == nil || tx.inner.ptr == nil {
		return nil, fmt.Errorf("invalid transaction handle")
	}

	result := C.dittoffi_transaction_info(tx.inner.ptr)

	if result.ptr == nil || result.len == 0 {
		return nil, fmt.Errorf("transaction info returned empty")
	}
	info := bytesFromFFI(result)

	return info, nil
}
