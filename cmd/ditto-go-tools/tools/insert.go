package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime/trace"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// InsertArgs holds arguments for the insert command
type InsertArgs struct {
	Collection string `json:"collection"`
	Data       string `json:"data"`
	Batch      bool   `json:"batch"`
	Attachment string `json:"attachment"`
	Upsert     bool   `json:"upsert"`
}

// DoInsert inserts documents into a collection
func DoInsert(ctx context.Context, config *Config, args *InsertArgs) error {
	ctx, task := trace.NewTask(ctx, "DoInsert")
	defer task.End()
	defer trace.StartRegion(ctx, "DoInsert").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Collection == "" {
		return fmt.Errorf("collection name is required for insert")
	}
	if args.Data == "" {
		return fmt.Errorf("data is required for insert")
	}
	if args.Attachment != "" {
		return fmt.Errorf("attachment support not yet implemented for insert")
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
	defer toolkit.StopSync()

	query := fmt.Sprintf("INSERT INTO %s DOCUMENTS (:doc)", args.Collection)
	if args.Upsert {
		query += " ON ID CONFLICT DO UPDATE"
	}

	if args.Batch {
		var docs []ditto.Document
		if err := json.Unmarshal([]byte(args.Data), &docs); err != nil {
			return fmt.Errorf("invalid JSON array for batch insert: %w", err)
		}

		// Validate documents
		for i, doc := range docs {
			if doc == nil {
				return fmt.Errorf("document at index %d is null", i)
			}
		}

		successCount := 0
		failureCount := 0
		for i, doc := range docs {
			_, err := toolkit.Execute(query, ditto.QueryArguments{"doc": doc})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to insert document %d: %v\n", i, err)
				failureCount++
			} else {
				successCount++
			}
		}

		fmt.Printf("Inserted %d of %d documents\n", successCount, len(docs))

		if failureCount > 0 {
			return fmt.Errorf("batch insert failed: %d of %d documents failed to insert", failureCount, len(docs))
		}
	} else {
		// Single document insert
		var doc ditto.Document
		if err := json.Unmarshal([]byte(args.Data), &doc); err != nil {
			return fmt.Errorf("invalid JSON for document: %w", err)
		}

		result, err := toolkit.Execute(query, ditto.QueryArguments{"doc": doc})
		if err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}
		defer result.Close()

		fmt.Println("Document inserted successfully")
	}

	return nil
}
