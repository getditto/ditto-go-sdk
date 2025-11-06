// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// Authenticator is used to authenticate with Ditto Cloud.
// You must set up authentication before you can start syncing.
type Authenticator struct {
	mu          sync.RWMutex
	dittoHandle *ffi.DittoHandle
	ditto       *Ditto // Back reference to parent Ditto instance

	// Expiration handler callback
	expirationHandler AuthenticationExpirationHandler

	// Status observers
	statusObservers map[uint64]AuthenticationStatusObservationHandler
	nextObserverID  uint64

	// Current authentication status
	currentStatus *AuthenticationStatus
}

// AuthenticationExpirationHandler is called when authentication is required or about to expire.
//
// See Authenticator.SetExpirationHandler()
type AuthenticationExpirationHandler func(ditto *Ditto, timeUntilExpiration time.Duration)

// AuthenticationStatusObservationHandler is called when authentication status changes.
type AuthenticationStatusObservationHandler func(status *AuthenticationStatus)

// ErrAuthenticatorNotAvailable is returned when authentication methods are called
// but the Authenticator is not available (e.g., when using SmallPeersOnly connection).
var ErrAuthenticatorNotAvailable = errors.New("authenticator is not available for this connection type")

// ErrExpirationHandlerNotSet is returned when StartSync is called on a server
// connection without setting an expiration handler.
var ErrExpirationHandlerNotSet = errors.New("expiration handler must be set before starting sync with server connection")

// newAuthenticator creates a new Authenticator instance
func newAuthenticator(handle *ffi.DittoHandle, ditto *Ditto) *Authenticator {
	return &Authenticator{
		dittoHandle:     handle,
		ditto:           ditto,
		statusObservers: make(map[uint64]AuthenticationStatusObservationHandler),
		currentStatus:   &AuthenticationStatus{},
	}
}

// Status returns the current authentication status.
//
// This method never returns nil.
func (a *Authenticator) Status() *AuthenticationStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get the current user ID from FFI
	userID := ffi.AuthClientGetUserID(a.dittoHandle)

	// Check if we're authenticated based on whether we have a user ID
	isAuthenticated := userID != ""

	if a.currentStatus == nil {
		return &AuthenticationStatus{
			IsAuthenticated: isAuthenticated,
			UserID:          userID,
		}
	}

	// Return a copy with the latest user ID
	return &AuthenticationStatus{
		IsAuthenticated: isAuthenticated,
		UserID:          userID,
		ClientInfo:      a.currentStatus.ClientInfo,
	}
}

// SetExpirationHandler sets the handler that will be called when authentication for this Ditto instance is about to
// expire.
//
// Assign a callback function to this property to be notified before authentication expires, allowing you to login
// or perform other necessary actions.
//
// Important: If the Ditto instance was initialized with DittoConfig.Connect set to DittoConfigConnectServer, this
// property must be set and the handler must properly authenticate when triggered.
func (a *Authenticator) SetExpirationHandler(handler AuthenticationExpirationHandler) {
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.expirationHandler = handler

	// Register with FFI - ignore any errors as they are internal implementation details
	_ = a.registerExpirationHandler()
}

