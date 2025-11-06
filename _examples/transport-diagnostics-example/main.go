// Copyright 2025 DittoLive Incorporated. All rights reserved.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
	"github.com/joho/godotenv"
)

// Configuration - values can be loaded from .env file, environment variables, or defaults
// The precedence order is:
//  1. Environment variables (highest priority)
//  2. .env file in working directory
//  3. Default values (lowest priority)
//
// Environment variables / .env file entries:
//
//	DITTO_APP_ID - Your Ditto application ID from https://portal.ditto.live/
//	DITTO_AUTH_URL - Your Ditto authentication URL (e.g., https://your-app-id.cloud.ditto.live)
//	DITTO_PLAYGROUND_TOKEN - Your Ditto playground token
var (
	dittoAppID           string
	dittoAuthURL         string
	dittoPlaygroundToken string
)

// Default values when environment variables are not set.
//
// Hard-code the values here if you aren't using environment variables or command-line options to specify them
const (
	defaultAppID           = "replace with your app ID"
	defaultAuthURL         = "replace with your authorization URL"
	defaultPlaygroundToken = "replace with your playground token"

	persistenceDir      = "/tmp/ditto-go-example-transport-diagnostics"
	logExportPath       = "./ditto-go-example-transport-diagnostics-logexport.jsonl.gz"
	diagnosticsInterval = 10 * time.Second
)

// exportDittoLog exports the Ditto logs to a file
func exportDittoLog() {
	// Remove existing log file if it exists
	os.Remove(logExportPath)

	// Export logs
	lineCount, err := ditto.ExportLog(logExportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unable to export log dump: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "info: exported %d log lines to %s\n", lineCount, logExportPath)
	}
}

func main() {
	// Load .env file first (environment variables take precedence)
	// godotenv.Load() silently ignores if .env file doesn't exist
	if err := godotenv.Load(); err != nil {
		// Only log if it's an actual error (not just missing file)
		if !os.IsNotExist(err) {
			log.Printf("warning: error loading .env file: %v", err)
		}
	}

	// Read configuration from environment variables, falling back to defaults
	dittoAppID = os.Getenv("DITTO_APP_ID")
	if dittoAppID == "" {
		dittoAppID = defaultAppID
	}

	dittoAuthURL = os.Getenv("DITTO_AUTH_URL")
	if dittoAuthURL == "" {
		dittoAuthURL = defaultAuthURL
	}

	dittoPlaygroundToken = os.Getenv("DITTO_PLAYGROUND_TOKEN")
	if dittoPlaygroundToken == "" {
		dittoPlaygroundToken = defaultPlaygroundToken
	}

	// Print SDK version
	fmt.Fprintf(os.Stderr, "info: running Ditto transport diagnostics with SDK version %s\n", ditto.Version())

	// Set log level
	ditto.SetMinimumLogLevel(ditto.LogLevelWarning)

	// Create Ditto configuration
	config := ditto.DefaultDittoConfig().
		WithDatabaseID(dittoAppID).
		WithPersistenceDirectory(persistenceDir).
		WithConnect(&ditto.DittoConfigConnectServer{URL: dittoAuthURL})

	// Open Ditto instance
	d, err := ditto.Open(config)
	if err != nil {
		log.Fatalf("error: failed to open Ditto: %v", err)
	}
	defer func() {
		d.Close()
		exportDittoLog()
	}()

	// Set up authentication expiration handler
	d.Auth().SetExpirationHandler(func(d *ditto.Ditto, timeUntilExpiration time.Duration) {
		clientInfoJSON, err := d.Auth().Login(
			dittoPlaygroundToken,
			ditto.DevelopmentAuthenticationProvider(),
		)
		if err != nil {
			log.Printf("error: expiration handler: %v", err)
		} else {
			log.Printf("info: expiration handler: logged in with client info: %s", clientInfoJSON)
		}
	})

	// Start sync
	if err := d.Sync().Start(); err != nil {
		log.Fatalf("error: failed to start sync: %v", err)
	}
	defer d.Sync().Stop()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a ticker for periodic diagnostics
	ticker := time.NewTicker(diagnosticsInterval)
	defer ticker.Stop()

	fmt.Printf("\ninfo: transport diagnostics will be printed every %v (press Ctrl+C to stop)...\n", diagnosticsInterval)
	fmt.Println("info: waiting for initial connections to establish...")

	// Wait a moment for initial connections
	time.Sleep(2 * time.Second)

	// Print diagnostics immediately, then every interval
	printDiagnostics := func() {
		diagnostics, err := d.TransportDiagnostics()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to get transport diagnostics: %v\n", err)
			return
		}

		// Convert to JSON with pretty printing
		jsonBytes, err := json.MarshalIndent(diagnostics, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to marshal diagnostics to JSON: %v\n", err)
			return
		}

		// Print timestamp and JSON
		fmt.Printf("\n=== Transport Diagnostics at %s ===\n", time.Now().Format(time.RFC3339))
		fmt.Println(string(jsonBytes))

		// Print summary using helper methods
		if diagnostics != nil {
			totalConnected := diagnostics.TotalConnectedPeers()
			activeTransports := diagnostics.ActiveTransports()
			fmt.Printf("\nSummary: %d active transport(s), %d total connected peer(s)\n", activeTransports, totalConnected)

			// Show details for each transport type
			for _, transport := range diagnostics.Transports {
				if transport.HasActivity() {
					fmt.Printf("  %s: %d connected, %d total peers\n",
						transport.ConnectionType, len(transport.Connected), transport.TotalPeers())
				}
			}
		}
	}

	// Print initial diagnostics
	printDiagnostics()

	// Main loop
	for {
		select {
		case <-ticker.C:
			printDiagnostics()
		case <-sigChan:
			fmt.Println("\ninfo: received interrupt signal, shutting down...")
			return
		}
	}
}
