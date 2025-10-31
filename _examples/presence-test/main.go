// This program demonstrates how to use the Ditto Presence API to handle connection requests
// using ditto.Presence.SetConnectionRequestHandler() and to track other devices in the Ditto
// mesh using ditto.Presence.Observe().
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getditto/ditto-go-sdk/ditto"
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
	instanceID           = flag.String("id", "", "Instance ID for this peer (optional, random if not provided)")
	waitSeconds          = flag.Int("wait", 30, "number of seconds to run before shutting down (0 or negative for no shutdown)")
	persistenceDirectory = flag.String("persistence-directory", "", "directory for Ditto persistence (defaults to /tmp/ditto-go-example-presence-<id>)")
	logExportPath        = flag.String("log-export-path", "", "path for exported log file (defaults to ./ditto-go-example-presence-<id>-logexport.jsonl.gz)")
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

	// Generate a random ID if not provided
	if *instanceID == "" {
		*instanceID = generateRandomID()
	}

	// Set default paths if not provided
	if *persistenceDirectory == "" {
		*persistenceDirectory = fmt.Sprintf("/tmp/ditto-go-example-presence-%s", *instanceID)
	}
	if *logExportPath == "" {
		*logExportPath = fmt.Sprintf("./ditto-go-example-presence-%s-logexport.jsonl.gz", *instanceID)
	}

	// Convert seconds to duration (0 or negative means wait forever)
	if *waitSeconds > 0 {
		waitDuration = time.Duration(*waitSeconds) * time.Second
	}

	// Print SDK version
	log.Printf("info: running Ditto presence example (instance %s) with SDK version %s",
		*instanceID, ditto.Version())

	// Set log level
	ditto.SetMinimumLogLevel(ditto.LogLevelWarning)

	// Create Ditto configuration with DittoConfigConnectServer (online mode)
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

	// We want to export the log at shutdown or failure
	defer exportDittoLog()

	// Set up authentication expiration handler
	d.Auth().SetExpirationHandler(func(d *ditto.Ditto, timeUntilExpiration time.Duration) {
		clientJSON, err := d.Auth().Login(
			dittoPlaygroundToken,
			ditto.DevelopmentAuthenticationProvider(),
		)
		if err != nil {
			log.Printf("error: expiration handler: %v", err)
		} else {
			log.Printf("info: expiration handler: logged in with client info: %s", clientJSON)
		}
	})

	// Start sync
	if err := d.Sync().Start(); err != nil {
		log.Fatalf("error: failed to start sync: %v", err)
	}
	defer d.Sync().Stop()

	// Set this peer's metadata
	metadata := map[string]any{
		"deviceName": fmt.Sprintf("Go SDK Example %s", *instanceID),
		"instanceID": *instanceID,
		"platform":   "Go",
		"version":    ditto.Version(),
	}
	if err := d.Presence().SetPeerMetadata(metadata); err != nil {
		log.Printf("warning: failed to set peer metadata: %v", err)
	}

	// Observe presence changes, sending the new presence graph to a channel to be processed
	// by the main event loop
	type PresenceEvent struct {
		graph *ditto.PresenceGraph
	}
	presenceChan := make(chan PresenceEvent)
	observer := d.Presence().Observe(func(graph *ditto.PresenceGraph) {
		presenceChan <- PresenceEvent{graph}
	})
	defer observer.Stop()

	// Set up connection request handler
	d.Presence().SetConnectionRequestHandler(handleConnectionRequest)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a timer for the wait duration (if not waiting forever)
	var timerChan <-chan time.Time
	if waitDuration > 0 {
		timerChan = time.After(waitDuration)
		log.Printf("info: waiting for %v to discover peers (or until interrupted)...",
			waitDuration)
	} else {
		log.Println("info: waiting forever to discover peers (until interrupted)...")
	}
	log.Println("info: try running this example on multiple devices on the same network")

	// Event loop - process ALL presence events until timeout or interrupt
	for {
		select {
		case presenceEvent := <-presenceChan:
			handlePresenceUpdate(presenceEvent.graph)
		case <-timerChan:
			log.Printf("\ninfo: %v elapsed, shutting down...", waitDuration)
			return
		case <-sigChan:
			log.Println("\ninfo: received interrupt signal, shutting down...")
			return
		}
	}

	// Clean up (deferred functions will handle observer stop, sync stop, and log export)
}

// handlePresenceUpdate checks for differences between new peer states and previous peer states, and describes the changes.
//
// Arguments:
//   - peer: The peer whose presence status has changed
//   - isPresent: true if the peer joined, false if the peer left
//   - activePeers: Map of currently active peers indexed by peer key string, which will be modified if appropriate
//   - mutex: Mutex to protect access to activePeers map
func handlePresenceUpdate(graph *ditto.PresenceGraph) {
	fmt.Println("---- PRESENCE UPDATE ----")

	fmt.Println("- Local Peer:")
	printPeerInfo(graph.LocalPeer)

	fmt.Println("- Remote Peers:")
	if len(graph.RemotePeers) > 0 {
		for _, peer := range graph.RemotePeers {
			printPeerInfo(peer)
		}
	} else {
		fmt.Println("- <none>")
	}

	fmt.Println("- Connections:")
	if len(graph.AllConnectionsByID) > 0 {
		for _, conn := range graph.AllConnectionsByID {
			printConnectionInfo(conn)
		}
	} else {
		fmt.Println("- <none>")
	}
	fmt.Println("----- END PRESENCE UPDATE ----")
}

// printPeerInfo prints a line of information about a peer
func printPeerInfo(peer *ditto.Peer) {
	fmt.Printf("  - Name: %s; OS: %s; SDK: %s; PeerKeyString: %s\n",
		peer.DeviceName, peer.OS, peer.DittoSDKVersion, peer.PeerKeyString)
}

func printConnectionInfo(conn *ditto.Connection) {
	fmt.Printf("  - %s <-> %s %s\n",
		conn.PeerKeyString1, conn.PeerKeyString2, conn.Type)
}

// generateRandomID generates a random 4-byte hex string for use as an instance ID
func generateRandomID() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	}
	return hex.EncodeToString(bytes)
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

// handleConnectionRequest handles incoming connection requests from peers
func handleConnectionRequest(request *ditto.ConnectionRequest) ditto.ConnectionRequestAuthorization {
	fmt.Println("---- CONNECTION REQUEST ----")
	fmt.Printf("- Peer Key: %s\n", request.PeerKey)

	if request.PeerMetadata != nil {
		fmt.Printf("- PeerMetadata: %v\n", request.PeerMetadata)
	}

	if request.IdentityData != nil {
		fmt.Printf("- IdentityData: %v\n", request.IdentityData)
	}

	// This example allows all connections
	fmt.Println("- Decision: ALLOW")
	fmt.Println("---- END CONNECTION REQUEST ----")

	return ditto.ConnectionRequestAuthorizationAllow
}
