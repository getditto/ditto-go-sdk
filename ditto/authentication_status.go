// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

// AuthenticationStatus represents the current authentication state.
type AuthenticationStatus struct {
	// IsAuthenticated is true if authenticated, otherwise false.
	IsAuthenticated bool

	// UserId is the user ID if authenticated and one was provided by the service, or otherwise an empty string.
	UserID string

	// ClientInfo returns additional information about the authenticated client.
	ClientInfo map[string]any
}
