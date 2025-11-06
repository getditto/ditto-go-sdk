// Copyright 2025 DittoLive Incorporated. All rights reserved.

// Demonstration of how to use RegisterObserver() and RegisterSubscription() to receive updates from Ditto peers.
package main

import (
	"flag"
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
	waitDuration         time.Duration
)

// Command-line flags
var (
	waitSeconds          = flag.Int("wait", 60, "number of seconds to run before shutting down (0 or negative for no shutdown)")
	persistenceDirectory = flag.String("persistence-directory", "/tmp/ditto-go-example-sync", "directory for Ditto persistence")
	logExportPath        = flag.String("log-export-path", "./ditto-go-example-sync-logexport.jsonl.gz", "path for exported log file")
)

// Default values when environment variables are not set.
//
// Hard-code the values here if you aren't using environment variables or command-line options to specify them
const (
	defaultAppID           = "replace with your app ID"
	defaultAuthURL         = "replace with your authorization URL"
	defaultPlaygroundToken = "replace with your playground token"
)

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

	// Parse command-line flags
	flag.Parse()

	// Convert seconds to duration (0 or negative means wait forever)
	if *waitSeconds > 0 {
		waitDuration = time.Duration(*waitSeconds) * time.Second
	}

	// Print SDK version
	log.Printf("info: running Ditto example app with SDK version %s", ditto.Version())

	// Create Ditto configuration
	config := ditto.DefaultDittoConfig().
		WithDatabaseID(dittoAppID).
		WithPersistenceDirectory(*persistenceDirectory).
		WithConnect(&ditto.DittoConfigConnectServer{URL: dittoAuthURL})

	// Open Ditto instance
	d, err := ditto.Open(config)
	if err != nil {
		log.Fatalf("error: failed to open Ditto: %v", err)
	}
	defer d.Close()

	ditto.SetMinimumLogLevel(ditto.LogLevelWarning)

	d.SetDeviceName("ditto-go-example-sync")

	// We want to export the log at shutdown or failure
	defer exportDittoLog()

	// Make sure we start with no cached credentials or tokens
	d.Auth().Logout(nil)

	// Set up authentication expiration handler, which will be called for initial login and after expiration
	provider := ditto.DevelopmentAuthenticationProvider()
	d.Auth().SetExpirationHandler(
		func(d *ditto.Ditto, timeUntilExpiration time.Duration) {
			clientInfoJSON, err := d.Auth().Login(
				dittoPlaygroundToken,
				provider,
			)
			if err != nil {
				log.Printf("error: expiration handler: %v", err)
			} else {
				log.Printf("info: expiration handler: logged in; client info: %v", clientInfoJSON)
			}
		})

	// Start sync
	if err := d.Sync().Start(); err != nil {
		log.Fatalf("error: failed to start sync: %v", err)
	}
	defer d.Sync().Stop()

	// Subscribe to changes from other peers
	// TODO: Update the subscriptionQuery to match your data model
	subscriptionQuery := "SELECT * FROM tasks"
	subscription, err := d.Sync().RegisterSubscription(subscriptionQuery)
	if err != nil {
		log.Fatalf("error: failed to register subscription: %v", err)
	}
	defer subscription.Cancel()

	// Register an observer to watch for changes and send them to the main thread via a channel
	// TODO: Update the observerQuery and observerArgs to match your data model
	updateChan := make(chan []string, 1)
	observerQuery := "SELECT * FROM tasks WHERE deleted = false ORDER BY _id ASC"
	var observerArgs ditto.QueryArguments = nil
	observer, err := d.Store().RegisterObserver(
		observerQuery, observerArgs,
		func(result *ditto.QueryResult) {
			// Need to free the QueryResult resources when finished
			defer result.Close()

			// Extract JSON for each result item and send it to the main event loop for processing
			items := result.Items()
			var jsonItems []string
			for _, item := range items {
				jsonItems = append(jsonItems, item.JSONString())
			}
			updateChan <- jsonItems
		})
	if err != nil {
		log.Fatalf("error: failed to register observer: %v", err)
	}
	defer observer.Cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a timer for the wait duration (if not waiting forever)
	var timerChan <-chan time.Time
	if waitDuration > 0 {
		timerChan = time.After(waitDuration)
		log.Printf("info: waiting for %v (or until interrupted)...", waitDuration)
	} else {
		log.Println("info: waiting forever (until interrupted)...")
	}

	// Handle events in a loop until timeout or interrupt
	for {
		select {
		case jsonItems := <-updateChan:
			handleCollectionUpdate(jsonItems)
			// Continue processing more updates
		case <-timerChan:
			if waitDuration > 0 {
				log.Printf("info: %v elapsed, shutting down...", waitDuration)
				return
			}
		case <-sigChan:
			log.Println("info: received interrupt signal, shutting down...")
			return
		}
	}

	// Clean up (deferred functions will handle subscription, observer, sync stop, and log export)
}

// handleCollectionUpdate is called when new collection contents are received.
//
// In a real application, this is where the UI would be updated or data processed.
// This demo app simply prints the JSON representation of the items.
func handleCollectionUpdate(jsonItems []string) {
	log.Println("info: --- update for collection:")
	for index, item := range jsonItems {
		log.Printf("info: item %d: %s\n", index, item)
	}
	log.Println("info: --- end of update for collection")
}

// exportDittoLog exports the Ditto logs to a file
func exportDittoLog() {
	if *logExportPath == "" {
		return
	}

	// Remove existing log file if it exists
	if err := os.Remove(*logExportPath); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: failed to remove existing log file: %v", err)
	}

	// Export logs
	lineCount, err := ditto.ExportLog(*logExportPath)
	if err != nil {
		log.Printf("error: unable to export log dump: %v", err)
	} else {
		log.Printf("info: exported %d log lines to %s", lineCount, *logExportPath)
	}
}
