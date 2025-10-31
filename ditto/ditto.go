// Package ditto contains the public API for the Ditto Go SDK.
//
// The Ditto Go SDK is a cross-platform SDK that allows apps to sync with and even without Internet connectivity.
// Simply install the SDK into your application, then use the APIs to read and write data into its storage system,
// and it will then automatically sync any changes to other devices. Unlike other synchronization techniques,
// the Ditto Go SDK is designed for “peer-to-peer” synchronization where it can directly communicate with other devices,
// no server required! The SDK automatically manages the complexity of using multiple network transports, like
// Bluetooth and Wi-Fi, to find and connect to other devices and then synchronize any changes.
//
// If you are looking for an overview of how to use the Ditto Go SDK, see our guide documentation:
//
//   - [Ditto SDK Guide]
//
// [Ditto SDK Guide]: https://docs.ditto.live/sdk/latest/home
package ditto

// TODO(doc): Update the link above to point to Go-specific SDK documentation, when that is available.

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/getditto/ditto-go-sdk/internal/cbor"
	"github.com/getditto/ditto-go-sdk/internal/ffi"
)

func init() {
	ffi.InitSDKVersion(sdkVersion)
}

// DefaultRootDirectory returns the default root directory used for Ditto data persistence.
//
// On different platforms:
//   - macOS: Returns "~/Library/Application Support/Ditto" (avoids permission issues in CI)
//   - Linux/Windows: Returns the current working directory
//   - iOS/Android: Would return the app's data directory (not currently supported in Go)
//
// The macOS behavior matches the Swift SDK and ensures consistent permissions in CI
// environments where the current directory may have restricted access.
//
// This value is used when a relative persistence directory path (or empty string)
// is provided in the configuration.
//
// See also:
//   - DittoConfig.PersistenceDirectory
//   - Ditto.AbsolutePersistenceDirectory
func DefaultRootDirectory() string {
	return ffi.DefaultRootDirectory()
}

// Version returns the semantic version of the Ditto SDK.
//
// This represents the underlying Ditto core library version, not the Go SDK wrapper version.
func Version() string {
	return ffi.GetSDKSemver()
}

// Ditto is the entry point for accessing Ditto-related functionality.
type Ditto struct {
	mu            sync.RWMutex
	dittoHandle   *ffi.DittoHandle
	store         *Store
	sync          *Sync
	config        *DittoConfig
	presence      *Presence
	smallPeerInfo *SmallPeerInfo
	diskUsage     *DiskUsage // TODO(gosdk-tier3) Add DiskUsage() getter <https://linear.app/ditto/issue/SDKS-1702/go-sdk-disk-usage>
	authenticator *Authenticator

	// Device identification
	deviceName string

	// Internal state
	// TODO: check whether we need these, or if the state can be retrieved from FFI.
	// Also, make sure we use proper synchronization with mu
	closed     bool
	syncActive bool
	activated  bool
}

// Open creates and returns a new Ditto instance using the provided configuration.
//
// This is the recommended way to initialize Ditto. The method will validate the
// configuration and establish the necessary internal structures.
//
// Parameters:
//   - config: The configuration to initialize the new Ditto instance with
//
// Returns:
//   - The newly created Ditto instance
//   - An error if initialization fails
//
// Errors:
//   - if the chosen persistence directory is already in use by another Ditto instance
//   - if the passed-in DittoConfig's contents do not meet the required validation criteria
func Open(config *DittoConfig) (*Ditto, error) {
	ffi.InitLogger()

	// Validate config
	if config == nil {
		return nil, &DittoError{Code: ffi.ErrorCodeInternal, Message: "Invalid configuration"}
	}
	if config.DatabaseID == "" {
		return nil, &DittoError{Code: ffi.ErrorCodeInternal, Message: "DatabaseID is required"}
	}

	// Serialize config to CBOR
	configCBOR, err := config.toCBOR()
	if err != nil {
		return nil, &DittoError{Code: ffi.ErrorCodeInternal, Message: "Failed to serialize config", Err: err}
	}

	dittoHandle, err := ffi.DittoOpenThrows(
		configCBOR,
		ffi.TransportConfigModePlatformDependent,
		ffi.DefaultRootDirectory(),
	)
	if err != nil {
		return nil, convertFFIError(err)
	}

	d := &Ditto{
		dittoHandle:   dittoHandle,
		config:        config,
		presence:      newPresence(dittoHandle),
		smallPeerInfo: newSmallPeerInfo(dittoHandle),
	}
	d.store = newStore(d)
	d.sync = newSync(d)

	// Initialize authenticator if using server connection
	if config.Connect.Type == DittoConfigConnectTypeServer {
		d.authenticator = newAuthenticator(dittoHandle, d)
	}

	sdkDebugTrace("gosdk: ditto.Open opened successfully")

	return d, nil
}

