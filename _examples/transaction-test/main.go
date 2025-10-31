package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/getditto/ditto-go-sdk/ditto"
	"github.com/joho/godotenv"
)

var (
	verbose      = flag.Bool("verbose", false, "Enable verbose logging")
	storageDir   = flag.String("storage-dir", "", "Directory for persistent storage (default: temp dir)")
	appID        = flag.String("app-id", "", "Application ID for Ditto")
	licenseToken = flag.String("license-token", "", "Offline license token")
)

func main() {
	flag.Parse()

	// Load environment variables from .env file if it exists
	_ = godotenv.Load()

	// Create Ditto instance
	d, err := createDitto()
	if err != nil {
		log.Fatalf("Failed to create Ditto: %v", err)
	}
	defer d.Close()

	fmt.Println("=== Ditto Transaction Examples ===")
	fmt.Println()

	// Demonstrate basic transaction commit
	fmt.Println("1. Basic Transaction Commit")
	if err := demoBasicCommit(d); err != nil {
		log.Printf("Basic commit failed: %v", err)
	}
	fmt.Println()

	// Demonstrate explicit rollback
	fmt.Println("2. Explicit Transaction Rollback")
	if err := demoExplicitRollback(d); err != nil {
		log.Printf("Explicit rollback failed: %v", err)
	}
	fmt.Println()

	// Demonstrate implicit rollback on error
	fmt.Println("3. Implicit Rollback on Error")
	if err := demoImplicitRollback(d); err != nil {
		log.Printf("Implicit rollback demonstration completed")
	}
	fmt.Println()

	// Demonstrate read-only transaction
	fmt.Println("4. Read-Only Transaction")
	if err := demoReadOnlyTransaction(d); err != nil {
		log.Printf("Read-only transaction failed: %v", err)
	}
	fmt.Println()

	// Demonstrate TransactionWithResult
	fmt.Println("5. Transaction With Result")
	if err := demoTransactionWithResult(d); err != nil {
		log.Printf("Transaction with result failed: %v", err)
	}
	fmt.Println()

	// Demonstrate multiple operations in one transaction
	fmt.Println("6. Multiple Operations in Single Transaction")
	if err := demoMultipleOperations(d); err != nil {
		log.Printf("Multiple operations failed: %v", err)
	}
	fmt.Println()

	fmt.Println("=== All Examples Complete ===")
}

func createDitto() (*ditto.Ditto, error) {
	// Determine storage directory
	var dir string
	if *storageDir != "" {
		dir = *storageDir
	} else if envDir := os.Getenv("DITTO_STORAGE_DIR"); envDir != "" {
		dir = envDir
	} else {
		tempDir := os.TempDir()
		dir = filepath.Join(tempDir, "ditto-transaction-test")
	}

	// Get app ID from flag, environment, or use default
	databaseID := *appID
	if databaseID == "" {
		databaseID = os.Getenv("DITTO_APP_ID")
	}
	if databaseID == "" {
		databaseID = "transaction-test-app"
	}

	// Create config
	config := ditto.DefaultDittoConfig().
		WithDatabaseID(databaseID).
		WithPersistenceDirectory(dir).
		WithConnect(&ditto.DittoConfigConnectSmallPeersOnly{})

	// Create Ditto instance
	d, err := ditto.Open(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ditto: %w", err)
	}

	// Apply license token if provided
	token := *licenseToken
	if token == "" {
		token = os.Getenv("DITTO_LICENSE")
	}
	if token != "" {
		if err := d.SetOfflineOnlyLicenseToken(token); err != nil {
			log.Printf("Warning: Failed to set license token: %v", err)
		}
	}

	if *verbose {
		log.Printf("Created Ditto instance with storage at: %s", dir)
		log.Printf("App ID: %s", databaseID)
	}

	return d, nil
}

