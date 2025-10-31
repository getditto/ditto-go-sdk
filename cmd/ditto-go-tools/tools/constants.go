package tools

// Authentication system constants for Ditto SDK
const (
	// AuthSystemSmallPeersOnly is used for peer-to-peer synchronization without a server
	AuthSystemSmallPeersOnly = "small-peers-only"

	// AuthSystemOnlineWithAuthentication is used for server-based synchronization with custom authentication
	AuthSystemOnlineWithAuthentication = "online-with-authentication"

	// AuthSystemOnlinePlayground is used for server-based synchronization with playground authentication
	AuthSystemOnlinePlayground = "online-playground"

	// AuthSystemSharedSecret is used for server-based synchronization with shared secret authentication
	AuthSystemSharedSecret = "shared-secret"
)

// Peer scope constants for presence observation
const (
	// PeerScopeLocal shows only the local peer
	PeerScopeLocal = "local"

	// PeerScopeRemote shows only remote peers
	PeerScopeRemote = "remote"

	// PeerScopeAll shows both local and remote peers
	PeerScopeAll = "all"
)
