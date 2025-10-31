package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/getditto/ditto-go-sdk/ditto"
)

// Config represents the application configuration options specified via
// command-line args, config file, and/or environment variables.
type Config struct {
	// DatabaseID is the unique identifier for the Ditto database/application.
	// Required for all authentication systems.
	// Set via: --db-id flag, DITTO_DATABASE_ID env var, or config file.
	DatabaseID string `json:"database_id"`

	// AuthSystem specifies the authentication mode to use.
	// Valid values: "small-peers-only", "online-with-authentication", "online-playground", "shared-secret".
	// Set via: --auth-system flag, DITTO_AUTH_SYSTEM env var, or config file.
	// Default: "small-peers-only".
	AuthSystem string `json:"auth_system"`

	// ConfigFile specifies the path to a TOML configuration file.
	// Set via: --config flag only.
	ConfigFile string `json:"config"`

	// LogLevel controls the verbosity of SDK logging.
	// Valid values: "error", "warning", "info", "debug".
	// Set via: --log-level flag, DITTO_LOG_LEVEL env var, or config file.
	// Default: "error".
	LogLevel string `json:"log_level"`

	// LogFile specifies where to write log output.
	// If empty, logs are written to stderr.
	// Set via: --log-file flag, DITTO_LOG_FILE env var, or config file.
	LogFile string `json:"log_file"`

	// PlaygroundToken is the authentication token for online-playground mode.
	// Required when AuthSystem is "online-playground".
	// Set via: --playground-token flag, DITTO_PLAYGROUND_TOKEN env var, or config file.
	PlaygroundToken string `json:"playground_token"`

	// SharedToken is the authentication token for shared-secret mode.
	// Required when AuthSystem is "shared-secret".
	// Set via: --shared-token flag, DITTO_SHARED_TOKEN env var, or config file.
	SharedToken string `json:"shared_token"`

	// SecretKey is the secret key for shared-secret authentication.
	// Required when AuthSystem is "shared-secret".
	// Set via: --secret-key flag, DITTO_SECRET_KEY env var, or config file.
	SecretKey string `json:"secret_key"`

	// OfflineOnlyLicenseToken is the license for offline-only operation.
	// Optional, used with "small-peers-only" mode for licensed features.
	// Set via: --offline-only-license-token flag, DITTO_LICENSE env var, or config file.
	OfflineOnlyLicenseToken string `json:"offline_only_license_token"`

	// PersistentRoot is the directory where Ditto stores persistent data.
	// Set via: --persistent-root flag, DITTO_PERSISTENT_ROOT env var, or config file.
	// Default: OS-specific application data directory + "/ditto-go-tools".
	PersistentRoot string `json:"persistent_root"`

	// AuthURL is the authentication server URL for online modes.
	// Required when AuthSystem is "online-with-authentication".
	// Set via: --auth-url flag, DITTO_AUTH_URL env var, or config file.
	AuthURL string `json:"auth_url"`

	// WebsocketURLs are the WebSocket endpoints for sync connections.
	// Optional, comma-separated list of URLs.
	// Set via: --websocket-urls flag, DITTO_WEBSOCKET_URL env var, or config file.
	WebsocketURLs []string `json:"websocket_urls"`

	// AuthEventHandler specifies a script/program to handle authentication events.
	// Optional, path to executable that processes auth events.
	// Set via: --auth-event-handler flag, DITTO_AUTH_EVENT_HANDLER env var, or config file.
	AuthEventHandler string `json:"auth_event_handler"`

	// EnableBLE controls whether Bluetooth Low Energy transport is enabled.
	// Set via: --enable-ble flag, DITTO_ENABLE_BLE env var, or config file.
	// Default: true.
	EnableBLE bool `json:"enable_ble"`

	// EnableLAN controls whether local area network transport is enabled.
	// Set via: --enable-lan flag, DITTO_ENABLE_LAN env var, or config file.
	// Default: true.
	EnableLAN bool `json:"enable_lan"`

	// TCPListener enables a TCP server for incoming connections.
	// Requires TCPListenerPort to be set.
	// Set via: --tcp-listener flag, DITTO_TCP_LISTENER env var, or config file.
	// Default: false.
	TCPListener bool `json:"tcp_listener"`

	// TCPListenerPort specifies the port for the TCP listener.
	// Required when TCPListener is true.
	// Set via: --tcp-listener-port flag, DITTO_TCP_LISTENER_PORT env var, or config file.
	TCPListenerPort uint16 `json:"tcp_listener_port"`

	// Name is a human-readable identifier for this peer.
	// Used in presence and diagnostic information.
	// Set via: --name flag, DITTO_NAME env var, or config file.
	// Default: "ditto-go-tools".
	Name string `json:"name"`

	// Trace specifies the file path for runtime tracing output.
	// Optional, enables Go runtime tracing when set.
	// Set via: --trace flag, DITTO_TRACE env var, or config file.
	Trace string `json:"trace"`

	// CPUProfile specifies the file path for CPU profiling output.
	// Optional, enables CPU profiling when set.
	// Set via: --cpuprofile flag, DITTO_CPUPROFILE env var, or config file.
	CPUProfile string `json:"cpuprofile"`

	// MemProfile specifies the file path for memory allocation profiling output.
	// Optional, writes allocation profile at program exit when set.
	// Set via: --memprofile flag, DITTO_MEMPROFILE env var, or config file.
	MemProfile string `json:"memprofile"`

	// HeapProfile specifies the file path for heap profiling output.
	// Optional, writes heap profile at program exit when set.
	// Set via: --heapprofile flag, DITTO_HEAPPROFILE env var, or config file.
	HeapProfile string `json:"heapprofile"`

	// NoColor disables color output in the terminal.
	// Set via: --no-color flag or NO_COLOR env var.
	// Default: false.
	NoColor bool `json:"no_color"`
}