func demoBasicCommit(d *ditto.Ditto) error {
	fmt.Println("Inserting document with ID 'car1' in a transaction...")

	action, err := d.Store().Transaction(
		nil, func(tx *ditto.Transaction) (ditto.TransactionCompletionAction, error) {
			// Insert a document
			_, err := tx.Execute(
				"INSERT INTO cars DOCUMENTS (:doc) ON ID CONFLICT DO UPDATE",
				ditto.QueryArguments{
					"doc": ditto.Document{
						"_id":   "car1",
						"make":  "Toyota",
						"model": "Camry",
						"year":  2024,
					},
				},
			)
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}

			// Commit the transaction
			return ditto.TransactionCompletionActionCommit, nil
		},
	)

	if err != nil {
		log.Fatalf("transaction failed: %w", err)
	}

	fmt.Printf("Transaction completed with action: %v\n", action)

	// Verify the document was inserted
	result, err := d.Store().Execute("SELECT * FROM cars WHERE _id = 'car1'")
	if err != nil {
		log.Fatalf("query failed: %w", err)
	}
	defer result.Close()

	found := false
	for _, item := range result.Items() {
		fmt.Printf("Verified: Document inserted successfully: %v\n", item.Value())
		found = true
		break
	}
	if !found {
		fmt.Println("Warning: Document not found after commit")
	}

	return nil
}

func demoExplicitRollback(d *ditto.Ditto) error {
	fmt.Println("Inserting document 'car2' then explicitly rolling back...")

	// Delete any existing car2 to ensure clean test
	_, _ = d.Store().Execute("EVICT FROM cars WHERE _id = 'car2'")

	action, err := d.Store().Transaction(
		nil, func(tx *ditto.Transaction) (ditto.TransactionCompletionAction, error) {
			// Insert a document
			_, err := tx.Execute(
				"INSERT INTO cars DOCUMENTS (:doc) ON ID CONFLICT DO UPDATE",
				ditto.QueryArguments{
					"doc": ditto.Document{
						"_id":   "car2",
						"make":  "Honda",
						"model": "Civic",
						"year":  2023,
					},
				},
			)
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}

			// Explicitly rollback
			fmt.Println("Explicitly requesting rollback...")
			return ditto.TransactionCompletionActionRollback, nil
		},
	)

	if err != nil {
		log.Fatalf("transaction failed: %w", err)
	}

	fmt.Printf("Transaction completed with action: %v\n", action)

	// Verify the document was NOT inserted
	result, err := d.Store().Execute("SELECT * FROM cars WHERE _id = 'car2'")
	if err != nil {
		log.Fatalf("query failed: %w", err)
	}
	defer result.Close()

	found := false
	for range result.Items() {
		found = true
		break
	}
	if !found {
		fmt.Println("Verified: Document was successfully rolled back")
	} else {
		fmt.Println("Warning: Document found despite rollback")
	}

	return nil
}

func demoImplicitRollback(d *ditto.Ditto) error {
	fmt.Println("Inserting document 'car3' then returning an error...")

	// Delete any existing car3 to ensure clean test
	_, _ = d.Store().Execute("EVICT FROM cars WHERE _id = 'car3'")

	action, err := d.Store().Transaction(
		nil, func(tx *ditto.Transaction) (ditto.TransactionCompletionAction, error) {
			// Insert a document
			_, err := tx.Execute(
				"INSERT INTO cars DOCUMENTS (:doc) ON ID CONFLICT DO UPDATE",
				ditto.QueryArguments{
					"doc": ditto.Document{
						"_id":   "car3",
						"make":  "Ford",
						"model": "Mustang",
						"year":  2023,
					},
				},
			)
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}

			// Return an error to trigger implicit rollback
			return ditto.TransactionCompletionActionCommit, errors.New("simulated error")
		},
	)

	if err != nil {
		fmt.Printf("Transaction failed as expected with action %v: %v\n", action, err)
	} else {
		log.Fatalf("transaction unexpectedly completed without error with action: %v", action)
	}

	// Verify the document was NOT inserted
	result, err := d.Store().Execute("SELECT * FROM cars WHERE _id = 'car3'")
	if err != nil {
		log.Fatalf("query failed: %w", err)
	}
	defer result.Close()

	found := false
	for range result.Items() {
		found = true
		break
	}
	if !found {
		fmt.Println("Verified: Document was implicitly rolled back on error")
	} else {
		fmt.Println("Warning: Document found despite error")
	}

	return nil
}

