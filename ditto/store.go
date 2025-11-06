// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"fmt"
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// QueryArguments is a map of string keys to JSON values.
//
// It is used in calls to Execute(), RegisterObserver(), and RegisterSubscription()..
type QueryArguments map[string]any

// QueryExecutor defines the interface for executing DQL queries.
//
// This interface is implemented by Store and Transaction, allowing both to execute
// DQL queries with the same method signature.
type QueryExecutor interface {
	// Execute executes a DQL query and returns matching items as a query result.
	//
	// The caller must close the returned QueryResult after extracting data from it.
	//
	// If multiple arguments are provided, they will be merged into a single map before being passed to the query engine.
	//
	// NOTE: Only returns results from the local store without waiting for any SyncSubscription to have
	// caught up with the latest changes. Only use this method if your program must proceed with immediate results.
	// Use a StoreObserver to receive updates to query results as soon as they have been synced to this peer.
	//
	// Parameters:
	//   - query: A string containing a valid query expressed in DQL
	//   - arguments: Optional variadic query arguments (dictionaries of values keyed by placeholder name without the leading ':')
	//
	// Returns an error if:
	//   - query is not valid DQL
	//   - arguments are not valid (contain unsupported types)
	//   - a mutating query is attempted in a read-only transaction (Transaction only)
	//   - a database store error occurs
	Execute(query string, arguments ...QueryArguments) (*QueryResult, error)
}

// Store is the entrypoint for all actions that relate to data stored by Ditto.
//
// You don't create one directly but can access it from a particular Ditto
// instance via its store property.
type Store struct {
	ditto       *Ditto
	dittoHandle *ffi.DittoHandle

	mu        sync.RWMutex
	observers []*StoreObserver
}

// newStore creates a new Store instance.
// This is an internal constructor used by the Ditto SDK.
// Users should access the Store through ditto.Store() instead.
func newStore(ditto *Ditto) *Store {
	return &Store{
		ditto:       ditto,
		dittoHandle: ditto.dittoHandle,
	}
}

// Execute executes a DQL query and returns matching items as a query result.
//
// The caller must close the returned QueryResult after extracting data from it.
//
// If multiple arguments are provided, they will be merged into a single map before being passed to the query engine.
//
// NOTE: Only returns results from the local store without waiting for any SyncSubscription to have
// caught up with the latest changes. Only use this method if your program must proceed with immediate results.
// Use a StoreObserver to receive updates to query results as soon as they have been synced to this peer.
//
// Parameters:
//   - query: A string containing a valid query expressed in DQL
//   - args: Optional variadic query arguments (dictionaries of values keyed by placeholder name without the leading ':')
//
// Returns an error if:
//   - query is not valid DQL
//   - arguments are not valid (contain unsupported types)
//   - a database store error occurs
func (s *Store) Execute(query string, args ...QueryArguments) (*QueryResult, error) {
	sdkDebugTraceF("gosdk: Store.Execute query=%s", query)

	if !s.dittoHandle.IsValid() {
		return nil, ErrDittoClosed
	}

	// If any args were given, merge them into a single map
	var mergedArgs QueryArguments
	if len(args) > 0 {
		mergedArgs = make(QueryArguments)
		for _, argMap := range args {
			for k, v := range argMap {
				mergedArgs[k] = v
			}
		}
	}

	resultHandle, err := ffi.ExecuteStatement(s.dittoHandle, query, mergedArgs)

	if err != nil {
		return nil, convertFFIError(err)
	}

	return newQueryResult(resultHandle), nil
}