// NewDefaultConfig creates a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		AuthSystem:      AuthSystemSmallPeersOnly,
		LogLevel:        "error",
		EnableBLE:       true,
		EnableLAN:       true,
		TCPListener:     false,
		TCPListenerPort: 0,
		Name:            "ditto-go-tools",
	}
}

// NewConfig creates and validates a new Config
func NewConfig() (*Config, error) {
	config := NewDefaultConfig()
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return config, nil
}

// Validate checks if the configuration is valid and returns an error if not
func (c *Config) Validate() error {
	if c.TCPListener && c.TCPListenerPort == 0 {
		return fmt.Errorf("tcp-listener requires tcp-listener-port")
	}

	if c.PersistentRoot == "" {
		c.PersistentRoot = filepath.Join(ditto.DefaultRootDirectory(), "ditto-go-tools")
	}

	authSystemCanonical := underscoreToHyphen(strings.ToLower(c.AuthSystem))

	switch authSystemCanonical {
	case AuthSystemSmallPeersOnly:
		if c.DatabaseID == "" {
			return fmt.Errorf("small-peers-only requires db-id to be provided")
		}
	case AuthSystemOnlineWithAuthentication:
		if c.AuthURL == "" && c.AuthEventHandler == "" {
			return fmt.Errorf("online-with-authentication requires auth-url or auth-event-handler to be provided")
		}
	case AuthSystemOnlinePlayground:
		if c.PlaygroundToken == "" {
			return fmt.Errorf("online-playground requires playground-token to be provided")
		}
		if c.DatabaseID == "" {
			return fmt.Errorf("online-playground requires db-id to be provided")
		}
		if c.AuthURL == "" {
			return fmt.Errorf("online-playground requires auth-url to be provided")
		}
	case AuthSystemSharedSecret:
		if c.DatabaseID == "" {
			return fmt.Errorf("shared-secret requires db-id to be provided")
		}
	default:
		return fmt.Errorf("valid auth-system values are %s, %s, %s, and %s",
			AuthSystemSmallPeersOnly,
			AuthSystemOnlineWithAuthentication,
			AuthSystemOnlinePlayground,
			AuthSystemSharedSecret)
	}

	return nil
}

// String returns a string representation of the config for debugging
func (c *Config) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("database_id:                 %s\n", c.DatabaseID))
	sb.WriteString(fmt.Sprintf("auth_system:                 %s\n", c.AuthSystem))
	sb.WriteString(fmt.Sprintf("config:                      %s\n", c.ConfigFile))
	sb.WriteString(fmt.Sprintf("log_level:                   %s\n", c.LogLevel))
	sb.WriteString(fmt.Sprintf("log_file:                    %s\n", c.LogFile))
	sb.WriteString(fmt.Sprintf("playground_token:            %s\n", c.PlaygroundToken))
	sb.WriteString(fmt.Sprintf("shared_token:                %s\n", c.SharedToken))
	sb.WriteString(fmt.Sprintf("secret_key:                  %s\n", c.SecretKey))
	sb.WriteString(fmt.Sprintf("offline_only_license_token:  %s\n", c.OfflineOnlyLicenseToken))
	sb.WriteString(fmt.Sprintf("persistent_root:             %s\n", c.PersistentRoot))
	sb.WriteString(fmt.Sprintf("auth_url:                    %s\n", c.AuthURL))
	sb.WriteString(fmt.Sprintf("websocket_urls:              %s\n", strings.Join(c.WebsocketURLs, ",")))
	sb.WriteString(fmt.Sprintf("auth_event_handler:          %s\n", c.AuthEventHandler))
	sb.WriteString(fmt.Sprintf("enable_ble:                  %t\n", c.EnableBLE))
	sb.WriteString(fmt.Sprintf("enable_lan:                  %t\n", c.EnableLAN))
	sb.WriteString(fmt.Sprintf("tcp_listener:                %t\n", c.TCPListener))
	sb.WriteString(fmt.Sprintf("tcp_listener_port:           %d\n", c.TCPListenerPort))
	sb.WriteString(fmt.Sprintf("name:                        %s\n", c.Name))
	sb.WriteString(fmt.Sprintf("trace:                       %s\n", c.Trace))
	sb.WriteString(fmt.Sprintf("cpuprofile:                  %s\n", c.CPUProfile))
	sb.WriteString(fmt.Sprintf("memprofile:                  %s\n", c.MemProfile))
	sb.WriteString(fmt.Sprintf("heapprofile:                 %s\n", c.HeapProfile))
	sb.WriteString(fmt.Sprintf("no_color:                    %t\n", c.NoColor))
	return sb.String()
}
