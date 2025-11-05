package ditto

import (
	"github.com/getditto/ditto-go-sdk/v5/internal/cbor"
)

// Document represents a document stored in a Ditto collection.
//
// It is a map of string keys to JSON values.
type Document map[string]any

// TransportConfig controls how Ditto connects to and communicates with other peers.
//
// The transport system supports multiple connection methods that can work simultaneously:
//   - Peer-to-peer: Direct device connections via Bluetooth, WiFi Direct, LAN
//   - Cloud: WebSocket connections to Ditto Cloud servers
//   - Custom servers: TCP connections to self-hosted infrastructure
//
// Example:
//
//	config := ditto.NewTransportConfig()
//	config.PeerToPeer.BluetoothLE.Enabled = true
//	config.PeerToPeer.LAN.Enabled = true
//	config.Connect.WebsocketURLs = []string{"wss://cloud.ditto.live"}
//	err := dittoInstance.SetTransportConfig(config)
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type TransportConfig struct {
	// PeerToPeer configures direct device-to-device transports
	PeerToPeer PeerToPeerConfig `cbor:"peer_to_peer" json:"peer_to_peer"`

	// Connect configures outgoing connections to servers
	Connect ConnectTransport `cbor:"connect" json:"connect"`

	// Listen configures this device to accept incoming connections
	Listen ListenConfig `cbor:"listen" json:"listen"`

	// Global contains global transport settings
	Global GlobalConfig `cbor:"global" json:"global"`
}

// ToCBOR encodes the TransportConfig to CBOR format for FFI.
//
// This method is used internally to serialize the configuration
// for the Ditto core library.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (tc *TransportConfig) ToCBOR() ([]byte, error) {
	return cbor.Encode(tc)
}

// PeerToPeerConfig configures direct peer-to-peer communication transports.
//
// These transports enable devices to sync directly without internet connectivity.
// Multiple transports can be enabled simultaneously for maximum connectivity.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type PeerToPeerConfig struct {
	// BluetoothLE enables Bluetooth Low Energy transport (all platforms)
	BluetoothLE BluetoothLEConfig `cbor:"bluetooth_le" json:"bluetooth_le"`

	// LAN enables Local Area Network transport via TCP/IP
	LAN LANConfig `cbor:"lan" json:"lan"`

	// AWDL enables Apple Wireless Direct Link (iOS/macOS only)
	AWDL AWDLConfig `cbor:"awdl" json:"awdl"`

	// WifiAware enables Wi-Fi Aware transport (Android only)
	WifiAware WifiAwareConfig `cbor:"wifi_aware" json:"wifi_aware"`
}

// BluetoothLEConfig configures Bluetooth Low Energy transport.
//
// Bluetooth LE provides:
//   - Low power consumption
//   - Works in background on mobile devices
//   - Range of approximately 10-30 meters
//   - No pairing required
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type BluetoothLEConfig struct {
	// Enabled controls whether Bluetooth LE transport is active
	Enabled bool `cbor:"enabled" json:"enabled"`
}

// LANConfig configures Local Area Network transport.
//
// LAN transport enables high-speed sync between devices on the same network.
// It uses TCP/IP for reliable data transfer and supports both multicast
// discovery and mDNS (Bonjour) for finding peers.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type LANConfig struct {
	// Enabled controls whether LAN transport is active
	Enabled bool `cbor:"enabled" json:"enabled"`

	// MDNSEnabled allows peer discovery via mDNS/Bonjour
	MDNSEnabled bool `cbor:"mdns_enabled" json:"mdns_enabled"`

	// MulticastEnabled allows peer discovery via multicast packets
	MulticastEnabled bool `cbor:"multicast_enabled" json:"multicast_enabled"`
}

// AWDLConfig configures Apple Wireless Direct Link transport.
//
// AWDL is Apple's proprietary peer-to-peer WiFi protocol used for:
//   - AirDrop
//   - AirPlay
//   - High-speed local transfers
//
// Only available on iOS and macOS devices.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type AWDLConfig struct {
	// Enabled controls whether AWDL transport is active
	Enabled bool `cbor:"enabled" json:"enabled"`
}

// WifiAwareConfig configures Wi-Fi Aware transport.
//
// Wi-Fi Aware (NAN - Neighbor Awareness Networking) enables:
//   - Direct WiFi connections without an access point
//   - Low-latency discovery of nearby devices
//   - High-bandwidth data transfer
//
// Only available on Android 8.0+ devices with hardware support.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type WifiAwareConfig struct {
	// Enabled controls whether Wi-Fi Aware transport is active
	Enabled bool `cbor:"enabled" json:"enabled"`
}

