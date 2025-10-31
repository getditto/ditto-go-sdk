package ditto

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"

	"github.com/getditto/ditto-go-sdk/internal/cbor"
	"github.com/getditto/ditto-go-sdk/internal/ffi"
)

// QueryResultItem represents a single match of a DQL query, similar to a "row" in SQL terms.
// It's a reference type serving as a "cursor", allowing for efficient access of the underlying
// data in various formats.
//
// The Value property is lazily materialized and kept in memory until it goes out of scope.
// To reduce the memory footprint, structure your code such that items can be processed as
// a stream, i.e. one by one (or in batches) and Dematerialize() or Close() them right after use.
type QueryResultItem struct {
	mu                sync.Mutex
	handle            *ffi.QueryResultItemHandle
	materializedValue Document
	closed            bool
}

// TODO: Do we really need the `closed` member? The `handle` may be enough to determine that state.

// newQueryResultItem creates a new QueryResultItem from an FFI handle.
// This is primarily for internal use.
func newQueryResultItem(handle *ffi.QueryResultItemHandle) *QueryResultItem {
	if handle == nil {
		return nil
	}

	item := &QueryResultItem{
		handle: handle,
	}

	// Set up finalizer to auto-free when garbage collected
	runtime.SetFinalizer(item, func(i *QueryResultItem) {
		i.Close()
	})

	return item
}

// UnmarshalTo decodes the query result into the value pointed to by v.
//
// Returns an error if the result cannot be retrieved, or if there is an error decoding the
// result into the destination.
func (i *QueryResultItem) UnmarshalTo(v any) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return ErrDittoClosed
	}

	// Get CBOR data from handle
	cborData, err := ffi.GetQueryResultItemCBORFromHandle(i.handle)
	if err != nil {
		return fmt.Errorf("failed to get CBOR data: %w", err)
	}

	//
	if err := cbor.Decode(cborData, v); err != nil {
		return fmt.Errorf("failed to decode CBOR: %w", err)

	}
	return nil
}

// Value returns the content as a materialized dictionary.
//
// The item's value is materialized on first access and subsequently on each access
// after performing Dematerialize(). Once materialized, the value is kept in memory
// until explicitly dematerialized or closed, or the item is garbage collected.
//
// Returns nil if this QueryResultItem has been closed.
func (i *QueryResultItem) Value() Document {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return Document{}
	}

	// If already materialized, return it
	if i.materializedValue != nil {
		return i.materializedValue
	}

	if i.handle.IsValid() {
		i.materializeInternal()
	}

	return i.materializedValue
}

// IsMaterialized returns true if value is currently held materialized in memory,
// otherwise returns false.
//
// See also
//   - Materialize()
//   - Dematerialize()
func (i *QueryResultItem) IsMaterialized() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.materializedValue != nil
}

// Materialize loads the CBOR representation of the item's content, decodes it as a
// dictionary so it can be accessed via Value(). Keeps the dictionary in memory until
// Dematerialize() or Close() is called. No-op if value is already materialized.
func (i *QueryResultItem) Materialize() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed || i.materializedValue != nil {
		return
	}

	i.materializeInternal()
}

// materializeInternal performs the actual materialization (must be called with lock held)
func (i *QueryResultItem) materializeInternal() {
	if !i.handle.IsValid() {
		return
	}

	// Get CBOR data from handle
	cborData, err := ffi.GetQueryResultItemCBORFromHandle(i.handle)
	if err != nil {
		LogErrorF("Failed to get CBOR data: %v", err)
		return
	}

	// Decode CBOR to map
	result, err := cbor.DecodeToMap(cborData)
	if err != nil {
		LogErrorF("Failed to decode CBOR: %v", err)
		// Create error marker
		i.materializedValue = Document{
			"_cbor_decode_error": true,
			"_error_message":     err.Error(),
		}
		return
	}

	i.materializedValue = result
}

// Dematerialize releases the materialized value from memory. No-op if item is not materialized.
//
// Note: If you will not need to materialize the value again, use Close() instead to free all underlying resources.
func (i *QueryResultItem) Dematerialize() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.materializedValue = nil
}

// CBORData returns the content of the item as CBOR data.
//
// Returns nil on error.
//
// Important: The returned CBOR data is not cached, make sure to call this method once
// and keep the returned data for as long as needed.
func (i *QueryResultItem) CBORData() []byte {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return nil
	}

	// If we have a handle, get fresh CBOR from it
	if i.handle.IsValid() {
		cborData, err := ffi.GetQueryResultItemCBORFromHandle(i.handle)
		if err != nil {
			LogErrorF("Failed to get CBOR data: %v", err)
			return nil
		}
		return cborData
	}

	// If no handle but we have materialized value, encode it to CBOR
	if i.materializedValue != nil {
		cborData, err := cbor.Encode(i.materializedValue)
		if err != nil {
			LogErrorF("Failed to encode to CBOR: %v", err)
			return nil
		}
		return cborData
	}

	LogErrorF("Item handle is closed, and no cached materialized value available")

	return nil
}

// JSONData returns the content of the item as JSON data containing a UTF-8 encoded string.
//
// Important: The returned JSON data is not cached, make sure to call this method once
// and keep returned data for as long as needed.
func (i *QueryResultItem) JSONData() []byte {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return nil
	}

	// If we have a handle, get JSON from FFI
	if i.handle.IsValid() {
		jsonData, err := ffi.GetQueryResultItemJSONData(i.handle)
		if err != nil {
			LogErrorF("Failed to get JSON data: %v", err)
			return nil
		}
		return jsonData
	}

	// If no handle but we have materialized value, encode it to JSON
	if i.materializedValue != nil {
		jsonData, err := json.Marshal(i.materializedValue)
		if err != nil {
			LogErrorF("Failed to encode to JSON: %v", err)
			return nil
		}
		return jsonData
	}

	LogErrorF("Item handle is closed, and no cached materialized value available")

	return nil
}

// JSONString deserializes the item value to a JSON string.
//
// Returns an error if the value cannot be marshaled to JSON.
func (i *QueryResultItem) JSONString() string {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return ""
	}

	// If we have a handle, get JSON from FFI
	if i.handle.IsValid() {
		jsonStr, err := ffi.GetQueryResultItemJSONString(i.handle)
		if err != nil {
			LogErrorF("Failed to get JSON data: %v", err)
			return ""
		}
		return jsonStr
	}

	// If no handle but we have materialized value, encode it to JSON
	if i.materializedValue != nil {
		jsonData, err := json.Marshal(i.materializedValue)
		if err != nil {
			LogErrorF("Failed to encode to JSON: %v", err)
			return ""
		}
		return string(jsonData)
	}

	LogErrorF("Item handle is closed, and no cached materialized value available")

	return ""
}

// Close releases any internal resources associated with this item.
//
// After calling Close, the item becomes invalid and should not be used.
// It's safe to call Close multiple times.
func (i *QueryResultItem) Close() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return
	}

	// Clear materialized value
	i.materializedValue = nil

	// Free the handle if we have one
	if i.handle.IsValid() {
		i.handle.Free()
		i.handle = nil
	}

	i.closed = true
}
