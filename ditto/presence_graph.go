package ditto

import (
	"encoding/json"
)

// PresenceGraph represents the Ditto mesh network of peers and their connections between each other.
//
// The local peer is the entry point, all others are remote peers known by the local peer (either directly or via
// other remote peers).
type PresenceGraph struct {
	// LocalPeer returns the local peer (usually the peer that is represented by the currently running Ditto instance).
	// The LocalPeer is the entry point, all others are remote peers known by the local peer (either directly or via other
	// remote peers).
	LocalPeer *Peer `json:"localPeer,omitempty"`

	// RemotePeers is all remote peers known by the localPeer, either directly or via other remote peers.
	RemotePeers []*Peer `json:"remotePeers,omitempty"`

	// AllConnectionsByID is a dictionary with all connections found in this graph by their IDs.
	AllConnectionsByID map[string]*Connection `json:"-"`
}

// ToJSON converts the graph to JSON
func (g *PresenceGraph) ToJSON() ([]byte, error) {
	data := map[string]any{
		"localPeer":   g.LocalPeer,
		"remotePeers": g.RemotePeers,
		"connections": g.AllConnectionsByID,
	}
	return json.Marshal(data)
}

// Peer represents a peer in a Ditto mesh network
type Peer struct {
	// Address uniquely identifies a peer within a Ditto mesh network.
	//
	// Deprecated: use PeerKeyString instead.
	Address Address

	// PeerKeyString is a unique identifier for a given peer, equal to or derived from the cryptographic public key used to authenticate it.
	//
	//  NOTE: This will be nil or empty when a peer is not updated to the latest version of the SDK.
	PeerKeyString string `json:"peerKeyString"`

	// Connections is the set of currently active connections of the peer. May be nil or empty.
	Connections []*Connection `json:"connections"`

	// DeviceName is the human-readable device name of the remote peer.
	//
	// This defaults to the hostname but can be manually set by the application developer of the other peer. It is not necessarily unique.
	DeviceName string `json:"deviceName"`

	// OS is the operating system of the remote peer.
	//
	// Note: When running on a Mac in the Mac Catalyst environment, the value returned will be iOS.
	OS PeerOS `json:"os,omitempty"`

	// IsConnectedToDittoCloud indicates whether the peer is connected to Ditto Cloud
	IsConnectedToDittoCloud bool `json:"isConnectedToDittoCloud"`

	// IsCompatible indicates whether the peer is compatible with the current peer.
	IsCompatible bool `json:"isCompatible"`

	// DittoSDKVersion is the Ditto SDK version the peer is running with.
	DittoSDKVersion string `json:"dittoSdkVersion"`

	// PeerMetadata is metadata associated with the peer, nil by default.
	//
	// Use Presence.SetPeerMetadata() Presence.SetPeerMetadataJSONData() to set this value.
	//
	// Peer metadata is dynamic and may change over the lifecycle of the DittoPresenceGraph. It may be nil or empty when
	// a remote peer initially appears in the presence graph and will be updated once the peer has synced its metadata
	// with the local peer.
	//
	// See also: Presence.SetPeerMetadata() for details on usage of metadata
	PeerMetadata map[string]any `json:"peerMetadata"`

	// IdentityServiceMetadata is metadata associated with the peer by the identity service. May be nil or empty.
	//
	// Use an authentication webhook to set this value. See Ditto’s online documentation for more information on how to
	// configure an authentication webhook.
	IdentityServiceMetadata map[string]any `json:"identityServiceMetadata"`
}

// Connection represents an active network connection between two peers in a Ditto mesh network
type Connection struct {
	// ID is a unique identifier for the connection.
	//
	// This ID is deterministic for any two peers and a given connection type.
	ID string `json:"id"`

	// PeerKeyString1 is the peer key of the peer at one end of the connection, as a string.
	//
	// The assignment to PeerKeyString1 and |eerKeyString2 is deterministic and stable for any two peers.
	PeerKeyString1 string `json:"peerKeyString1"`

	// PeerKeyString2 is the peer key of the peer at the other end of the connection, as a string.
	//
	// The assignment to PeerKeyString1 and PeerKeyString2 is deterministic and stable for any two peers.
	PeerKeyString2 string `json:"peerKeyString2"`

	// Type is the type of transport enabling this connection.
	Type ConnectionType `json:"connectionType"`
}

// ConnectionType is the type of Connection between two Peers signaling what transport is being used for it.
type ConnectionType string

const (
	// ConnectionTypeBluetooth represents Bluetooth Low Energy connections
	ConnectionTypeBluetooth ConnectionType = "Bluetooth"

	// ConnectionTypeAccessPoint represents LAN connections via WiFi/Ethernet
	ConnectionTypeAccessPoint ConnectionType = "AccessPoint"

	// ConnectionTypeP2PWiFi represents direct WiFi connections (AWDL, Wi-Fi Aware)
	ConnectionTypeP2PWiFi ConnectionType = "P2PWiFi"

	// ConnectionTypeWebSocket represents cloud or server-mediated connections
	ConnectionTypeWebSocket ConnectionType = "WebSocket"
)

// Address represents a peer's network address
type Address struct {
	AddressType AddressType
	IPAddress   string
	Port        uint16
}

// AddressType represents the type of address
type AddressType int

const (
	// AddressTypeIPv4 represents IPv4 addresses
	AddressTypeIPv4 AddressType = iota

	// AddressTypeIPv6 represents IPv6 addresses
	AddressTypeIPv6

	// AddressTypeBluetooth represents Bluetooth addresses
	AddressTypeBluetooth
)

// PeerOS represents the operating system of a peer
//
// Note: When running on a Mac in the Mac Catalyst environment, the OS is classified as iOS
type PeerOS string

const (
	// PeerOSUnknown is returned when a peer's operating system has not been determined
	PeerOSUnknown PeerOS = ""

	// PeerOSGeneric represents a general other operating system
	PeerOSGeneric PeerOS = "Generic"

	// PeerOSiOS represents Apple iOS devices (iPhone, iPad)
	PeerOSiOS PeerOS = "iOS"

	// PeerOStvOS represents Apple tvOS devices (Apple TV)
	PeerOStvOS PeerOS = "tvOS"

	// PeerOSAndroid represents Android devices
	PeerOSAndroid PeerOS = "Android"

	// PeerOSLinux represents Linux systems
	PeerOSLinux PeerOS = "Linux"

	// PeerOSWindows represents Microsoft Windows systems
	PeerOSWindows PeerOS = "Windows"

	// PeerOSmacOS represents Apple macOS systems
	PeerOSmacOS PeerOS = "macOS"
)
