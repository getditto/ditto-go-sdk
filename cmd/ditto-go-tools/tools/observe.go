package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/trace"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// ObserveArgs holds arguments for the observe command
type ObserveArgs struct {
	Query          []string `json:"query"`
	TimeoutSec     int32    `json:"timeout_sec"`
	SaveAttachment bool     `json:"save_attachment"`
}

// DoObserve observes data for specified queries
func DoObserve(ctx context.Context, config *Config, args *ObserveArgs) error {
	ctx, task := trace.NewTask(ctx, "DoObserve")
	defer task.End()
	defer trace.StartRegion(ctx, "DoObserve").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if len(args.Query) == 0 {
		return fmt.Errorf("one or more queries must be provided to observe")
	}
	if args.SaveAttachment {
		return fmt.Errorf("save is currently not supported for observe")
	}

	toolkit, err := NewToolkit(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create toolkit: %w", err)
	}
	defer toolkit.Close()

	if err := toolkit.StartSync(); err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}

	// Register subscriptions and observers for each query
	var subscriptions []*ditto.SyncSubscription
	var observers []*ditto.StoreObserver

	// Clean up resources on exit
	// Cancel subscriptions first to stop data flow, then cancel observers
	defer func() {
		for _, subscription := range subscriptions {
			if subscription != nil {
				subscription.Cancel()
			}
		}
		for _, observer := range observers {
			if observer != nil {
				observer.Cancel()
			}
		}
	}()

	for _, query := range args.Query {
		// Create a copy of query for the closure
		q := query

		// Register subscription
		subscription, err := toolkit.RegisterSubscription(q, nil)
		if err != nil {
			return fmt.Errorf("failed to register subscription for query %q: %w", q, err)
		}
		subscriptions = append(subscriptions, subscription)

		// Register observer
		observer, err := toolkit.RegisterObserver(
			q, nil, func(result *ditto.QueryResult) {
				defer result.Close() // Always close the result
				fmt.Printf("Query result received for: %s\n", q)
				items := result.Items()
				for _, item := range items {
					// Convert item to JSON string for display
					if jsonBytes, err := json.Marshal(item.Value); err == nil {
						fmt.Printf("  %s\n", string(jsonBytes))
					} else {
						fmt.Printf("  <error marshaling item: %v>\n", err)
					}
				}
			},
		)
		if err != nil {
			return fmt.Errorf("failed to register observer for query %q: %w", q, err)
		}
		observers = append(observers, observer)
	}

	// Quit on timeout or SIGINT/SIGTERM
	quit := timeoutOrInterrupt(time.Duration(args.TimeoutSec) * time.Second)
	<-quit

	return nil
}