// Config returns the configuration used to initialize this Ditto instance.
//
// NOTE: Sensitive data such as passphrases and private keys are redacted from the returned configuration and
// replaced with the string "[REDACTED]".
//
// Returns nil if any errors occur. Any error conditions would be internal errors, not expected in production code.
func (d *Ditto) Config() *DittoConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		LogError("gosdk: Config() failed: closed")
		return nil
	}

	// Get CBOR-encoded config from FFI
	cborData := ffi.GetDittoConfig(d.dittoHandle)
	if cborData == nil {
		LogError("gosdk: Config() failed: cborData is nil")
		return nil
	}

	// Decode CBOR to DittoConfig
	var config DittoConfig
	if err := cbor.Decode(cborData, &config); err != nil {
		LogError("gosdk: Config() failed: cbor.Decode() returned nil")
		return nil
	}

	return &config
}

// Store returns the Store instance for this Ditto instance.
func (d *Ditto) Store() *Store {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store
}

// Sync returns the ditto.Sync instance for this ditto.Ditto instance.
//
// The Sync component manages data synchronization with other peers, including:
//   - Starting and stopping sync
//   - Managing subscriptions
//   - Monitoring sync status
//
// Returns nil if the Ditto instance is closed.
func (d *Ditto) Sync() *Sync {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.sync
}

// Auth returns the Authenticator instance for this Ditto.
//
// The Authenticator manages authentication for server connections, including:
//   - Setting expiration handlers for token refresh
//   - Logging in with tokens or credentials
//   - Monitoring authentication status
//
// Returns nil if the Ditto instance is not using a server connection
// (e.g., when using DittoConfigConnectSmallPeersOnly).
func (d *Ditto) Auth() *Authenticator {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.authenticator
}

// TODO(gosdk-tier3): Implement TransportConfig <https://linear.app/ditto/issue/SDKS-1386/go-sdk-or-transport-config>

// TransportConfig returns the current transport configuration.
//
// The transport configuration controls how this instance discovers and connects
// to other peers, including:
//   - Peer-to-peer transports (Bluetooth LE, WiFi Direct, LAN)
//   - WebSocket connections
//   - Connection settings and limits
//
// The returned configuration is a snapshot; modifications to it will not affect
// the running instance unless explicitly set via SetTransportConfig.
//
// See also: SetTransportConfig, UpdateTransportConfig
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *Ditto) TransportConfig() *TransportConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return &TransportConfig{}
	}

	cborSlice, err := ffi.GetTransportConfig(d.dittoHandle)
	if err != nil {
		LogErrorF("Failed to get transport config: %v", err)
		return &TransportConfig{}
	}

	// Decode CBOR bytes to TransportConfig struct
	var config TransportConfig
	if err := cbor.Decode(cborSlice, &config); err != nil {
		LogErrorF("Failed to decode transport config: %v", err)
		return &TransportConfig{}
	}

	return &config
}

// TODO(gosdk-tier3): Implement UpdateTransportConfig <https://linear.app/ditto/issue/SDKS-1386/go-sdk-or-transport-config>

