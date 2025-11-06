// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"sync"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// PresenceObserver represents an active presence observation subscription.
//
// PresenceObservers are created through Presence.Observe()
// and should be retained as long as updates are needed. Call Stop()
// to cease receiving updates
type PresenceObserver struct {
	mu      sync.RWMutex
	id      string
	handler PresenceObservationHandler
	handle  *ffi.PresenceObserverHandle
}

// PresenceObservationHandler is called for each peer change
type PresenceObservationHandler func(graph *PresenceGraph)

// newPresenceObserver creates a new presence observer with the specified callback.
//
// This is typically called internally by Presence.Observe().
// Applications should use Presence.Observe() instead of calling this directly.
func newPresenceObserver(callback PresenceObservationHandler) *PresenceObserver {
	return &PresenceObserver{
		id:      generateObserverID(),
		handler: callback,
	}
}

// Stop terminates the observer and cleans up all associated resources.
func (o *PresenceObserver) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// TODO: implement this
}

// TODO: Would it be more appropriate to use an atomic counter or a UUID in generateObserverID()?

// generateObserverID generates a unique observer ID
func generateObserverID() string {
	// Simple implementation, should use UUID in production
	return time.Now().Format("20060102150405.999999999")
}
