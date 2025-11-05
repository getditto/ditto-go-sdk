package tools

import (
	"context"
	"fmt"
	"runtime/trace"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// DeleteArgs holds arguments for the delete command
type DeleteArgs struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Query      string `json:"query"`
}

// DoDelete deletes documents from a collection
func DoDelete(ctx context.Context, config *Config, args *DeleteArgs) error {
	ctx, task := trace.NewTask(ctx, "DoDelete")
	defer task.End()
	defer trace.StartRegion(ctx, "DoDelete").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Collection == "" {
		return fmt.Errorf("collection name is required for delete")
	}
	if args.ID == "" && args.Query == "" {
		return fmt.Errorf("either ID or query condition is required for delete")
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

	// Build DELETE query
	query := fmt.Sprintf("DELETE FROM %s", args.Collection)
	var queryArgs ditto.QueryArguments

	// Add WHERE clause
	if args.ID != "" {
		query += " WHERE _id = :id"
		queryArgs = ditto.QueryArguments{"id": args.ID}
	} else if args.Query != "" {
		query += fmt.Sprintf(" WHERE %s", args.Query)
	}

	result, err := toolkit.Execute(query, queryArgs)
	if err != nil {
		return fmt.Errorf("failed to delete document(s): %w", err)
	}
	defer result.Close()

	// Get mutated items count if available
	mutatedIDs := result.MutatedDocumentIDs()
	if len(mutatedIDs) > 0 {
		fmt.Printf("Deleted %d document(s)\n", len(mutatedIDs))
		for _, id := range mutatedIDs {
			fmt.Printf("  Deleted: %v\n", id)
		}
	} else {
		fmt.Println("Delete operation completed (no documents matched)")
	}

	return nil
}
