package main

// Demonstrates log in using playground authentication token, and use of authentication status observer.

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

// Environment variables / .env file entries:
//
//	DITTO_APP_ID - Your Ditto application ID from https://portal.ditto.live/
//	DITTO_AUTH_URL - Your Ditto authentication URL (e.g., https://your-app-id.cloud.ditto.live)
//	DITTO_PLAYGROUND_TOKEN - Your Ditto playground token
var (
	dittoAppID           string
	dittoAuthURL         string
	dittoPlaygroundToken string
	persistenceDirectory string
	logExportPath        string
	waitDuration         time.Duration
)

// Default values when environment variables are not set.
//
// Hard-code the values here if you aren't using environment variables or command-line options to specify them
const (
	defaultAppID           = "replace with your app ID"
	defaultAuthURL         = "replace with your authorization URL"
	defaultPlaygroundToken = "replace with your playground token"
)

func init() {
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
}

func main() {
	// Parse command-line flags
	waitSeconds := flag.Int("wait", 10, "number of seconds to run before shutting down (0 or negative for no shutdown)")
	flag.StringVar(&persistenceDirectory, "persistence-directory", "/tmp/ditto-go-example-auth-example", "directory for Ditto persistence")
	flag.StringVar(&logExportPath, "log-export-path", "./ditto-go-example-auth-example-logexport.jsonl.gz", "path for exported log file")
	flag.Parse()

	// Convert seconds to duration (0 or negative means wait forever)
	if *waitSeconds > 0 {
		waitDuration = time.Duration(*waitSeconds) * time.Second
	}

	// Print SDK version
	log.Printf("info: running Ditto auth example with SDK version %s", ditto.Version())

	// Set Ditto log level
	ditto.SetMinimumLogLevel(ditto.LogLevelError)

	// Create a Ditto instance with server authentication
	config := ditto.DefaultDittoConfig().
		WithDatabaseID(dittoAppID).
		WithPersistenceDirectory(persistenceDirectory).
		WithConnect(&ditto.DittoConfigConnectServer{URL: dittoAuthURL})

	log.Println("info: calling ditto.Open()")
	d, err := ditto.Open(config)
	if err != nil {
		log.Fatalf("Failed to open Ditto: %v\n", err)
	}
	defer func() {
		log.Println("info: calling ditto.Close()")
		d.Close()
		exportDittoLog()
	}()

	d.SetDeviceName("ditto-go-auth-example")

	// Start observing authentication status changes
	log.Println("info: calling Auth.ObserveStatus()")
	observer := d.Auth().ObserveStatus(func(status *ditto.AuthenticationStatus) {
		logAuthenticationStatus(status)
	})
	defer func() {
		log.Println("info: calling observer.Cancel()")
		observer.Cancel()
	}()

	// Call logout to ensure we are starting without any stored credentials
	log.Printf("info: calling Logout")
	d.Auth().Logout(nil)

	// Set expiration handler, which will be called for initial login
	log.Printf("info: calling SetExpirationHandler")
	d.Auth().SetExpirationHandler(
		func(d *ditto.Ditto, timeUntilExpiration time.Duration) {
			log.Println("info: expiration handler: calling Auth.Login()")
			clientInfoJSON, err := d.Auth().Login(
				dittoPlaygroundToken,
				ditto.DevelopmentAuthenticationProvider(),
			)
			if err != nil {
				log.Printf("error: expiration handler: unable to log in: %v", err)
			} else {
				log.Printf("info: expiration handler: logged in; client info: %s", clientInfoJSON)
			}
		})

	// Start sync, which will initiate login
	log.Printf("info: calling Sync.Start()")
	err = d.Sync().Start()
	if err != nil {
		log.Fatalf("error: unable to start Ditto sync: %v", err)
	}
	defer func() {
		log.Printf("info: calling Sync.Stop()")
		d.Sync().Stop()
	}()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a timer for the wait duration (if not waiting forever)
	var shutdownTimer <-chan time.Time
	if waitDuration > 0 {
		shutdownTimer = time.After(waitDuration)
		log.Printf("info: waiting for %v (or until interrupted)...", waitDuration)
	} else {
		log.Println("info: waiting forever (until interrupted)...")
	}

	// Wait for timeout or interrupt
	select {
	case <-shutdownTimer:
		return
	case <-sigChan:
		log.Println("info: received interrupt signal, shutting down...")
		return
	}

	// deferred cleanup functions will run before exit
}

// logAuthenticationStatus prints out the values of an authentication status update
func logAuthenticationStatus(status *ditto.AuthenticationStatus) {
	log.Println("info: ---- Authentication Status Update ----")
	log.Printf("info: - IsAuthenticated: %t\n", status.IsAuthenticated)
	log.Printf("info: - UserID: %s\n", status.UserID)
	log.Printf("info: - ClientInfo: %s\n", status.ClientInfo)
	log.Println("info: ---- End of Authentication Status Update ----")
}

// exportDittoLog exports the Ditto logs to a file
func exportDittoLog() {
	// Remove existing log file if it exists
	os.Remove(logExportPath)

	// Export logs
	log.Println("info: calling ditto.ExportLog()")
	lineCount, err := ditto.ExportLog(logExportPath)
	if err != nil {
		log.Printf("error: unable to export log dump: %v\n", err)
	} else {
		log.Printf("info: exported %d log lines to %s\n", lineCount, logExportPath)
	}
}
