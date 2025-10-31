package ditto

import (
	"sync"

	"github.com/getditto/ditto-go-sdk/internal/ffi"
)

// Sync provides access to sync related functionality of Ditto.
type Sync struct {
	mu            sync.RWMutex
	ditto         *Ditto
	subscriptions []*SyncSubscription
}

// newSync creates a new Sync instance
func newSync(ditto *Ditto) *Sync {
	return &Sync{
		ditto: ditto,
	}
}

// RegisterSubscription installs and returns a sync subscription for a query, configuring
// Ditto to receive updates from other peers for documents matching that
// query. The passed in query must be a SELECT query, otherwise a
// store error is returned..
//
// Parameters:
//   - query: a string containing a valid query expressed in DQL
//   - args: Optional query arguments (variadic for convenience)
//
// Returns:
//   - An active SyncSubscription for the passed in query and arguments. The caller is responsible for keeping a reference to it and calling Cancel() when it is no longer needed. Otherwise it will remain active until the owning Ditto object is closed.
//   - An error if the query is invalid or registration fails
func (s *Sync) RegisterSubscription(query string, args ...map[string]any) (*SyncSubscription, error) {
	sdkDebugTraceF("gosdk: Sync.RegisterSubscription(); query: %s", query)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ditto.closed {
		return nil, ErrDittoClosed
	}

	// Get arguments if provided
	var queryArgs map[string]any
	if len(args) > 0 {
		queryArgs = args[0]
	}

	subscription, err := newSyncSubscription(s, query, queryArgs)
	if err != nil {
		return nil, err
	}

	s.subscriptions = append(s.subscriptions, subscription)
	return subscription, nil
}

// Subscriptions returns a snapshot of all active sync subscriptions.
//
// The returned slice is a copy and can be safely modified without
// affecting the internal subscription list.
//
// The order of the returned subscriptions is not defined, and may vary
// between calls. The caller should treat it as an unordered set.
//
// Returns:
//   - A slice of active SyncSubscription instances
func (s *Sync) Subscriptions() []*SyncSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*SyncSubscription, len(s.subscriptions))
	copy(result, s.subscriptions)
	return result
}

// removeSubscription removes a subscription from the sync's internal list.
// This is called internally when a subscription is cancelled.
func (s *Sync) removeSubscription(subscription *SyncSubscription) {
	sdkDebugTraceF("gosdk: Sync.removeSubscription(); id=%s; query=%s", subscription.id, subscription.query)

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.subscriptions {
		if sub == subscription {
			s.subscriptions = append(s.subscriptions[:i], s.subscriptions[i+1:]...)
			break
		}
	}
}

// cancelAllSubscriptions cancels all active subscriptions.
// This is called internally when the Ditto instance is closed.
func (s *Sync) cancelAllSubscriptions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, subscription := range s.subscriptions {
		subscription.cancel()
	}
	s.subscriptions = nil
}

// Start starts the network transports. Ditto will connect to other devices.
//
// By default Ditto will enable all peer-to-peer transport types. The default network configuration can be modified
// with updateTransportConfig() or replaced via the transportConfig property.
func (s *Sync) Start() error {
	s.ditto.mu.Lock()
	defer s.ditto.mu.Unlock()

	if s.ditto.closed {
		return ErrDittoClosed
	}

	if err := ffi.StartSync(s.ditto.dittoHandle); err != nil {
		return convertFFIError(err)
	}

	s.ditto.syncActive = true
	return nil
}

// Stop stops all network transports.
//
// You may continue to use the Ditto store locally but no data will sync to or from other devices.
func (s *Sync) Stop() {
	s.ditto.mu.Lock()
	defer s.ditto.mu.Unlock()

	if s.ditto.closed {
		return
	}

	ffi.StopSync(s.ditto.dittoHandle)
	s.ditto.syncActive = false
}

// IsActive returns true if synchronization is currently active.
func (s *Sync) IsActive() bool {
	s.ditto.mu.RLock()
	defer s.ditto.mu.RUnlock()
	return s.ditto.syncActive
}
