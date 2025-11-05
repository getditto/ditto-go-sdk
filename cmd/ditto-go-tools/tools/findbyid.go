package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/trace"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// FindByIDArgs holds arguments for the find-by-id command
type FindByIDArgs struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Format     string `json:"format"`
}

// DoFindByID finds a document by its ID
func DoFindByID(ctx context.Context, config *Config, args *FindByIDArgs) error {
	ctx, task := trace.NewTask(ctx, "DoFindByID")
	defer task.End()
	defer trace.StartRegion(ctx, "DoFindByID").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Collection == "" {
		return fmt.Errorf("collection name is required for find-by-id")
	}
	if args.ID == "" {
		return fmt.Errorf("document ID is required for find-by-id")
	}

	// Validate format
	if args.Format != "json" && args.Format != "pretty" {
		return fmt.Errorf("invalid format: must be 'json' or 'pretty'")
	}

	if !isValidCollectionName(args.Collection) {
		return fmt.Errorf("invalid collection name: %q", args.Collection)
	}

	toolkit, err := NewToolkit(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create toolkit: %w", err)
	}
	defer toolkit.Close()

	if err := toolkit.StartSync(); err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}

	// Build SELECT query to find document by ID
	query := fmt.Sprintf("SELECT * FROM %s WHERE _id = :id", args.Collection)
	queryArgs := ditto.QueryArguments{"id": args.ID}

	result, err := toolkit.Execute(query, queryArgs)
	if err != nil {
		return fmt.Errorf("failed to find document: %w", err)
	}
	defer result.Close()

	// Get the first document from result by iterating over the items
	var doc ditto.Document
	found := false
	for _, item := range result.Items() {
		doc = item.Value()
		found = true
		break
	}

	if !found {
		fmt.Printf("Document with ID '%s' not found in collection '%s'\n", args.ID, args.Collection)
		return nil
	}

	// Format and display the document
	if args.Format == "pretty" {
		// Pretty print with indentation
		jsonBytes, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format document: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// Compact JSON
		jsonBytes, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to format document: %w", err)
		}
		fmt.Println(string(jsonBytes))
	}

	return nil
}
