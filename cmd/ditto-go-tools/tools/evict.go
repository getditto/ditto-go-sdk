package tools

import (
	"context"
	"fmt"
	"runtime/trace"

	"github.com/getditto/ditto-go-sdk/ditto"
)

// EvictArgs holds arguments for the evict command
type EvictArgs struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Query      string `json:"query"`
}

// DoEvict evicts documents from local storage
func DoEvict(ctx context.Context, config *Config, args *EvictArgs) error {
	ctx, task := trace.NewTask(ctx, "DoEvict")
	defer task.End()
	defer trace.StartRegion(ctx, "DoEvict").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Collection == "" {
		return fmt.Errorf("collection name is required for evict")
	}
	if args.ID == "" && args.Query == "" {
		return fmt.Errorf("either ID or query condition is required for evict")
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

	// Build EVICT query
	query := fmt.Sprintf("EVICT FROM %s", args.Collection)
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
		return fmt.Errorf("failed to evict document(s): %w", err)
	}
	defer result.Close()

	// Get mutated items count if available (eviction is a mutation)
	mutatedIDs := result.MutatedDocumentIDs()
	if len(mutatedIDs) > 0 {
		fmt.Printf("Evicted %d document(s) from local storage\n", len(mutatedIDs))
		for _, id := range mutatedIDs {
			fmt.Printf("  Evicted: %v\n", id)
		}
	} else {
		fmt.Println("Evict operation completed (no documents matched)")
	}

	return nil
}
