// Copyright 2025 DittoLive Incorporated. All rights reserved.

// This program demonstrates the using Ditto and running DQL queries using native serialization.
// Instead of the general-purpose ditto.Document which can hold any arbitrary structure,
// query arguments may be any type that can be serialized into CBOR.
// Similarly, query results may be deserialized from CBOR into any struct with matching structure.
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
	log.Println("=== Ditto DQL Native Serialization Example ===")
	log.Println("This example demonstrates various DQL EXECUTE statements with native (de)serialization")
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

	log.Println()
	log.Println("=== All DQL Native Serialization Execute examples completed ===")
}

type Car struct {
	// Struct tags may be `cbor:` or `json:`, as supported by "github.com/fxamacker/cbor"
	Id      string `cbor:"_id"`
	Make    string `cbor:"make"`
	Model   string `cbor:"model"`
	Year    int    `cbor:"year"`
	Color   string `cbor:"color"`
	Miles   int    `cbor:"miles"`
	InStock bool   `cbor:"inStock"`
}

// demonstrateInsert shows various INSERT operations, using native serialization
func demonstrateInsert(dit *ditto.Ditto) {
	log.Println("--- INSERT Operations ---")

	// Insert a single document
	result, err := dit.Store().Execute(
		"INSERT INTO cars DOCUMENTS (:doc)",
		ditto.QueryArguments{
			"doc": Car{
				Id:      "car1",
				Make:    "Toyota",
				Model:   "Camry",
				Year:    2022,
				Color:   "blue",
				Miles:   15000,
				InStock: true,
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
			"doc1": Car{
				Id:      "car2",
				Make:    "Honda",
				Model:   "Accord",
				Year:    2023,
				Color:   "red",
				Miles:   5000,
				InStock: true,
			},
			"doc2": Car{
				Id:      "car3",
				Make:    "Tesla",
				Model:   "Model 3",
				Year:    2024,
				Color:   "white",
				Miles:   0,
				InStock: true,
			},
			"doc3": Car{
				Id:      "car4",
				Make:    "Ford",
				Model:   "F-150",
				Year:    2021,
				Color:   "black",
				Miles:   25000,
				InStock: false,
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

// demonstrateSelect shows various SELECT operations, using native deserialization
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
		var car Car
		if err := item.UnmarshalTo(&car); err != nil {
			log.Fatalf("Error unmarshalling item #%d: %v", i, err)
		}
		log.Printf("    Car %d: %s %s (%v)", i+1, car.Make, car.Model, car.Year)
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
	for i, item := range result3.Items() {
		// Anonymous structs can be used too
		var result struct {
			Make  string `cbor:"make"`
			Model string `cbor:"model"`
			Year  int    `cbor:"year"`
		}
		if err := item.UnmarshalTo(&result); err != nil {
			log.Fatalf("Error unmarshalling item #%d: %v", i, err)
		}
		log.Printf("    %v %s %s", result.Year, result.Make, result.Model)
	}
}