// ConnectTransport configures outgoing connections to Ditto Cloud or custom servers.
//
// This enables synchronization through internet infrastructure when
// peer-to-peer connections are not available or sufficient.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type ConnectTransport struct {
	// TCPServers lists TCP server addresses to connect to (e.g., "192.168.1.100:4040")
	TCPServers []string `cbor:"tcp_servers" json:"tcp_servers"`

	// WebsocketURLs lists WebSocket endpoints for cloud sync (e.g., "wss://cloud.ditto.live")
	WebsocketURLs []string `cbor:"websocket_urls" json:"websocket_urls"`

	// RetryInterval specifies milliseconds between connection retry attempts (default: 5000)
	RetryInterval int `cbor:"retry_interval" json:"retry_interval"`
}

// ListenConfig configures this device to accept incoming connections.
//
// This is typically used for:
//   - Creating a local sync server
//   - Bridging between network segments
//   - Debugging and monitoring
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type ListenConfig struct {
	// TCP configures raw TCP listening
	TCP ListenTCPConfig `cbor:"tcp" json:"tcp"`

	// HTTP configures HTTP/WebSocket listening
	HTTP ListenHTTPConfig `cbor:"http" json:"http"`
}

// ListenTCPConfig configures TCP server listening.
//
// When enabled, other Ditto instances can connect to this device
// using the TCP transport.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type ListenTCPConfig struct {
	// Enabled controls whether to listen for TCP connections
	Enabled bool `cbor:"enabled" json:"enabled"`

	// InterfaceIP is the network interface to bind to (default: "[::]" for all interfaces)
	InterfaceIP string `cbor:"interface_ip" json:"interface_ip"`

	// Port is the TCP port to listen on (default: 4040)
	Port int `cbor:"port" json:"port"`
}

// ListenHTTPConfig configures HTTP/WebSocket server listening.
//
// This enables other devices to connect via WebSocket for synchronization
// and optionally serves static content for web applications.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type ListenHTTPConfig struct {
	// Enabled controls whether to listen for HTTP connections
	Enabled bool `cbor:"enabled" json:"enabled"`

	// InterfaceIP is the network interface to bind to (default: "[::]" for all interfaces)
	InterfaceIP string `cbor:"interface_ip" json:"interface_ip"`

	// Port is the HTTP port to listen on (default: 80)
	Port int `cbor:"port" json:"port"`

	// WebsocketSync enables WebSocket connections for data sync
	WebsocketSync bool `cbor:"websocket_sync" json:"websocket_sync"`

	// StaticContentPath serves static files from this directory (optional)
	StaticContentPath string `cbor:"static_content_path,omitempty" json:"static_content_path,omitempty"`

	// TLSKeyPath is the path to the TLS private key file (optional, enables HTTPS)
	TLSKeyPath string `cbor:"tls_key_path,omitempty" json:"tls_key_path,omitempty"`

	// TLSCertificatePath is the path to the TLS certificate file (optional, enables HTTPS)
	TLSCertificatePath string `cbor:"tls_certificate_path,omitempty" json:"tls_certificate_path,omitempty"`
}

// GlobalConfig contains global transport settings that affect all transports.
//
// These settings control advanced routing and grouping behaviors.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type GlobalConfig struct {
	// SyncGroup partitions the mesh network (0 = default group)
	// Devices only sync with others in the same sync group
	SyncGroup uint32 `cbor:"sync_group" json:"sync_group"`

	// RoutingHint provides a hint for connection routing optimization
	RoutingHint uint32 `cbor:"routing_hint" json:"routing_hint"`
}

// TransportDiagnostics provides detailed information about all transport status.
//
// This information is useful for:
//   - Debugging connectivity issues
//   - Monitoring network health
//   - Understanding sync performance
//
// Obtain diagnostics via Ditto.TransportDiagnostics().
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type TransportDiagnostics struct {
	// Transports contains diagnostics for each transport type
	Transports []TransportSnapshot `json:"transports"`
}

// TotalConnectedPeers returns the total number of connected peers across all transports.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (td *TransportDiagnostics) TotalConnectedPeers() int {
	total := 0
	for _, transport := range td.Transports {
		total += len(transport.Connected)
	}
	return total
}

// ActiveTransports returns the number of transports that have at least one connected peer.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (td *TransportDiagnostics) ActiveTransports() int {
	active := 0
	for _, transport := range td.Transports {
		if len(transport.Connected) > 0 {
			active++
		}
	}
	return active
}

// TransportByType returns the transport snapshot for the specified connection type,
// or nil if no transport of that type is found.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (td *TransportDiagnostics) TransportByType(connectionType string) *TransportSnapshot {
	for i := range td.Transports {
		if td.Transports[i].ConnectionType == connectionType {
			return &td.Transports[i]
		}
	}
	return nil
}

