package tools

import (
	"context"
	"fmt"
	"runtime/trace"
)

// CountArgs holds arguments for the count command
type CountArgs struct {
	Collection string `json:"collection"`
	Query      string `json:"query"`
}

// DoCount counts documents in a collection
func DoCount(ctx context.Context, config *Config, args *CountArgs) error {
	ctx, task := trace.NewTask(ctx, "DoCount")
	defer task.End()
	defer trace.StartRegion(ctx, "DoCount").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Collection == "" {
		return fmt.Errorf("collection name is required for count")
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

	// Build COUNT query
	query := fmt.Sprintf("SELECT COUNT(*) AS _count FROM %s", args.Collection)

	// Add WHERE clause if query condition is provided
	if args.Query != "" {
		query += fmt.Sprintf(" WHERE %s", args.Query)
	}

	result, err := toolkit.Execute(query, nil)
	if err != nil {
		return fmt.Errorf("failed to count documents: %w", err)
	}
	defer result.Close()

	// Extract count from result by iterating over the items
	var count interface{}
	found := false
	for _, item := range result.Items() {
		// The result should contain a single row with the count
		doc := item.Value()
		count = doc["_count"]
		found = true
		break
	}

	if found {
		fmt.Printf("Count: %v\n", count)
	} else {
		fmt.Println("Count: 0")
	}

	return nil
}
