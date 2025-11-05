package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/trace"
	"strings"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// UpdateArgs holds arguments for the update command
type UpdateArgs struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Query      string `json:"query"`
	Data       string `json:"data"`
}

// DoUpdate updates documents in a collection
func DoUpdate(ctx context.Context, config *Config, args *UpdateArgs) error {
	ctx, task := trace.NewTask(ctx, "DoUpdate")
	defer task.End()
	defer trace.StartRegion(ctx, "DoUpdate").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Collection == "" {
		return fmt.Errorf("collection name is required for update")
	}
	if args.Data == "" {
		return fmt.Errorf("data is required for update")
	}
	if args.ID == "" && args.Query == "" {
		return fmt.Errorf("either ID or query condition is required for update")
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

	// Parse the update data
	var updateData ditto.QueryArguments
	if err := json.Unmarshal([]byte(args.Data), &updateData); err != nil {
		return fmt.Errorf("invalid JSON for update data: %w", err)
	}

	// Build UPDATE query for partial updates
	// Create SET clause from the update data
	setClauses := []string{}
	queryArgs := make(ditto.QueryArguments)
	for key, value := range updateData {
		if key != "_id" { // Don't update the _id field
			setClauses = append(setClauses, fmt.Sprintf("%s = :%s", key, key))
			queryArgs[key] = value
		}
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s", args.Collection, strings.Join(setClauses, ", "))

	// Add WHERE clause
	if args.ID != "" {
		query += fmt.Sprintf(" WHERE _id = :id")
		queryArgs["id"] = args.ID
	} else if args.Query != "" {
		query += fmt.Sprintf(" WHERE %s", args.Query)
	}

	result, err := toolkit.Execute(query, queryArgs)
	if err != nil {
		return fmt.Errorf("failed to update document(s): %w", err)
	}
	defer result.Close()

	// Check for mutated documents
	mutatedIDs := result.MutatedDocumentIDs()
	if len(mutatedIDs) > 0 {
		fmt.Printf("Updated %d document(s)\n", len(mutatedIDs))
	} else {
		fmt.Println("Update completed (no documents matched)")
	}

	return nil
}
