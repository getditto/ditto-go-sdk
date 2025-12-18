// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// StoreObservationHandler is the callback function invoked when query results change.
//
// The callback receives the current QueryResult containing all documents
// that match the observer's query. The callback is invoked:
//   - Immediately after registration with the initial results
//   - Whenever documents matching the query are added, updated, or removed
//
// Callbacks are executed on a separate goroutine and should be thread-safe.
// Long-running operations in the callback may delay subsequent updates.
type StoreObservationHandler func(*QueryResult)

// SignalNext is a function that signals the observer to deliver the next update.
//
// This function is provided to StoreObservationHandlerWithSignalNext to control the
// flow of updates. Call this function when your application is ready to
// receive the next change notification.
type SignalNext func()

// StoreObservationHandlerWithSignalNext is called when query results change with flow control.
//
// This callback variant provides backpressure control for high-frequency updates.
// The callback receives both the QueryResult and a SignalNext function that must
// be called to indicate readiness for the next update.
type StoreObservationHandlerWithSignalNext func(*QueryResult, SignalNext)

// StoreObserver represents an active observation of a DQL query.
//
// A StoreObserver monitors a SELECT query and notifies when matching documents
// change in the local store.  The observer remains active until explicitly
// canceled or until the Ditto instance is closed.
//
// Observers provide real-time updates as data changes due to:
//   - Local modifications
//   - Sync updates from remote peers
//   - Conflict resolution
//
// Example:
//
//	observer, err := RegisterObserver(
//		"SELECT * FROM tasks WHERE completed = false",
//		func(result *QueryResult) {
//			fmt.Printf("Active tasks: %d\n", result.ItemCount())
//		},
//	)
//	defer observer.Cancel()
//
// Thread Safety:
// All StoreObserver methods are thread-safe.
type StoreObserver struct {
	mu     sync.Mutex
	ditto  *Ditto
	handle *ffi.StoreObserverHandle
}

// Cancel stops the observer and releases its resources.
//
// After calling Cancel, the observer will no longer receive updates
// and its callback will not be invoked. This method is idempotent -
// calling it multiple times has no additional effect.
//
// It's recommended to call Cancel when the observer is no longer needed
// to free resources and prevent unnecessary processing.
func (o *StoreObserver) Cancel() {
	if enableSDKDebugTrace {
		query := o.QueryString()
		if query == "" {
			query = "(unknown/canceled)"
		}
		sdkDebugTraceF("gosdk: StoreObserver.Cancel(); query: %s", query)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if o.IsCanceled() {
		return
	}

	// Cancel FFI observer
	if o.handle != nil {
		ffi.CancelStoreObserver(o.handle)
		ffi.FreeStoreObserver(o.handle)
		o.handle = nil
	}
}

// Ditto returns the Ditto instance that this store observer is registered with
func (o *StoreObserver) Ditto() *Ditto {
	return o.ditto
}

// QueryString returns the DQL query string for this observer.
//
// This is the query that was provided when the observer was registered.
// The query string is immutable and does not change during the observer's
// lifetime.
func (o *StoreObserver) QueryString() string {
	return ffi.StoreObserverQueryString(o.handle)
}

// QueryArguments returns the query arguments for this observer.
//
// Returns the map of named parameters that was provided when the observer
// was registered. The returned map is a copy and modifications to it do
// not affect the observer.
func (o *StoreObserver) QueryArguments() QueryArguments {
	cborBytes := ffi.StoreObserverQueryArguments(o.handle)
	result, err := cbor.DecodeToMap(cborBytes)
	if err != nil {
		LogErrorF("failed to decode observer query arguments: %v", err)
		return nil
	}
	return result
}

// IsCanceled returns true if the observer has been canceled.
//
// A canceled observer no longer receives updates. An observer is
// considered canceled if:
//   - Cancel() was explicitly called
//   - The owning Store was closed
//   - The Ditto instance was shut down
func (o *StoreObserver) IsCanceled() bool {
	return ffi.StoreObserverIsCancelled(o.handle)
}