// UpdateTransportConfig safely updates the transport configuration using a callback.
//
// This method retrieves the current configuration, applies your modifications,
// and atomically updates the transport settings. This is the recommended way
// to modify transport configuration.
//
// Parameters:
//   - updateFn: A function that receives the current config and modifies it
//
// Example:
//
//	dittoInstance.UpdateTransportConfig(func(config *TransportConfig) {
//		// Enable only Bluetooth LE
//		config.PeerToPeer.BluetoothLE.Enabled = true
//		config.PeerToPeer.LAN.Enabled = false
//		config.PeerToPeer.AWDL.Enabled = false
//	})
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *Ditto) UpdateTransportConfig(updateFn func(*TransportConfig)) {
	sdkDebugTrace("gosdk: Ditto.UpdateTransportConfig() called")

	// Note: d.TransportConfig() and d.SetTransportConfig() will
	// lock d.mu, so we don't need to lock it here.

	config := d.TransportConfig()

	if updateFn != nil {
		updateFn(config)
	}

	configCBOR, err := config.ToCBOR()
	if err != nil {
		LogErrorF("Failed to encode transport config: %v", err)
		return
	}

	// Apply to FFI
	err = ffi.SetTransportConfig(d.dittoHandle, configCBOR)
	if err != nil {
		LogErrorF("Failed to set transport config: %v", err)
	}
}

// TODO(gosdk-tier3): Implement SetTransportConfig <https://linear.app/ditto/issue/SDKS-1386/go-sdk-or-transport-config>

// SetTransportConfig updates the transport configuration.
//
// This method replaces the entire transport configuration with the provided one.
// Changes take effect immediately if sync is running, or will be applied when
// sync is next started.
//
// Parameters:
//   - config: The new transport configuration to apply
//
// Returns:
//   - nil if successful
//   - ErrDittoClosed if the instance is closed
//   - An error if the configuration is invalid or cannot be applied
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *Ditto) SetTransportConfig(config *TransportConfig) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDittoClosed
	}

	// Convert config to CBOR - the CBORData method is already defined in types.go
	configCBOR, err := config.ToCBOR()
	if err != nil {
		return fmt.Errorf("failed to encode transport config: %w", err)
	}

	// Apply to FFI
	err = ffi.SetTransportConfig(d.dittoHandle, configCBOR)
	if err != nil {
		return fmt.Errorf("failed to set transport config: %w", err)
	}

	return nil
}

// Close shuts down the Ditto instance and releases all resources.
//
// This method:
//   - Stops all synchronization
//   - Cancels all active observers and subscriptions
//   - Closes all network connections
//   - Flushes pending data to disk
//   - Releases the database lock
//
// After calling Close(), the Ditto instance cannot be used again.
// Always call Close() when done with a Ditto instance to ensure
// proper cleanup and to allow other processes to access the database.
//
// Close is idempotent - calling it multiple times is safe.
func (d *Ditto) Close() {
	sdkDebugTrace("gosdk: ditto.Close() called")

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}

	if d.store != nil {
		d.store.cancelAllObservers()
	}

	if d.sync != nil {
		d.sync.cancelAllSubscriptions()
	}

	ffi.StopSync(d.dittoHandle)
	d.syncActive = false

	ffi.DittoClose(d.dittoHandle)

	d.closed = true

	sdkDebugTrace("gosdk: ditto.Close() exiting")
}

// IsClosed returns true if the Ditto instance has been closed.
//
// Once closed, a Ditto instance cannot be reopened and all operations
// will return a DittoError with ErrorCodeInternal.
func (d *Ditto) IsClosed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.closed
}

// SetOfflineOnlyLicenseToken activates an offline Ditto instance with a license token.
//
// You cannot sync with Ditto before activation. The offline license token is only
// valid for development and testing scenarios with OfflinePlayground, Manual, or
// SharedKey identities.
//
// License tokens can be obtained from the Ditto portal (https://portal.ditto.live).
//
// Parameters:
//   - token: The base64-encoded license token
//
// Returns:
//   - nil if activation successful
//   - ErrDittoClosed if the instance is closed
//   - An error if the token is invalid or expired
func (d *Ditto) SetOfflineOnlyLicenseToken(token string) error {
	sdkDebugTrace("gosdk: Ditto.SetOfflineOnlyLicenseToken() called")

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ErrDittoClosed
	}

	return ffi.SetOfflineOnlyLicenseToken(d.dittoHandle, token)
}

