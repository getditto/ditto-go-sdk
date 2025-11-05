package ditto

import (
	"errors"
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// Transaction represents an active transaction in the Ditto store.
//
// Transactions provide atomic, isolated execution of multiple DQL operations.
// All changes made within a transaction are committed together, or rolled back
// if an error occurs or if explicitly requested.
//
// Important: Do not call Store.Execute or start another transaction from within
// a transaction scope, as this will cause a deadlock.
type Transaction struct {
	mu     sync.Mutex
	handle *ffi.TransactionHandle
	store  *Store
	info   *TransactionInfo // cached on first call to Info()
	closed bool
}

// TransactionInfo contains metadata about a transaction.
type TransactionInfo struct {
	// ID is the globally unique identifier for this transaction.
	ID string `cbor:"id"`

	// Hint is an optional description provided when creating the transaction,
	// useful for debugging and monitoring.
	Hint *string `cbor:"hint,omitempty"`

	// IsReadOnly indicates whether this transaction can only perform read operations.
	// Read-only transactions can run concurrently and provide better performance.
	IsReadOnly bool `cbor:"is_read_only"`
}

// TransactionCompletionAction represents how to complete a transaction.
type TransactionCompletionAction int

const (
	// TransactionCompletionActionCommit commits the transaction, applying all changes.
	TransactionCompletionActionCommit = TransactionCompletionAction(ffi.TransactionCompletionActionCommit)

	// TransactionCompletionActionRollback rolls back the transaction, discarding all changes.
	TransactionCompletionActionRollback = TransactionCompletionAction(ffi.TransactionCompletionActionRollback)
)

// String returns a string representation of the TransactionCompletionAction.
func (a TransactionCompletionAction) String() string {
	switch a {
	case TransactionCompletionActionCommit:
		return "commit"
	case TransactionCompletionActionRollback:
		return "rollback"
	default:
		return "unknown"
	}
}

// TransactionOptions configures a new transaction.
// Can be nil to use default options (empty hint, read-write transaction).
type TransactionOptions struct {
	// Hint provides an optional description for the transaction,
	// useful for debugging and monitoring.
	Hint string

	// IsReadOnly creates a read-only transaction if true.
	//
	// Read-only transactions can run concurrently and provide better performance
	// for queries that don't modify data.
	IsReadOnly bool
}

// Execute executes a DQL query within this transaction and returns the query result.
//
// The query executes against the current state of the transaction, including any
// modifications made by previous Execute calls within the same transaction.
//
// If multiple arguments are provided, they will be merged into a single map before being passed to the query engine.
//
// NOTE: Only returns results from the local store without waiting for any
// SyncSubscription to have caught up with the latest changes.
//
// Returns an error if:
//   - query is not valid DQL
//   - arguments are not valid (contain unsupported types)
//   - this is a read-only transaction but a mutating query was executed
//   - a database store error occurs
func (t *Transaction) Execute(query string, args ...QueryArguments) (*QueryResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, ErrTransactionClosed
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

	resultHandle, err := ffi.TransactionExecuteAsyncThrows(t.handle, query, mergedArgs)
	if err != nil {
		return nil, err
	}

	// newQueryResult will take ownership of the result handle
	return newQueryResult(resultHandle), nil
}

// complete commits or rolls back the transaction.
//
// This method finalizes the transaction with the specified action.
// Returns the actual completion action taken, which may differ from
// the requested action in some error scenarios.
//
// This method is only called internally. It is not public.
func (t *Transaction) complete(action TransactionCompletionAction) (TransactionCompletionAction, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return TransactionCompletionActionRollback, ErrTransactionClosed
	}

	resultAction, err := ffi.TransactionCompleteAsyncThrows(t.handle, ffi.TransactionCompletionAction(action))
	t.closed = true
	if err != nil {
		return TransactionCompletionActionRollback, err
	}

	return TransactionCompletionAction(resultAction), nil
}

// Info returns metadata about this transaction.
//
// The returned TransactionInfo contains:
//   - ID: A globally unique identifier for this transaction
//   - Hint: An optional description provided when creating the transaction
//   - IsReadOnly: Whether this transaction can only perform read operations
//
// This information is useful for debugging, monitoring, and logging transaction activity.
func (t *Transaction) Info() TransactionInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.info != nil {
		return *t.info
	}

	infoBytes, err := ffi.TransactionInfo(t.handle)
	if err != nil {
		// This should never happen for a valid transaction handle
		// Return a default TransactionInfo if it does
		return TransactionInfo{}
	}

	var info TransactionInfo
	if err := cbor.Decode(infoBytes, &info); err != nil {
		// This should never happen for valid CBOR data from FFI
		// Return a default TransactionInfo if it does
		return TransactionInfo{}
	}

	t.info = &info
	return info
}

// cleanup ensures cleanup of the transaction resources.
//
// This method is called internally by the finalizer to ensure FFI resources
// are properly released. It is safe to call multiple times.
func (t *Transaction) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return
	}

	// Rollback if not already completed (ignore errors during cleanup)
	_, _ = ffi.TransactionCompleteAsyncThrows(t.handle, ffi.TransactionCompletionActionRollback)
	t.closed = true

	// Free the transaction handle
	ffi.TransactionFree(t.handle)
}

// Store returns the Store instance that this transaction belongs to.
//
// This allows accessing the parent store's methods from within a transaction context.
func (t *Transaction) Store() *Store {
	return t.store
}

// ErrTransactionClosed is returned when attempting to use a closed transaction.
var ErrTransactionClosed = errors.New("transaction is already closed")
