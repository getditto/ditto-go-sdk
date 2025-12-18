// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

var subscriptionIDCounter atomic.Uint64

func generateSubscriptionID() string {
	id := subscriptionIDCounter.Add(1)
	return fmt.Sprintf("sub_%d", id)
}

// SyncSubscription configures Ditto to receive updates from remote peers about documents matching the subscription's query.
//
// Create a sync subscription by calling Sync.RegisterSubscription(). The subscription will remain active until either
// explicitly canceled via SyncSubscription.Cancel() or the owning Ditto object is closed.
type SyncSubscription struct {
	mu     sync.Mutex
	id     string
	sync   *Sync
	handle *ffi.SubscriptionHandle
}

// newSyncSubscription creates a new sync subscription
func newSyncSubscription(sync *Sync, query string, args map[string]any) (*SyncSubscription, error) {
	// Register subscription with FFI
	handle, err := ffi.SyncRegisterSubscriptionThrows(sync.ditto.dittoHandle, query, args)
	if err != nil {
		return nil, err
	}

	subscription := &SyncSubscription{
		id:     generateSubscriptionID(),
		sync:   sync,
		handle: handle,
	}

	sdkDebugTraceF("gosdk: SyncSubscription.newSyncSubscription(): created with id %s; query %s", subscription.id, query)

	return subscription, nil
}

// Cancel cancels the subscription
func (s *SyncSubscription) Cancel() {
	s.cancel()

	// Remove from sync's subscription list
	s.sync.removeSubscription(s)
}

// cancel is internal cancel without removing from sync
func (s *SyncSubscription) cancel() {
	if enableSDKDebugTrace {
		query := ffi.SyncSubscriptionQueryString(s.handle)
		if query == "" {
			query = "(unknown/canceled)"
		}
		sdkDebugTraceF("gosdk: SyncSubscription.cancel() called for id %s; query: %s", s.id, query)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handle != nil {
		// Already holding the lock, so can't use IsCanceled()
		if ffi.SyncSubscriptionIsCancelled(s.handle) {
			return
		}

		ffi.CancelSubscription(s.handle)
		s.handle = nil
	}
}

// IsCanceled returns true if the subscription has been canceled
func (s *SyncSubscription) IsCanceled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ffi.SyncSubscriptionIsCancelled(s.handle)
}

// Ditto returns the Ditto instance this subscription belongs to
func (s *SyncSubscription) Ditto() *Ditto {
	return s.sync.ditto
}

// QueryString returns the query string passed when registering the subscription
func (s *SyncSubscription) QueryString() string {
	return ffi.SyncSubscriptionQueryString(s.handle)
}

// QueryArguments returns the query arguments passed when registering the subscription
func (s *SyncSubscription) QueryArguments() QueryArguments {
	cborBytes := ffi.SyncSubscriptionQueryArguments(s.handle)
	result, err := cbor.DecodeToMap(cborBytes)
	if err != nil {
		LogErrorF("failed to decode subscription query arguments: %v", err)
		return nil
	}
	return result
}
