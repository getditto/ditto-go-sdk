package ffi

/*
#include <stdlib.h>
#include "dittoffi.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// DifferHandle wraps the C differ pointer
type DifferHandle struct {
	ptr *C.dittoffi_differ_t
}

// NewDiffer creates a new differ with default identity key paths (["_id"])
func NewDiffer() *DifferHandle {
	ptr := C.dittoffi_differ_new()
	if ptr == nil {
		return nil
	}
	return &DifferHandle{ptr: ptr}
}

// DiffItems calculates the diff between the last set of items and the current ones.
// Takes a slice of QueryResultItemHandle pointers.
// Returns raw CBOR bytes that can be decoded by the caller.
// Note: This method updates the differ's internal state. Subsequent calls to DiffItems
// will compare against the items provided in this call.
func (d *DifferHandle) DiffItems(items []*QueryResultItemHandle) ([]byte, error) {
	if d == nil || d.ptr == nil {
		return nil, fmt.Errorf("invalid differ handle")
	}

	var ffiItems []*C.dittoffi_query_result_item_t
	ffiSlice := C.slice_ref_dittoffi_query_result_item_ptr_t{ptr: nil, len: 0}
	itemCount := len(items)
	if itemCount > 0 {
		ffiItems = make([]*C.dittoffi_query_result_item_t, itemCount)
		for i, item := range items {
			if item != nil {
				ffiItems[i] = item.ptr
			}
		}
		ffiSlice.ptr = (**C.dittoffi_query_result_item_t)(unsafe.Pointer(&ffiItems[0]))
		ffiSlice.len = C.size_t(itemCount)
	}

	diffCBOR := C.dittoffi_differ_diff(d.ptr, ffiSlice)
	if diffCBOR.ptr == nil {
		return nil, fmt.Errorf("differ returned nil result")
	}

	return bytesFromFFI(diffCBOR), nil
}

// Free frees the differ handle
func (d *DifferHandle) Free() {
	if d != nil && d.ptr != nil {
		C.dittoffi_differ_free(d.ptr)
		d.ptr = nil
	}
}