// Presence returns the Presence component for monitoring peer connections.
//
// The Presence API allows you to:
//   - Observe peers in the mesh network
//   - Monitor connection status and quality
//   - Track peer metadata and capabilities
//
// The returned Presence instance remains valid for the lifetime of the Ditto instance.
func (d *Ditto) Presence() *Presence {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.presence
}

// TODO(gosdk-tier2): Implement SmallPeerInfo <https://linear.app/ditto/issue/SDKS-1375/go-sdk-or-small-peer-info>

// SmallPeerInfo returns the SmallPeerInfo component for detailed sync information.
//
// SmallPeerInfo provides access to detailed information about sync state,
// including document synchronization progress and metadata.
//
// The returned SmallPeerInfo instance remains valid for the lifetime of the Ditto instance.
func (d *Ditto) SmallPeerInfo() *SmallPeerInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.smallPeerInfo
}

// DeviceName returns the human-readable name for this device.
//
// When using Presence.Observe(), each remote peer is represented by a device name.
// By default, this is a truncated version of the device's hostname.
//
// The device name is used for debugging and peer identification in the Ditto mesh.
// Names do not need to be unique among peers.
//
// See also: SetDeviceName
func (d *Ditto) DeviceName() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.deviceName == "" {
		// Get default device name from FFI if not set
		d.deviceName = ffi.GetDeviceName()
	}
	return d.deviceName
}

// SetDeviceName sets a custom identifier for this peer.
//
// The device name is visible to other peers in the mesh network and helps
// identify devices during debugging and monitoring.
//
// Important:
//   - Changes only take effect after restarting sync
//   - Names longer than 24 bytes will be truncated
//   - Names do not need to be unique
//
// Parameters:
//   - name: The new device name
func (d *Ditto) SetDeviceName(name string) {
	sdkDebugTrace("gosdk: Ditto.SetDeviceName() called")

	d.mu.Lock()
	defer d.mu.Unlock()

	// Update local cache
	d.deviceName = name

	if d.closed {
		return
	}

	if d.syncActive {
		// Log warning that changes take effect after sync restart
		LogWarning("gosdk: warning: changes to device name take effect when sync is restarted")
	}

	// Call FFI to set the device name
	if err := ffi.SetDeviceName(d.dittoHandle, name); err != nil {
		// In practice, this should never fail, so we don't force the caller to check results
		LogErrorF("gosdk: failed to set device name: %v", err)
	}
}

// IsActivated returns whether the SDK has been activated with a valid license token.
func (d *Ditto) IsActivated() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.activated {
		// Check activation status via FFI
		d.activated = ffi.IsActivated(d.dittoHandle)
	}
	return d.activated
}

// DatabaseID returns the application identifier for this Ditto instance.
//
// This is the database ID that was set when creating the Ditto configuration.
// It groups all peers that can synchronize with each other.
//
// Note: In older versions of the Ditto SDK, this was named "AppID".
//
// Returns an empty string if the instance is closed.
func (d *Ditto) DatabaseID() string {
	if config := d.Config(); config != nil {
		return config.DatabaseID
	}
	return ""
}

// setDevelopmentMode enables development mode with a built-in offline license.
//
// This method is a convenience for testing and development scenarios where
// you don't have a production license token. It automatically applies a
// development-only offline license.
//
// Warning: This should only be used for development and testing.
// Production applications must use proper license tokens.
//
// Parameters:
//   - enabled: true to enable development mode, false has no effect
//
// Returns:
//   - nil if successful or enabled is false
//   - An error if activation fails
func (d *Ditto) setDevelopmentMode(enabled bool) error {
	sdkDebugTraceF("gosdk: Ditto.setDevelopmentMode(%t) called", enabled)

	if !enabled {
		return nil
	}

	// Use a development offline-only license token
	// This is a standard development token that allows offline-only operation
	devToken := "o2d1c2VyX2lkeCR0aGlzLWlzLWEtZGV2ZWxvcG1lbnQtdG9rZW4tZm9yLXRlc3RpbmdqZXhwaXJ5X2RheXMZAA=="
	return d.SetOfflineOnlyLicenseToken(devToken)
}

