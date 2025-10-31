package ditto

import (
	"github.com/getditto/ditto-go-sdk/internal/cbor"
	"github.com/getditto/ditto-go-sdk/internal/ffi"
)

// DittoConfig is a configuration object for initializing a Ditto instance.
//
// DittoConfig encapsulates all the parameters required to configure a Ditto instance, including identity,
// connectivity, and persistence.
type DittoConfig struct {
	// DatabaseID is the unique identifier for the Ditto database.
	//
	// This must be a valid UUID string. You can find the database ID in the
	// Ditto portal, or provide your own if you only need to sync with
	// small peers only.
	//
	// Note: "Database ID" was previously referred to as "App ID" in older
	// versions of the Ditto SDK.
	DatabaseID string `cbor:"database_id" json:"databaseId"`

	// Connect specifies how this instance discovers and connects to peers.
	// This includes network settings and authentication options.
	Connect DittoConfigConnectUnion `cbor:"connect" json:"connect"`

	// PersistenceDirectory specifies where Ditto should persist data.
	//
	// Accepts:
	//   - Absolute path: "/path/to/ditto-data"
	//   - Relative path: "./ditto-data" (relative to DefaultRootDirectory)
	//   - Empty string: uses default "{root}/ditto-{database-id}"
	//
	// It is not recommended to directly read or write to this directory.
	PersistenceDirectory string `cbor:"persistence_directory,omitempty" json:"persistenceDirectory,omitempty"`

	// Experimental allows setting configuration for experimental features.
	Experimental map[string]any `cbor:"experimental" json:"experimental"`
}

type DittoConfigConnectUnion struct {
	Type DittoConfigConnectType `cbor:"type" json:"type"`

	*DittoConfigConnectServer         `cbor:",omitempty" json:",omitempty"`
	*DittoConfigConnectSmallPeersOnly `cbor:",omitempty" json:",omitempty"`
}

// DefaultDittoConfig returns a default DittoConfig instance with standard settings.
//
// This provides a quick starting point for development, but production
// applications should customize the configuration as needed.
func DefaultDittoConfig() *DittoConfig {
	// Get default config from FFI
	configBytes := ffi.GetDefaultConfig()

	// Check if we got empty bytes - this is an error condition
	if len(configBytes) == 0 {
		LogError("FFI returned empty config bytes")
		return &DittoConfig{}
	}

	config := &DittoConfig{}
	if err := cbor.Decode(configBytes, &config); err != nil {
		LogErrorF("Failed to decode CBOR config: %v", err)
		return &DittoConfig{}
	}

	// Set default database ID if not present
	if config.DatabaseID == "" {
		config.DatabaseID = DefaultDatabaseID()
	}

	// Ensure the Connect struct for the type is present.
	// If no fields are present, then the decoder will leave the field nil.
	switch config.Connect.Type {
	case DittoConfigConnectTypeServer:
		if config.Connect.DittoConfigConnectServer == nil {
			config.Connect.DittoConfigConnectServer = &DittoConfigConnectServer{}
		}
	case DittoConfigConnectTypeSmallPeersOnly:
		if config.Connect.DittoConfigConnectSmallPeersOnly == nil {
			config.Connect.DittoConfigConnectSmallPeersOnly = &DittoConfigConnectSmallPeersOnly{}
		}
	}

	return config
}

// DefaultDatabaseID returns the default identifier for a Ditto instance.
//
// This identifier is used when no explicit Database ID is provided.
// It is typically a constant UUID string that ensures a predictable default value.
// In most cases, you should provide your own unique identifier to avoid
// conflicts when syncing with other peers.
func DefaultDatabaseID() string {
	return ffi.GetDefaultDatabaseID()
}

type DittoConfigConnectType string

const (
	DittoConfigConnectTypeServer         DittoConfigConnectType = "server"
	DittoConfigConnectTypeSmallPeersOnly DittoConfigConnectType = "small_peers_only"
)

// DittoConfigConnect is an interface for different connection configurations.
//
// Implementations determine how a Ditto instance connects to other peers:
//   - [DittoConfigConnectSmallPeersOnly]: For peer-to-peer only scenarios
//   - [DittoConfigConnectServer]: For production Ditto Cloud connections
type DittoConfigConnect interface {
	connectConfig() // private marker method
}

// DittoConfigConnectServer connects a Ditto instance to a Big Peer at the specified URL.
//
// IMPORTANT: For sync to work with server connections, Ditto requires
//
//   - (a) an [AuthenticationExpirationHandler] to be set via [Authenticator.SetExpirationHandler], and
//   - (b) that handler to properly authenticate when requested. [Sync.Start] will return an error if the expiration handler is nil.
type DittoConfigConnectServer struct {
	// URL is the server endpoint for cloud sync
	URL string `cbor:"url" json:"url"`
}

func (c DittoConfigConnectServer) connectConfig() {}

// DittoConfigConnectSmallPeersOnly restricts connectivity to small peers only, optionally using a shared secret
// (in the form of a private key) for authentication.
//
// If a PrivateKey is provided, it will be used as a shared secret for authenticating peer-to-peer connections.
// The default value is nil, which means no encryption is used in transit.
type DittoConfigConnectSmallPeersOnly struct {
	// PrivateKey is an optional shared secret for mesh encryption.
	// When provided, only peers with the same key can sync together.
	PrivateKey []byte `cbor:"private_key,omitempty" json:"privateKey,omitempty"`
}

func (c DittoConfigConnectSmallPeersOnly) connectConfig() {}

// WithDatabaseID sets the database ID and returns the config for chaining.
//
// This is a fluent interface method that allows method chaining.
//
// Parameters:
//   - id: The database ID (UUID string)
//
// Returns:
//   - The same [DittoConfig] instance for chaining
func (c *DittoConfig) WithDatabaseID(id string) *DittoConfig {
	c.DatabaseID = id
	return c
}

// WithPersistenceDirectory sets the persistence directory and returns the config for chaining.
//
// Parameters:
//   - dir: The directory path (absolute, relative, or empty for default)
//
// Returns:
//   - The same [DittoConfig] instance for chaining
func (c *DittoConfig) WithPersistenceDirectory(dir string) *DittoConfig {
	c.PersistenceDirectory = dir
	return c
}

// WithConnect sets the connection configuration and returns the config for chaining.
//
// Parameters:
//   - connect: The connection configuration ([DittoConfigConnectServer], [DittoConfigConnectSmallPeersOnly])
//
// Returns:
//   - The same DittoConfig instance for chaining
func (c *DittoConfig) WithConnect(connect DittoConfigConnect) *DittoConfig {
	c.Connect = DittoConfigConnectUnion{}
	switch ct := connect.(type) {
	case *DittoConfigConnectServer:
		c.Connect.Type = DittoConfigConnectTypeServer
		c.Connect.DittoConfigConnectServer = ct
	case *DittoConfigConnectSmallPeersOnly:
		c.Connect.Type = DittoConfigConnectTypeSmallPeersOnly
		c.Connect.DittoConfigConnectSmallPeersOnly = ct
	case nil:
		c.Connect.Type = ""
	default:
		panic("unknown connect type")
	}
	return c
}

// toCBOR serializes the configuration to CBOR format for FFI.
//
// This method is used internally to convert the Go configuration
// structure into the binary format expected by the Ditto core library.
//
// Returns:
//   - CBOR-encoded configuration bytes
//   - An error if serialization fails
func (c *DittoConfig) toCBOR() ([]byte, error) {
	if c.Experimental == nil {
		// The Ditto FFI requires a non-nil CBOR Map.
		c.Experimental = make(map[string]any)
	}

	return cbor.Encode(c)
}
