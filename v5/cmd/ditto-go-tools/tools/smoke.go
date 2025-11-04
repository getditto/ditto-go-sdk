package tools

import (
	"context"
	"crypto/rand"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// DoSmokeTest performs a basic smoke test of Ditto functionality
func DoSmokeTest(ctx context.Context, config *Config) error {
	ctx, task := trace.NewTask(ctx, "DoSmokeTest")
	defer task.End()
	defer trace.StartRegion(ctx, "DoSmokeTest").End()

	fmt.Println("Creating Ditto instance...")
	toolkit, err := NewToolkit(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create toolkit: %w", err)
	}
	defer toolkit.Close()

	fmt.Printf("Initialized with persistence directory \"%s\"\n", toolkit.GetPersistenceDirectory())

	fmt.Println("Starting sync...")
	if err := toolkit.StartSync(); err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}

	// Generate a random collection name for the smoke test
	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("failed to generate random collection name: %w", err)
	}
	collectionName := fmt.Sprintf("ditto_go_tools_smoke_test_%x", randBytes)
	fmt.Printf("Using test collection: %s\n", collectionName)

	// Insert a test document to ensure the collection exists
	insertQuery := fmt.Sprintf("INSERT INTO %s DOCUMENTS (:doc)", collectionName)
	testDoc := ditto.Document{"test": true, "timestamp": time.Now().Unix()}
	_, err = toolkit.Execute(insertQuery, ditto.QueryArguments{"doc": testDoc})
	if err != nil {
		return fmt.Errorf("failed to insert test document: %w", err)
	}
	fmt.Println("Inserted test document")

	// Register observer on the test collection
	selectQuery := fmt.Sprintf("SELECT * FROM %s", collectionName)
	obs, err := toolkit.RegisterObserver(
		selectQuery, nil, func(result *ditto.QueryResult) {
			fmt.Println("Observer callback invoked")
		},
	)
	if err != nil {
		return fmt.Errorf("failed to register observer: %w", err)
	}
	defer obs.Cancel()

	fmt.Println("Waiting 5 seconds...")
	time.Sleep(5 * time.Second)

	fmt.Println("Stopping sync...")
	toolkit.StopSync()

	fmt.Println("Shutting down Ditto...")
	return nil
}