// AbsolutePersistenceDirectory returns the absolute path to the persistence directory.
//
// This property returns the fully resolved path where Ditto stores its data,
// taking into account any relative paths or defaults specified in the configuration.
//
// The value depends on what was specified in DittoConfig.PersistenceDirectory:
//   - Absolute path: returned unchanged
//   - Relative path: resolved relative to DefaultRootDirectory()
//   - Empty string: defaults to "{root}/ditto-{database-id}"
//
// It is not recommended to directly read or write to this directory as its
// structure is managed by Ditto and may change between versions.
//
// Returns an empty string if the instance is closed.
func (d *Ditto) AbsolutePersistenceDirectory() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return ""
	}

	return ffi.GetAbsolutePersistenceDirectory(d.dittoHandle)
}

// TransportDiagnostics returns detailed diagnostic information about network transports.
//
// This method provides comprehensive runtime statistics and debugging information about
// all configured transports, including:
//   - Transport enable/disable status
//   - Active connections and their states
//   - Connection durations and peer identities
//   - Transport-specific performance metrics
//   - Error conditions and warnings
//   - Peer discovery status
//
// The diagnostics cover all transport types:
//   - Bluetooth LE: Connection count, scanning status, power state
//   - LAN: Network interfaces, multicast/mDNS discovery status
//   - P2P WiFi: AWDL/Wi-Fi Aware availability and active sessions
//   - WebSocket: Server connections, retry attempts, latency
//
// This information is essential for:
//   - Debugging connectivity issues
//   - Monitoring network health
//   - Understanding sync performance
//   - Troubleshooting peer discovery problems
//
// Returns:
//   - *TransportDiagnostics: Current transport status and metrics
//   - ErrDittoClosed if the instance is closed
//   - error: If diagnostics cannot be retrieved
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *Ditto) TransportDiagnostics() (*TransportDiagnostics, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return nil, ErrDittoClosed
	}

	// Get diagnostics from FFI (returns JSON)
	jsonData, err := ffi.GetTransportDiagnostics(d.dittoHandle)
	if err != nil {
		return nil, err
	}

	// Validate that we received valid JSON data
	if len(jsonData) == 0 {
		return nil, fmt.Errorf("received empty JSON data from transport diagnostics")
	}

	// Parse the JSON data into TransportDiagnostics struct
	var diagnostics TransportDiagnostics
	if err := json.Unmarshal(jsonData, &diagnostics); err != nil {
		return nil, fmt.Errorf("failed to parse transport diagnostics JSON: %w", err)
	}

	// Initialize the Transports slice if it's nil to ensure consistent behavior
	if diagnostics.Transports == nil {
		diagnostics.Transports = make([]TransportSnapshot, 0)
	}

	// Ensure all transport snapshots have properly initialized slices
	for i := range diagnostics.Transports {
		transport := &diagnostics.Transports[i]
		if transport.Connecting == nil {
			transport.Connecting = make([]int64, 0)
		}
		if transport.Connected == nil {
			transport.Connected = make([]int64, 0)
		}
		if transport.Disconnecting == nil {
			transport.Disconnecting = make([]int64, 0)
		}
		if transport.Disconnected == nil {
			transport.Disconnected = make([]int64, 0)
		}
	}

	return &diagnostics, nil
}

// IsEncrypted indicates whether the Ditto data is encrypted.
// Returns false if the Ditto instance is closed.
func (d *Ditto) IsEncrypted() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.closed {
		return false
	}

	return ffi.IsEncrypted(d.dittoHandle)
}
