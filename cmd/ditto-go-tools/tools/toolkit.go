// Package tools provides command implementations and utilities for the ditto-go-tools
// command-line application. It includes the Toolkit wrapper for Ditto SDK operations,
// command handlers (DoXXX functions), configuration management, and helper functions
// for data formatting and transformation.
package tools

import (
	"context"
	"fmt"
	"runtime/trace"
	"strings"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// Toolkit wraps a Ditto instance with application-specific behavior
type Toolkit struct {
	ctx    context.Context
	config *Config
	ditto  *ditto.Ditto
}

// NewToolkit creates a Ditto instance configured as specified.
func NewToolkit(ctx context.Context, config *Config) (*Toolkit, error) {
	defer trace.StartRegion(ctx, "NewToolkit").End()
	trace.Log(ctx, "app", "NewToolkit start")

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	persistenceDir := config.PersistentRoot
	if persistenceDir == "" {
		tempDir, err := NewTempPersistenceDir(false)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp persistence dir: %w", err)
		}
		persistenceDir = tempDir.Path()
	}

	authSystem := underscoreToHyphen(strings.ToLower(config.AuthSystem))

	// Create Ditto configuration
	dittoConfig := ditto.DefaultDittoConfig().
		WithDatabaseID(config.DatabaseID).
		WithPersistenceDirectory(persistenceDir)

	switch authSystem {
	case AuthSystemSmallPeersOnly:
		if config.DatabaseID == "" {
			return nil, fmt.Errorf("database-id must be provided for small-peers-only")
		}

		// No private key for small-peers-only
		dittoConfig = dittoConfig.WithConnect(&ditto.DittoConfigConnectSmallPeersOnly{PrivateKey: nil})

	case AuthSystemOnlinePlayground:
		if config.DatabaseID == "" {
			return nil, fmt.Errorf("database-id must be provided for online-playground")
		}
		if config.PlaygroundToken == "" {
			return nil, fmt.Errorf("playground-token must be provided for online-playground")
		}
		if config.AuthURL == "" {
			return nil, fmt.Errorf("auth-url must be provided for online-playground")
		}

		dittoConfig = dittoConfig.WithConnect(&ditto.DittoConfigConnectServer{URL: config.AuthURL})

	case AuthSystemSharedSecret:
		if config.DatabaseID == "" {
			return nil, fmt.Errorf("database-id must be provided for shared-secret")
		}

		dittoConfig = dittoConfig.WithConnect(&ditto.DittoConfigConnectSmallPeersOnly{PrivateKey: []byte(config.SecretKey)})

	default:
		return nil, fmt.Errorf("%s is not yet supported by this application", authSystem)
	}

	dittoInstance, err := ditto.Open(dittoConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ditto instance: %w", err)
	}
	// NOTE: Toolkit takes ownership of dittoInstance on success.
	// Error paths explicitly close it before returning.

	if config.LogLevel != "" {
		logLevel, err := toLogLevel(config.LogLevel)
		if err != nil {
			dittoInstance.Close()
			return nil, fmt.Errorf("invalid log level: %w", err)
		}

		ditto.SetMinimumLogLevel(logLevel)
	}

	// Set offline-only license token for small-peers-only authentication
	if authSystem == AuthSystemSmallPeersOnly && config.OfflineOnlyLicenseToken != "" {
		if err = dittoInstance.SetOfflineOnlyLicenseToken(config.OfflineOnlyLicenseToken); err != nil {
			dittoInstance.Close()
			return nil, fmt.Errorf("failed to set offline-only license token: %w", err)
		}
	}

	if authSystem == AuthSystemOnlinePlayground {
		// set auth expiration handler
		if auth := dittoInstance.Auth(); auth != nil {
			auth.SetExpirationHandler(
				func(d *ditto.Ditto, timeUntilExpiration time.Duration) {
					clientInfoJSON, err := d.Auth().Login(
						config.PlaygroundToken,
						ditto.DevelopmentAuthenticationProvider(),
					)
					if err != nil {
						fmt.Printf("error: expiration handler: %v\n", err)
					} else {
						fmt.Printf("info: expiration handler: logged in; client info: %s\n", clientInfoJSON)
					}
				},
			)
		}
	}

	return &Toolkit{
		ctx:    ctx,
		config: config,
		ditto:  dittoInstance,
	}, nil
}