func demoReadOnlyTransaction(d *ditto.Ditto) error {
	fmt.Println("Querying documents in a read-only transaction...")

	opts := &ditto.TransactionOptions{
		IsReadOnly: true,
		Hint:       "Read-only query example",
	}

	action, err := d.Store().Transaction(
		opts, func(tx *ditto.Transaction) (ditto.TransactionCompletionAction, error) {
			// Get transaction info
			info := tx.Info()
			fmt.Printf(
				"Transaction ID: %s, IsReadOnly: %v, Hint: %v\n",
				info.ID, info.IsReadOnly, *info.Hint,
			)

			// Query data (read-only)
			result, err := tx.Execute("SELECT * FROM cars")
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}
			defer result.Close()

			// Collect items to count them
			var itemSlice []*ditto.QueryResultItem
			for _, item := range result.Items() {
				itemSlice = append(itemSlice, item)
			}
			fmt.Printf("Found %d cars in read-only transaction\n", len(itemSlice))
			for _, item := range itemSlice {
				fmt.Printf("  - %v\n", item.Value())
			}

			return ditto.TransactionCompletionActionCommit, nil
		},
	)

	if err != nil {
		log.Fatalf("read-only transaction failed: %w", err)
	}

	fmt.Printf("Transaction completed with action: %v\n", action)
	return nil
}

func demoTransactionWithResult(d *ditto.Ditto) error {
	fmt.Println("Using TransactionWithResult to insert and return a value...")

	// Use the generic TransactionWithResult function to return a string from within the transaction
	result, err := ditto.TransactionWithResult(
		d.Store(), nil, func(tx *ditto.Transaction) (string, error) {
			// Insert a document
			_, err := tx.Execute(
				"INSERT INTO cars DOCUMENTS (:doc) ON ID CONFLICT DO UPDATE",
				ditto.QueryArguments{
					"doc": ditto.Document{
						"_id":   "car4",
						"make":  "Tesla",
						"model": "Model 3",
						"year":  2024,
					},
				},
			)
			if err != nil {
				return "", err
			}

			// Return a result value
			return "Successfully inserted Tesla Model 3", nil
		},
	)

	if err != nil {
		log.Fatalf("TransactionWithResult failed: %w", err)
	}

	if result == "" {
		log.Fatalf("TransactionWithResult returned empty string")
	}

	fmt.Printf("Result: %s\n", result)
	return nil
}

func demoMultipleOperations(d *ditto.Ditto) error {
	fmt.Println("Performing multiple operations in a single transaction...")

	action, err := d.Store().Transaction(
		nil, func(tx *ditto.Transaction) (ditto.TransactionCompletionAction, error) {
			// Operation 1: Insert a new car
			_, err := tx.Execute(
				"INSERT INTO cars DOCUMENTS (:doc) ON ID CONFLICT DO UPDATE",
				ditto.QueryArguments{
					"doc": ditto.Document{
						"_id":   "car5",
						"make":  "BMW",
						"model": "X5",
						"year":  2024,
						"price": 65000,
					},
				},
			)
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}
			fmt.Println("  - Inserted BMW X5")

			// Operation 2: Query current inventory
			result, err := tx.Execute("SELECT * FROM cars")
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}

			fmt.Printf("  - Current inventory: %d cars\n", result.ItemCount())

			// Operation 3: Update a car's price
			_, err = tx.Execute(
				"UPDATE cars SET price = :newPrice WHERE _id = 'car5'",
				ditto.QueryArguments{"newPrice": 62000},
			)
			if err != nil {
				return ditto.TransactionCompletionActionRollback, err
			}
			fmt.Println("  - Updated BMW X5 price to $62,000")

			// All operations successful
			return ditto.TransactionCompletionActionCommit, nil
		},
	)

	if err != nil {
		log.Fatalf("transaction failed: %w", err)
	}

	fmt.Printf("Transaction completed with action: %v\n", action)
	return nil
}