// RegisterObserver installs and returns a store observer for a query, configuring
// Ditto to trigger the passed in change handler whenever documents in the local
// store change such that the result of the matching query changes.
//
// The passed in query must be a SELECT query, otherwise a store error is returned.
//
// Use a Differ to calculate a diff between subsequent results delivered to the change handler.
//
// The first invocation of the change handler will happen shortly after
// this method has returned.
//
// The observer will remain active until:
//   - the observer.Cancel() method is called, or
//   - the owning Ditto instance has shut down
//
// Parameters:
//   - query: A string containing a valid SELECT query expressed in DQL
//   - args: A dictionary of values keyed by the placeholder name without the leading ':'.
//     Example: QueryArguments{"mileage": 123}. Can be nil for queries without parameters.
//   - handler: Function called when query results change
//
// Returns:
//   - An active StoreObserver for the passed in query and arguments. You'll have to keep it
//     to be able to cancel the observation. Otherwise it will remain active until the owning
//     Ditto object goes out of scope.
//
// Returns an error if:
//   - query is not valid DQL
//   - query is not a SELECT query
//   - arguments are not valid (contain unsupported types)
//
// Call the returned StoreObserver's Cancel() method to stop observing changes.
func (s *Store) RegisterObserver(query string, args QueryArguments, handler StoreObservationHandler) (*StoreObserver, error) {
	sdkDebugTraceF("gosdk: Store.RegisterObserver query=%s args=%s", query, args)

	if handler == nil {
		return nil, &DittoError{Code: ffi.ErrorCodeInternal, Message: "store observation handler is nil"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dittoHandle.IsValid() {
		return nil, ErrDittoClosed
	}

	// Create wrapper handler that converts FFI result to Go QueryResult
	wrappedCallback := func(resultHandle *ffi.QueryResultHandle) {
		if resultHandle == nil {
			return
		}

		result := newQueryResult(resultHandle)
		sdkDebugTraceF("gosdk: invoking user store observation handler; query=%s args=%s", query, args)
		handler(result)
		sdkDebugTraceF("gosdk: user store observation handler returned; query=%s args=%s", query, args)
	}

	observerHandle, err := ffi.StoreRegisterObserverThrows(
		s.dittoHandle,
		query,
		args,
		wrappedCallback,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register observer: %w", err)
	}

	observer := &StoreObserver{
		store:     s,
		handle:    observerHandle,
		query:     query,
		queryArgs: args,
		callback:  handler,
	}

	s.observers = append(s.observers, observer)

	return observer, nil
}

// RegisterObserverWithSignalNext creates a store observer with backpressure control.
//
// This variant of RegisterObserver provides flow control for high-frequency
// updates. The callback receives both the query result and a signal function
// that must be called when the application is ready to receive the next update.
// This prevents overwhelming the application with rapid changes.
//
// The signal function should be called after processing each update to indicate
// readiness for the next one. The next invocation of the callback will coalesce
// the changes since the previous one.
//
// After invoking the callback once, the observer will wait to deliver
// another callback until after you've called SignalNext.
//
// Parameters:
//   - query: A SELECT query in DQL format
//   - args: Optional query parameter maps
//   - handler: Function called with results and a signal function
//
// Use this method when:
//   - Query results change frequently
//   - Processing updates is computationally expensive
//   - You need to control the rate of updates
func (s *Store) RegisterObserverWithSignalNext(query string, args QueryArguments, handler StoreObservationHandlerWithSignalNext) (*StoreObserver, error) {
	sdkDebugTraceF("gosdk: Store.RegisterObserverWithSignalNext query=%s args=%s", query, args)

	if handler == nil {
		return nil, &DittoError{Code: ffi.ErrorCodeInternal, Message: "store observation handler is nil"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dittoHandle.IsValid() {
		return nil, ErrDittoClosed
	}

	// Create wrapper handler that converts FFI result to QueryResult
	// and passes through the signal function from FFI
	wrappedCallback := func(resultHandle *ffi.QueryResultHandle, signalNext func()) {
		if resultHandle == nil {
			return
		}

		result := newQueryResult(resultHandle)
		handler(result, signalNext)
	}

	observerHandle, err := ffi.StoreRegisterObserverWithSignalNextThrows(
		s.dittoHandle,
		query,
		args,
		wrappedCallback,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register observer with signal: %w", err)
	}

	observer := &StoreObserver{
		store:                  s,
		handle:                 observerHandle,
		query:                  query,
		queryArgs:              args,
		callbackWithSignalNext: handler,
	}

	s.observers = append(s.observers, observer)

	return observer, nil
}

// Observers returns all currently active store observers.
//
// This method provides visibility into all observers that are currently
// registered and monitoring for changes to query results.
//
// Returns:
//   - A slice of active StoreObserver instances
//   - nil if an error occurs
//
// The returned slice is a snapshot of observers at the time of the call.
// Observers may be added or removed after this method returns.
//
// The order of the values in the result is not defined, and may vary between calls.
// The result should be treated as an unordered set.
func (s *Store) Observers() []*StoreObserver {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get observers from FFI
	observerHandles, err := ffi.StoreObservers(s.dittoHandle)
	if err != nil {
		return nil
	}

	// Convert FFI handles to StoreObserver instances
	observers := make([]*StoreObserver, 0, len(observerHandles))

	// Match FFI handles with our tracked observers
	for _, trackedObserver := range s.observers {
		if trackedObserver != nil && trackedObserver.handle != nil {
			// Check if this observer is still active according to FFI
			for _, ffiHandle := range observerHandles {
				if ffiHandle != nil && ffiHandle == trackedObserver.handle {
					observers = append(observers, trackedObserver)
					break
				}
			}
		}
	}

	return observers
}

// NewAttachment creates a new attachment from a file on disk.
//
// The file at the specified path is copied into Ditto's store. The returned
// Attachment object can then be inserted into a document. The attachment
// and its metadata will be replicated to other peers.
//
// Parameters:
//   - path: Absolute or relative path to the file
//   - metadata: Optional metadata to associate with the attachment
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (s *Store) NewAttachment(path string, metadata map[string]any) (*Attachment, error) {
	if !s.dittoHandle.IsValid() {
		return nil, ErrDittoClosed
	}

	return newAttachment(s.dittoHandle, path, metadata)
}

// FetchAttachment retrieves an attachment using its token.
//
// This method initiates the fetch of an attachment, which may need to be
// downloaded from other peers if not available locally. The callback is
// invoked with progress updates and when the fetch completes or fails.
//
// Parameters:
//   - token: The attachment token obtained from a document
//   - callback: Function called with fetch progress events
//
// The callback receives AttachmentFetchEvent values indicating:
//   - Progress: Download progress with bytes transferred
//   - Completed: Attachment successfully fetched
//   - Deleted: Attachment was deleted
//
// Returns a AttachmentFetcher that can be used to cancel the fetch.
// Keep this object alive for the duration of the fetch operation.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (s *Store) FetchAttachment(token *AttachmentToken, callback FetchCallback) (*AttachmentFetcher, error) {
	if !s.dittoHandle.IsValid() {
		return nil, ErrDittoClosed
	}

	if token == nil {
		return nil, &DittoError{Code: ffi.ErrorCodeInternal, Message: "invalid attachment token"}
	}

	fetcher := newAttachmentFetcher(s.dittoHandle, token, callback)
	return fetcher, nil
}

// Transaction executes multiple DQL queries within a single atomic transaction.
//
// This ensures that either all statements are executed successfully, or none are
// executed at all, providing strong consistency guarantees. Certain mesh
// configurations may impose limitations on these guarantees. For more details,
// refer to the Ditto documentation at https://ditto.live/link/sdk-latest-crud-transactions.
//
// Transactions are initiated as read-write transactions by default, and only a
// single read-write transaction is being executed at any given time. Any other
// read-write transaction started concurrently will wait until the current
// transaction has been committed or rolled back. Therefore, it is crucial to
// make sure a transaction finishes as early as possible so other read-write
// transactions aren't blocked for a long time.
//
// A transaction can also be configured to be read-only using TransactionOptions.
// Multiple read-only transactions can be executed concurrently. However, executing
// a mutating DQL statement in a read-only transaction will return an error.
//
// If errors occur in an Execute() call within a transaction function and the error
// is caught and handled within the function, the transaction will continue to run
// and not be rolled back. When an error is returned at any point from the
// transaction function or while committing the transaction, the transaction is
// implicitly rolled back and the error is returned to the caller.
//
// When a Ditto instance goes out of scope, it will drive all pending transactions
// to completion before being shut down.
//
// Important: Calling Store.Execute() or creating a nested transaction within a
// transaction may lead to a deadlock.
//
// For a complete guide on transactions, please refer to the Ditto documentation at
// https://ditto.live/link/sdk-latest-crud-transactions.
//
// Parameters:
//   - opts: Optional transaction configuration. Can be nil to use defaults (no hint,
//     read-write transaction).
//   - fn: Function to execute within the transaction. It receives a Transaction
//     object and should return the desired completion action (commit or rollback)
//     and any error encountered.
//
// Returns:
//   - TransactionCompletionAction: The action taken (commit or rollback)
//   - error: An error if the transaction fails, or any error returned from fn
func (s *Store) Transaction(opts *TransactionOptions, fn func(*Transaction) (TransactionCompletionAction, error)) (TransactionCompletionAction, error) {
	tx, err := s.beginTransaction(opts)
	if err != nil {
		return TransactionCompletionActionRollback, err
	}

	defer tx.cleanup()

	action, err := fn(tx)
	if err != nil {
		// Rollback on error (will be logged by core)
		_, _ = tx.complete(TransactionCompletionActionRollback)
		return TransactionCompletionActionRollback, err
	}

	finalAction, err := tx.complete(action)
	if err != nil {
		return TransactionCompletionActionRollback, err
	}

	return finalAction, nil
}

// TransactionWithResult executes a transaction and returns a value.
//
// This is a convenience function similar to Store.Transaction, but propagates the return
// value of the transaction function rather than the completion action.
//
// The transaction is committed implicitly if the function returns without error,
// and rolled back if an error is returned.
//
// See Store.Transaction() for important details about transaction behavior, deadlock
// warnings, and mesh configuration limitations.
//
// Note that this is a package-level function rather than a method on Store, because methods with
// generic type parameters are not supported in Go.
//
// Parameters:
//   - store: The Store to execute the transaction on
//   - opts: Optional transaction configuration. Can be nil to use defaults (no hint,
//     read-write transaction).
//   - fn: Function to execute within the transaction. It receives a Transaction
//     object and should return a value and any error encountered.
//
// Returns:
//   - T: The value returned from fn, or the zero value for T on error
//   - error: An error if the transaction fails, or any error returned from fn
func TransactionWithResult[T any](store *Store, opts *TransactionOptions, fn func(*Transaction) (T, error)) (T, error) {
	var result T
	wrappedFn := func(t *Transaction) (TransactionCompletionAction, error) {
		var err error
		result, err = fn(t)
		if err != nil {
			return TransactionCompletionActionRollback, err
		}
		return TransactionCompletionActionCommit, nil
	}
	_, err := store.Transaction(opts, wrappedFn)
	return result, err
}

// beginTransaction is an internal helper to start a new transaction.
//
// This method is called by the public Transaction methods. It should not
// be called directly by SDK users.
func (s *Store) beginTransaction(opts *TransactionOptions) (*Transaction, error) {
	if !s.dittoHandle.IsValid() {
		return nil, ErrDittoClosed
	}

	var hint string
	var isReadOnly bool
	if opts != nil {
		hint = opts.Hint
		isReadOnly = opts.IsReadOnly
	}

	var hintPtr *string
	if hint != "" {
		hintPtr = &hint
	}

	txHandle, err := ffi.StoreBeginTransactionAsyncThrows(s.dittoHandle, hintPtr, isReadOnly)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		handle: txHandle,
		store:  s,
	}, nil
}

// transactions returns information about currently active transactions.
//
// This method returns a slice of TransactionInfo for all transactions
// that are currently active (started but not yet committed or rolled back).
// This is useful for monitoring and debugging transaction usage.
//
// Returns nil if an error occurs.
//
// This method is currently private, as it is in other SDKs.
func (s *Store) transactions() []*TransactionInfo {
	cborData, err := ffi.GetStoreTransactions(s.dittoHandle)
	if err != nil {
		LogErrorF("failed to get store txInfo: %v", err)
		return nil
	}

	var txInfo []*TransactionInfo
	err = cbor.Decode(cborData, &txInfo)
	if err != nil {
		LogErrorF("failed to decode txInfo: %v", err)
		return nil
	}

	return txInfo
}

// cancelAllObservers cancels all active observers
func (s *Store) cancelAllObservers() {
	sdkDebugTrace("gosdk: Store.cancelAllObservers()")

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, observer := range s.observers {
		observer.Cancel()
	}
	s.observers = nil
}
