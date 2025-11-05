// This program demonstrates the basics of initializing Ditto and running DQL queries against the local database.
// This does not sync data with any other instances of Ditto.
package main

import (
	"log"
	"os"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// Configuration
var (
	persistenceDir = "/tmp/ditto-go-example-execute-example"
)

func main() {
	log.Println("=== Ditto DQL Execute Example ===")
	log.Println("This example demonstrates various DQL EXECUTE statements")
	log.Println()

	// Clean up persistence directory from previous runs
	os.RemoveAll(persistenceDir)

	// Configure Ditto with offline identity (no sync needed for local queries)
	config := ditto.DefaultDittoConfig().
		WithDatabaseID("ditto-go-example-execute-example").
		WithPersistenceDirectory(persistenceDir).
		WithConnect(&ditto.DittoConfigConnectSmallPeersOnly{})

	// Create Ditto instance
	dit, err := ditto.Open(config)
	if err != nil {
		log.Fatalf("Failed to create Ditto: %v", err)
	}
	defer dit.Close()

	log.Println("✓ Ditto instance created with offline identity")
	log.Println()

	ditto.SetMinimumLogLevel(ditto.LogLevelWarning)

	// Run various DQL execute examples
	demonstrateInsert(dit)
	demonstrateSelect(dit)
	demonstrateUpdate(dit)
	demonstrateEvict(dit)

	log.Println()
	log.Println("=== All DQL Execute examples completed ===")
}

// demonstrateInsert shows various INSERT operations
func demonstrateInsert(dit *ditto.Ditto) {
	log.Println("--- INSERT Operations ---")

	// Insert a single document
	result, err := dit.Store().Execute(
		"INSERT INTO cars DOCUMENTS (:doc)",
		ditto.QueryArguments{
			"doc": ditto.Document{
				"_id":     "car1",
				"make":    "Toyota",
				"model":   "Camry",
				"year":    2022,
				"color":   "blue",
				"miles":   15000,
				"inStock": true,
			},
		},
	)
	if err != nil {
		log.Printf("Error inserting single document: %v", err)
		return
	}
	defer result.Close()
	log.Printf("  Inserted document with ID: %v", result.MutatedDocumentIDs())

	// Insert multiple documents
	result2, err := dit.Store().Execute(
		"INSERT INTO cars DOCUMENTS (:doc1), (:doc2), (:doc3)",
		ditto.QueryArguments{
			"doc1": ditto.Document{
				"_id":     "car2",
				"make":    "Honda",
				"model":   "Accord",
				"year":    2023,
				"color":   "red",
				"miles":   5000,
				"inStock": true,
			},
			"doc2": ditto.Document{
				"_id":     "car3",
				"make":    "Tesla",
				"model":   "Model 3",
				"year":    2024,
				"color":   "white",
				"miles":   0,
				"inStock": true,
			},
			"doc3": ditto.Document{
				"_id":     "car4",
				"make":    "Ford",
				"model":   "F-150",
				"year":    2021,
				"color":   "black",
				"miles":   25000,
				"inStock": false,
			},
		},
	)
	if err != nil {
		log.Printf("Error inserting multiple documents: %v", err)
		return
	}
	defer result2.Close()
	log.Printf("  Inserted %d documents with IDs: %v", len(result2.MutatedDocumentIDs()), result2.MutatedDocumentIDs())
	log.Println()
}

// demonstrateSelect shows various SELECT operations
func demonstrateSelect(dit *ditto.Ditto) {
	log.Println("--- SELECT Operations ---")

	// Select all documents
	result, err := dit.Store().Execute("SELECT * FROM cars")
	if err != nil {
		log.Printf("Error selecting all: %v", err)
		return
	}
	defer result.Close()
	log.Printf("  SELECT * returned %d cars", result.ItemCount())
	for i, item := range result.Items() {
		doc := item.Value()
		log.Printf("    Car %d: %s %s (%v)", i+1, doc["make"], doc["model"], doc["year"])
	}

	// Select with WHERE clause
	result2, err := dit.Store().Execute(
		"SELECT * FROM cars WHERE year > :minYear AND inStock = true",
		ditto.QueryArguments{"minYear": 2022},
	)
	if err != nil {
		log.Printf("Error selecting with WHERE: %v", err)
		return
	}
	defer result2.Close()
	log.Printf("  Found %d cars newer than 2022 in stock", result2.ItemCount())

	// Select specific fields
	result3, err := dit.Store().Execute("SELECT make, model, year FROM cars ORDER BY year DESC")
	if err != nil {
		log.Printf("Error selecting with ORDER BY: %v", err)
		return
	}
	defer result3.Close()
	log.Printf("  Cars ordered by year (newest first):")
	for _, item := range result3.Items() {
		doc := item.Value()
		log.Printf("    %v %s %s", doc["year"], doc["make"], doc["model"])
	}

	// Select with LIMIT
	result4, err := dit.Store().Execute("SELECT * FROM cars LIMIT 2")
	if err != nil {
		log.Printf("Error selecting with LIMIT: %v", err)
		return
	}
	defer result4.Close()
	log.Printf("  Limited query returned %d cars", result4.ItemCount())
	log.Println()
}

// demonstrateUpdate shows UPDATE operations
func demonstrateUpdate(dit *ditto.Ditto) {
	log.Println("--- UPDATE Operations ---")

	// Update a single document
	result, err := dit.Store().Execute(
		"UPDATE cars SET color = :newColor WHERE _id = :id",
		ditto.QueryArguments{
			"newColor": "silver",
			"id":       "car1",
		},
	)
	if err != nil {
		log.Printf("Error updating document: %v", err)
		return
	}
	defer result.Close()
	log.Printf("  Updated car1 color to silver and increased miles")

	// Update multiple documents
	result2, err := dit.Store().Execute(
		"UPDATE cars SET inStock = :status WHERE miles > :threshold",
		ditto.QueryArguments{
			"status":    false,
			"threshold": 20000,
		},
	)
	if err != nil {
		log.Printf("Error updating multiple: %v", err)
		return
	}
	defer result2.Close()
	log.Printf("  Marked %d high-mileage cars as not in stock", len(result2.MutatedDocumentIDs()))

	// Verify updates
	verifyResult, err := dit.Store().Execute("SELECT _id, color, miles, inStock FROM cars WHERE _id = 'car1'")
	if err != nil {
		log.Printf("Error verifying update: %v", err)
		return
	}
	defer verifyResult.Close()
	if verifyResult.ItemCount() > 0 {
		doc := verifyResult.Item(0).Value()
		log.Printf("  Verified car1: color=%s, miles=%v, inStock=%v", doc["color"], doc["miles"], doc["inStock"])
	}
	log.Println()
}

// demonstrateEvict shows EVICT operations
func demonstrateEvict(dit *ditto.Ditto) {
	log.Println("--- EVICT Operations ---")

	// Count before eviction
	countBefore, err := selectCount(dit, "SELECT * FROM cars")
	if err != nil {
		log.Printf("Error counting: %v", err)
		return
	}
	log.Printf("  Cars before eviction: %d", countBefore)

	// Evict specific document
	result, err := dit.Store().Execute(
		"EVICT FROM cars WHERE _id = :id",
		ditto.QueryArguments{"id": "car3"},
	)
	if err != nil {
		log.Printf("Error deleting document: %v", err)
		return
	}
	defer result.Close()
	log.Printf("  Evicted car4")

	// Evict with condition
	result2, err := dit.Store().Execute(
		"EVICT FROM cars WHERE inStock = false AND miles > :maxMiles",
		map[string]any{"maxMiles": 15000},
	)
	if err != nil {
		log.Printf("Error deleting with condition: %v", err)
		return
	}
	defer result2.Close()
	log.Printf("  Evicted %d cars not in stock with high mileage", len(result2.MutatedDocumentIDs()))

	// Count after evict
	countAfter, err := selectCount(dit, "SELECT * FROM cars")
	if err != nil {
		log.Printf("Error counting: %v", err)
		return
	}
	log.Printf("  Cars after evictions: %d", countAfter)
	log.Println()
}

// Helper function to count number of items that satisfy a query
//
// DQL doesn't support "SELECT COUNT(*) FROM ..." like SQL yet, so
// we have to do this instead.
func selectCount(dit *ditto.Ditto, query string) (int, error) {
	result, err := dit.Store().Execute(query)
	if err != nil {
		return 0, err
	}
	defer result.Close()
	return result.ItemCount(), nil
}
