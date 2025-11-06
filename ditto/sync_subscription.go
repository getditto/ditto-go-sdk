package ditto

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

var subscriptionIDCounter atomic.Uint64

func generateSubscriptionID() string {
	id := subscriptionIDCounter.Add(1)
	return fmt.Sprintf("sub_%d", id)
}

// SyncSubscription configures Ditto to receive updates from remote peers about documents matching the subscription’s query.
//
// Create a sync subscription by calling Sync.RegisterSubscription(). The subscription will remain active until either
// explicitly cancelled via SyncSubscription.Cancel() or the owning Ditto object is closed.
type SyncSubscription struct {
	mu        sync.Mutex
	id        string
	sync      *Sync
	handle    *ffi.SubscriptionHandle
	query     string
	queryArgs map[string]any
	canceled  bool
}

// newSyncSubscription creates a new sync subscription
func newSyncSubscription(sync *Sync, query string, args map[string]any) (*SyncSubscription, error) {
	// Register subscription with FFI
	handle, err := ffi.SyncRegisterSubscriptionThrows(sync.ditto.dittoHandle, query, args)
	if err != nil {
		return nil, err
	}

	subscription := &SyncSubscription{
		id:        generateSubscriptionID(),
		sync:      sync,
		handle:    handle,
		query:     query,
		queryArgs: args,
	}

	sdkDebugTraceF("gosdk: SyncSubscription.newSyncSubscription(): created with id %s; query %s", subscription.id, query)

	return subscription, nil
}

// Cancel cancels the subscription
func (s *SyncSubscription) Cancel() {
	sdkDebugTraceF("gosdk: SyncSubscription.Cancel() called for id %s; query: %s", s.id, s.query)

	s.cancel()

	// Remove from sync's subscription list
	s.sync.removeSubscription(s)
}

// cancel is internal cancel without removing from sync
func (s *SyncSubscription) cancel() {
	sdkDebugTraceF("gosdk: SyncSubscription.cancel() called for id %s; query: %s", s.id, s.query)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.canceled {
		return
	}

	s.canceled = true

	if s.handle != nil {
		ffi.CancelSubscription(s.handle)
		s.handle = nil
	}
}

// IsCancelled returns true if the subscription has been canceled
func (s *SyncSubscription) IsCancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.canceled
}

// Ditto returns the Ditto instance this subscription belongs to
func (s *SyncSubscription) Ditto() *Ditto {
	return s.sync.ditto
}

// QueryString returns the query string passed when registering the subscription
func (s *SyncSubscription) QueryString() string {
	return s.query
}

// QueryArguments returns the query arguments passed when registering the subscription
func (s *SyncSubscription) QueryArguments() map[string]any {
	return s.queryArgs
}