// TransportSnapshot provides diagnostic information about a specific transport.
//
// Each transport type (TCP, mDNS, Bluetooth, etc.) has its own snapshot
// showing which peers are in various connection states.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type TransportSnapshot struct {
	// ConnectionType is the type of transport (e.g., "TCP", "mDNS", "Bluetooth")
	ConnectionType string `json:"connection_type"`

	// Connecting contains site IDs of peers currently connecting
	Connecting []int64 `json:"connecting"`

	// Connected contains site IDs of peers currently connected
	Connected []int64 `json:"connected"`

	// Disconnecting contains site IDs of peers currently disconnecting
	Disconnecting []int64 `json:"disconnecting"`

	// Disconnected contains site IDs of peers that have disconnected
	Disconnected []int64 `json:"disconnected"`
}

// TotalPeers returns the total number of peers across all connection states.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (ts *TransportSnapshot) TotalPeers() int {
	return len(ts.Connecting) + len(ts.Connected) + len(ts.Disconnecting) + len(ts.Disconnected)
}

// IsActive returns true if this transport has any connected peers.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (ts *TransportSnapshot) IsActive() bool {
	return len(ts.Connected) > 0
}

// HasActivity returns true if this transport has any peer activity (connecting, connected, or disconnecting).
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (ts *TransportSnapshot) HasActivity() bool {
	return len(ts.Connecting) > 0 || len(ts.Connected) > 0 || len(ts.Disconnecting) > 0
}

// TODO: This NewTransportConfig() may not return the right result.
// Compare with other SDKs' implementations.

// NewTransportConfig creates a new TransportConfig with safe defaults.
//
// By default, all transports are disabled. You must explicitly enable
// the transports you want to use.
//
// Common patterns:
//
//	// Enable all peer-to-peer transports
//	config := ditto.NewTransportConfig().EnableAllPeerToPeer()
//
//	// Enable specific transports
//	config := ditto.NewTransportConfig()
//	config.PeerToPeer.BluetoothLE.Enabled = true
//	config.PeerToPeer.LAN.Enabled = true
//
//	// Add cloud sync
//	config.AddWebsocketURL("wss://cloud.ditto.live")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewTransportConfig() *TransportConfig {
	return &TransportConfig{
		PeerToPeer: PeerToPeerConfig{
			BluetoothLE: BluetoothLEConfig{Enabled: false},
			LAN: LANConfig{
				Enabled:          false,
				MDNSEnabled:      true,
				MulticastEnabled: true,
			},
			AWDL:      AWDLConfig{Enabled: false},
			WifiAware: WifiAwareConfig{Enabled: false},
		},
		Connect: ConnectTransport{
			TCPServers:    []string{},
			WebsocketURLs: []string{},
			RetryInterval: 5000,
		},
		Listen: ListenConfig{
			TCP: ListenTCPConfig{
				Enabled:     false,
				InterfaceIP: "[::]",
				Port:        4040,
			},
			HTTP: ListenHTTPConfig{
				Enabled:       false,
				InterfaceIP:   "[::]",
				Port:          80,
				WebsocketSync: true,
			},
		},
		Global: GlobalConfig{
			SyncGroup:   0,
			RoutingHint: 0,
		},
	}
}

// EnableAllPeerToPeer enables all available peer-to-peer transports.
//
// This is a convenience method that enables:
//   - Bluetooth LE
//   - LAN (with mDNS and multicast)
//   - AWDL (on Apple platforms)
//   - Wi-Fi Aware (on Android)
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config := ditto.NewTransportConfig().
//		EnableAllPeerToPeer().
//		AddWebsocketURL("wss://cloud.ditto.live")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) EnableAllPeerToPeer() *TransportConfig {
	t.PeerToPeer.BluetoothLE.Enabled = true
	t.PeerToPeer.LAN.Enabled = true
	t.PeerToPeer.AWDL.Enabled = true
	t.PeerToPeer.WifiAware.Enabled = true
	return t
}

// AddWebsocketURL adds a WebSocket URL for cloud synchronization.
//
// Multiple URLs can be added for redundancy. Ditto will attempt to
// connect to all provided URLs.
//
// Parameters:
//   - url: WebSocket URL (must start with ws:// or wss://)
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config.AddWebsocketURL("wss://primary.ditto.live").
//	      AddWebsocketURL("wss://backup.ditto.live")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) AddWebsocketURL(url string) *TransportConfig {
	t.Connect.WebsocketURLs = append(t.Connect.WebsocketURLs, url)
	return t
}