// GetExpirationHandler returns the currently set expiration handler.
//
// See Authenticator.SetExpirationHandler()
func (a *Authenticator) GetExpirationHandler() AuthenticationExpirationHandler {
	if a == nil {
		return nil
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.expirationHandler
}

// ObserveStatus registers an observer callback that will be called whenever authentication status changes.
//
// Returns a AuthenticationStatusObservationHandler that needs to be held as long as you want to receive the updates.
// Call AuthenticationStatusObservationHandler.Cancel()
func (a *Authenticator) ObserveStatus(observer AuthenticationStatusObservationHandler) *AuthenticationStatusObserver {
	if a == nil || observer == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	id := a.nextObserverID
	a.nextObserverID++

	if a.statusObservers == nil {
		a.statusObservers = make(map[uint64]AuthenticationStatusObservationHandler)
	}

	a.statusObservers[id] = observer

	// Immediately call with current status
	if a.currentStatus != nil {
		go observer(a.currentStatus)
	}

	return &AuthenticationStatusObserver{
		authenticator: a,
		id:            id,
	}
}

// Login logs into Ditto with a third-party token.
//
// The login has succeeded if the returned err is nil, or has failed if it is non-nil.
//
// The returned clientInfoJSON string will contain any JSON value returned by the auth webjook endpoint under the "clientInfo" key.
// This will be returned whether authentication succeeded or failed.
func (a *Authenticator) Login(token string, provider AuthenticationProvider) (clientInfoJSON string, err error) {
	clientInfoJSON, err = ffi.AuthClientLoginWithTokenAndFeedback(a.dittoHandle, token, provider.String())
	if err != nil {
		return
	}

	// Parse client info JSON if present
	var clientInfo map[string]any
	if clientInfoJSON != "" {
		if err := json.Unmarshal([]byte(clientInfoJSON), &clientInfo); err != nil {
			// Log the error but don't fail the login
			LogWarningF("Failed to parse client info JSON: %v", err)
		}
	}

	// Get the user ID after successful login
	a.mu.Lock()
	userID := ffi.AuthClientGetUserID(a.dittoHandle)
	a.mu.Unlock()

	// Update status to authenticated
	a.notifyStatusObservers(&AuthenticationStatus{
		IsAuthenticated: true,
		UserID:          userID,
		ClientInfo:      clientInfo,
	})

	return
}

// LoginWithCredentials authenticates with username and password.
//
// Returns nil on success, or an error on failure.
func (a *Authenticator) LoginWithCredentials(username, password string, provider AuthenticationProvider) (err error) {
	err = ffi.AuthClientLoginWithCredentials(a.dittoHandle, username, password, provider.String())
	if err != nil {
		return
	}

	// Get the user ID after successful login
	a.mu.Lock()
	userID := ffi.AuthClientGetUserID(a.dittoHandle)
	a.mu.Unlock()

	// Update status to authenticated
	a.notifyStatusObservers(&AuthenticationStatus{
		IsAuthenticated: true,
		UserID:          userID,
		ClientInfo:      nil,
	})

	return
}

// Logout logs out of Ditto.
//
// This will stop sync, shut down all replication sessions, and remove any cached authentication credentials.
// Use the optional cleanup closure to perform any required cleanup.
//
// Note that this does not remove any data from the store. The ∂itto instance returned in the cleanup callback will no
// longer be authorized for write transactions, however, data may be evicted.
//
// If cleanup is not nil, it will be called with the relevant Ditto instance as the sole argument that allows you to
// perform any required cleanup of the store as part of the logout process.
func (a *Authenticator) Logout(cleanup func(*Ditto)) {
	a.ditto.Sync().Stop()

	// Call FFI logout function
	err := ffi.AuthClientLogout(a.dittoHandle)
	if err != nil {
		LogErrorF("Logout failed: %v", err)
	}

	if cleanup != nil {
		cleanup(a.ditto)
	}

	a.notifyStatusObservers(&AuthenticationStatus{})
}

// notifyStatusObservers is called internally when authentication status changes
func (a *Authenticator) notifyStatusObservers(newStatus *AuthenticationStatus) {
	a.mu.Lock()
	a.currentStatus = newStatus
	observers := make([]AuthenticationStatusObservationHandler, 0, len(a.statusObservers))
	for _, observer := range a.statusObservers {
		observers = append(observers, observer)
	}
	a.mu.Unlock()

	// Call observers outside the lock to prevent deadlocks
	for _, observer := range observers {
		go observer(newStatus)
	}
}

// registerExpirationHandler registers the expiration handler with FFI
func (a *Authenticator) registerExpirationHandler() error {
	if a.expirationHandler == nil {
		return nil
	}

	// Create a login provider callback that bridges from FFI to our Go handler
	callback := func(secondsUntilExpiration uint32) {
		// Convert seconds to duration
		expiresIn := time.Duration(secondsUntilExpiration) * time.Second

		// Call the expiration handler
		if a.expirationHandler != nil {
			a.expirationHandler(a.ditto, expiresIn)
		}
	}

	// Register the login provider with FFI
	return ffi.SetLoginProvider(a.dittoHandle, callback)
}

// AuthenticationStatusObserver allows canceling a status observer.
type AuthenticationStatusObserver struct {
	authenticator *Authenticator
	id            uint64
}

// Cancel stops observation and cleans up all associated resources.
//
// After calling Cancel, the observer will no longer be notified of
// status changes.
func (h *AuthenticationStatusObserver) Cancel() {
	if h == nil || h.authenticator == nil {
		return
	}

	h.authenticator.mu.Lock()
	defer h.authenticator.mu.Unlock()

	delete(h.authenticator.statusObservers, h.id)
}