// Close shuts down the Ditto instance
func (t *Toolkit) Close() {
	defer trace.StartRegion(t.ctx, "Toolkit.Close").End()
	trace.Log(t.ctx, "app", "Toolkit.Close start")

	if t.ditto != nil {
		t.ditto.Close()
	}
}

// StartSync starts synchronization
func (t *Toolkit) StartSync() error {
	defer trace.StartRegion(t.ctx, "Toolkit.StartSync").End()
	trace.Log(t.ctx, "app", "Toolkit.StartSync start")

	return t.ditto.Sync().Start()
}

// StopSync stops synchronization
func (t *Toolkit) StopSync() {
	defer trace.StartRegion(t.ctx, "Toolkit.StopSync").End()
	trace.Log(t.ctx, "app", "Toolkit.StopSync start")

	t.ditto.Sync().Stop()
}

// Execute executes a DQL query and returns the result
func (t *Toolkit) Execute(query string, queryArgs ditto.QueryArguments) (*ditto.QueryResult, error) {
	defer trace.StartRegion(t.ctx, "Toolkit.Execute").End()
	trace.Log(t.ctx, "app", "Toolkit.Execute start")

	return t.ditto.Store().Execute(query, queryArgs)
}

// RegisterObserver registers an observer for a DQL query
func (t *Toolkit) RegisterObserver(
	query string, queryArgs ditto.QueryArguments, callback func(*ditto.QueryResult),
) (*ditto.StoreObserver, error) {
	defer trace.StartRegion(t.ctx, "Toolkit.RegisterObserver").End()
	trace.Log(t.ctx, "app", "Toolkit.RegisterObserver start")

	return t.ditto.Store().RegisterObserver(
		query, queryArgs, func(queryResult *ditto.QueryResult) {
			defer trace.StartRegion(t.ctx, "Toolkit.RegisterObserver callback").End()
			trace.Log(t.ctx, "app", "Toolkit.RegisterObserver callback invoked")
			callback(queryResult)
		},
	)
}

// RegisterSubscription registers a sync subscription for a DQL query
func (t *Toolkit) RegisterSubscription(query string, queryArgs ditto.QueryArguments) (*ditto.SyncSubscription, error) {
	defer trace.StartRegion(t.ctx, "Toolkit.RegisterSubscription").End()
	trace.Log(t.ctx, "app", "Toolkit.RegisterSubscription start")

	return t.ditto.Sync().RegisterSubscription(query, queryArgs)
}

// GetPersistenceDirectory returns the path to the persistence directory
func (t *Toolkit) GetPersistenceDirectory() string {
	defer trace.StartRegion(t.ctx, "Toolkit.GetPersistenceDirectory").End()
	trace.Log(t.ctx, "app", "Toolkit.GetPersistenceDirectory start")

	return t.config.PersistentRoot
}

// RegisterPresenceObserver starts observing presence changes and returns the observer
func (t *Toolkit) RegisterPresenceObserver(callback func(graph *ditto.PresenceGraph)) (*ditto.PresenceObserver, error) {
	defer trace.StartRegion(t.ctx, "Toolkit.RegisterPresenceObserver").End()
	trace.Log(t.ctx, "app", "Toolkit.RegisterPresenceObserver start")

	observer := t.ditto.Presence().Observe(
		func(graph *ditto.PresenceGraph) {
			defer trace.StartRegion(t.ctx, "Toolkit.RegisterPresenceObserver callback").End()
			trace.Log(t.ctx, "app", "Toolkit.RegisterPresenceObserver callback invoked")
			callback(graph)
		},
	)
	return observer, nil
}