// SetWebsocketURLs replaces all WebSocket URLs with the provided list.
//
// This overwrites any previously configured WebSocket URLs.
//
// Parameters:
//   - urls: List of WebSocket URLs to use
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config.SetWebsocketURLs([]string{
//		"wss://us-east.ditto.live",
//		"wss://eu-west.ditto.live",
//	})
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) SetWebsocketURLs(urls []string) *TransportConfig {
	t.Connect.WebsocketURLs = urls
	return t
}

// EnableTCPListen configures this device to accept incoming TCP connections.
//
// This is useful for creating a local sync server or bridge device that
// other peers can connect to directly.
//
// Parameters:
//   - interfaceIP: Network interface to bind to ("[::]" for all interfaces)
//   - port: TCP port to listen on (default 4040)
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	// Listen on all interfaces, port 4040
//	config.EnableTCPListen("[::]", 4040)
//
//	// Listen only on specific interface
//	config.EnableTCPListen("192.168.1.100", 4040)
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) EnableTCPListen(interfaceIP string, port int) *TransportConfig {
	t.Listen.TCP.Enabled = true
	t.Listen.TCP.InterfaceIP = interfaceIP
	t.Listen.TCP.Port = port
	return t
}

// EnableHTTPListen configures this device to accept HTTP/WebSocket connections.
//
// This enables other devices to sync via WebSocket and optionally serves
// static content for web applications.
//
// Parameters:
//   - interfaceIP: Network interface to bind to ("[::]" for all interfaces)
//   - port: HTTP port to listen on (default 80, or 443 with TLS)
//   - enableWebsocket: Whether to enable WebSocket sync
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	// Basic HTTP server with WebSocket sync
//	config.EnableHTTPListen("[::]", 8080, true)
//
//	// HTTPS server (requires separate TLS configuration)
//	config.EnableHTTPListen("[::]", 443, true).
//		SetHTTPTLS("/path/to/cert.pem", "/path/to/key.pem")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) EnableHTTPListen(interfaceIP string, port int, enableWebsocket bool) *TransportConfig {
	t.Listen.HTTP.Enabled = true
	t.Listen.HTTP.InterfaceIP = interfaceIP
	t.Listen.HTTP.Port = port
	t.Listen.HTTP.WebsocketSync = enableWebsocket
	return t
}

// SetHTTPTLS configures TLS certificates for HTTPS listening.
//
// This enables secure WebSocket connections (wss://) to this device.
//
// Parameters:
//   - certPath: Path to the TLS certificate file (PEM format)
//   - keyPath: Path to the TLS private key file (PEM format)
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config.EnableHTTPListen("[::]", 443, true).
//		SetHTTPTLS("/etc/ssl/server.crt", "/etc/ssl/server.key")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) SetHTTPTLS(certPath, keyPath string) *TransportConfig {
	t.Listen.HTTP.TLSCertificatePath = certPath
	t.Listen.HTTP.TLSKeyPath = keyPath
	return t
}

// SetStaticContentPath configures the HTTP server to serve static files.
//
// This is useful for serving web applications that use Ditto's WebSocket
// sync capabilities.
//
// Parameters:
//   - path: Directory path containing static files to serve
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config.EnableHTTPListen("[::]", 8080, true).
//		SetStaticContentPath("/var/www/html")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) SetStaticContentPath(path string) *TransportConfig {
	t.Listen.HTTP.StaticContentPath = path
	return t
}

// AddTCPServer adds a TCP server address for outgoing connections.
//
// The device will attempt to connect to this server for synchronization.
//
// Parameters:
//   - address: Server address in format "host:port" (e.g., "192.168.1.100:4040")
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config.AddTCPServer("192.168.1.100:4040").
//		AddTCPServer("backup.local:4040")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) AddTCPServer(address string) *TransportConfig {
	t.Connect.TCPServers = append(t.Connect.TCPServers, address)
	return t
}

// SetSyncGroup sets the sync group for mesh partitioning.
//
// Devices only sync with others in the same sync group. This is useful for:
//   - Multi-tenant applications
//   - Testing isolated mesh networks
//   - Gradual rollouts
//
// Parameters:
//   - group: Sync group ID (0 = default group)
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	// Isolate test devices in group 42
//	config.SetSyncGroup(42)
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) SetSyncGroup(group uint32) *TransportConfig {
	t.Global.SyncGroup = group
	return t
}

// DisableAllPeerToPeer disables all peer-to-peer transports.
//
// This forces the device to only sync through cloud or custom servers.
// Useful for:
//   - Cloud-only deployments
//   - Debugging server connections
//   - Reducing battery usage on mobile devices
//
// Returns the same TransportConfig for method chaining.
//
// Example:
//
//	config := ditto.NewTransportConfig().
//		DisableAllPeerToPeer().
//		AddWebsocketURL("wss://cloud.ditto.live")
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (t *TransportConfig) DisableAllPeerToPeer() *TransportConfig {
	t.PeerToPeer.BluetoothLE.Enabled = false
	t.PeerToPeer.LAN.Enabled = false
	t.PeerToPeer.AWDL.Enabled = false
	t.PeerToPeer.WifiAware.Enabled = false
	return t
}
