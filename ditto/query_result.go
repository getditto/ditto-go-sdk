// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"iter"
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// QueryResult represents the result set from executing a DQL query.
//
// QueryResult provides methods to access individual items, iterate through
// results, and retrieve metadata about query execution. Results are
// immutable once created.
//
// When a QueryResult is returned from a query, the caller should
// extract necessary data from it and then call the Close()
// method to free associated resources in the underlying query engine.
type QueryResult struct {
	mu     sync.Mutex
	handle *ffi.QueryResultHandle
	count  int
}

// newQueryResult creates a new DQL query result from an FFI handle.
//
// This is for internal use. Application code receives
// QueryResult instances from Store.Execute() or Transaction.Execute()
func newQueryResult(handle *ffi.QueryResultHandle) *QueryResult {
	return &QueryResult{
		handle: handle,
		count:  ffi.GetQueryResultItemCount(handle),
	}
}

// Items returns an iterator of the items matching a DQL query
//
// Each item will be automatically closed after the iteration.
// Note that this means references to the items should not be held.
// Use itemsAsSlice if you need longer-lived references to the items.
func (r *QueryResult) Items() iter.Seq2[int, *QueryResultItem] {
	iterator := func(yield func(i int, item *QueryResultItem) bool) {
		count := r.ItemCount()

		for i := 0; i < count; i++ {
			item := r.Item(i)
			if item == nil {
				// Skip items we can't get
				continue
			}
			cont := yield(i, item)
			item.Close()
			if !cont {
				break
			}
		}
	}
	return iterator
}

// itemsAsSlice returns all items in the result set as a slice.
//
// Unlike Items(), this method does not automatically close the items.
// The caller is responsible for calling Close() for each returned item.
//
// This is private because we want to discourage users from using methods
// that would require explicit freeing of resources.
func (r *QueryResult) itemsAsSlice() []*QueryResultItem {
	count := r.ItemCount()
	items := make([]*QueryResultItem, 0, count)
	for i := 0; i < count; i++ {
		item := r.Item(i)
		if item != nil {
			items = append(items, item)
		}
	}
	return items
}

// Values returns the Document value for each item in the result set.
func (r *QueryResult) Values() []Document {
	documents := make([]Document, 0, r.count)
	for _, item := range r.Items() {
		documents = append(documents, item.Value())
	}
	return documents
}

// ItemCount returns the total number of items in the result set.
func (r *QueryResult) ItemCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.count
}

// Item returns the item in the result set with index i.
// Returns nil if index is out of bounds or the item cannot be retrieved.
func (r *QueryResult) Item(i int) *QueryResultItem {
	r.mu.Lock()
	defer r.mu.Unlock()

	if i >= r.count || i < 0 {
		return nil
	}

	// Get the item handle from FFI
	itemHandle, err := ffi.GetQueryResultItemAt(r.handle, i)
	if err != nil {
		LogErrorF("unable to get query result item %d: %v", i, err)
		return nil
	}

	// Create QueryResultItem with handle
	return newQueryResultItem(itemHandle)

}

// MutatedDocumentIDs returns the IDs of documents that were mutated locally by a mutating DQL query passed to
// Execute(). Empty slice if no documents have been mutated. nil if an error occurs.
//
// NOTE: A [StoreObserver] can only be registered with a SELECT query, which is non-mutating, and thus the query result
// passed to the [StoreObservationHandler] always returns an empty array in that case.
//
// IMPORTANT: The returned document IDs are not cached. Make sure to call this method once and keep the return value
// for as long as needed.
func (r *QueryResult) MutatedDocumentIDs() []any {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.handle == nil {
		return nil
	}

	// Get the count of mutated document IDs
	count := ffi.GetQueryResultMutatedDocumentIDCount(r.handle)
	if count == 0 {
		return []any{}
	}

	// Collect all document IDs
	var documentIDs []any
	for i := 0; i < count; i++ {
		// Get CBOR data for document ID at index
		cborData, err := ffi.GetQueryResultMutatedDocumentIDAt(r.handle, i)
		if err != nil {
			// Log error and continue - don't fail the entire operation
			continue
		}

		// Decode CBOR to get the raw value
		var value any
		err = cbor.Decode(cborData, &value)
		if err != nil {
			// Log error and continue
			continue
		}

		documentIDs = append(documentIDs, value)
	}

	return documentIDs
}

// CommitID returns the commit ID associated with this query result, if any.
//
// This ID uniquely identifies the state of the local store after the query was executed. The commit ID is available
// for all query results that reflect committed data.
//
// For write transactions, the commit ID is only available after the transaction has been successfully committed.
// Queries executed within an uncommitted transaction will not have a commit ID.
func (r *QueryResult) CommitID() (uint64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return ffi.GetQueryResultCommitID(r.handle)
}

// Close releases any FFI resources associated with this result.
//
// Always call Close when done with a QueryResult to prevent memory leaks.
// It's safe to call Close multiple times.
//
// Note: This does NOT close the individual QueryResultItems. Each QueryResultItem
// manages its own lifecycle independently and may outlive the QueryResult.
// QueryResultItems will be garbage collected and finalized when no longer referenced.
func (r *QueryResult) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.handle != nil {
		ffi.FreeQueryResult(r.handle)
		r.handle = nil
	}
}
