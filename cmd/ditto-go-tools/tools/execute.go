// Copyright 2025 DittoLive Incorporated. All rights reserved.

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/trace"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// ExecuteArgs holds arguments for the execute command
type ExecuteArgs struct {
	Query      string `json:"query"`
	Data       string `json:"data"`
	Attachment string `json:"attachment"`
	Metadata   string `json:"metadata"`
	Format     string `json:"format"`
}

// DoExecute executes a synchronous query
func DoExecute(ctx context.Context, config *Config, args *ExecuteArgs) error {
	ctx, task := trace.NewTask(ctx, "DoExecute")
	defer task.End()
	defer trace.StartRegion(ctx, "DoExecute").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.Query == "" {
		return fmt.Errorf("a query must be provided to execute")
	}
	if args.Attachment != "" {
		return fmt.Errorf("attachment is currently not supported by execute")
	}
	if args.Metadata != "" {
		return fmt.Errorf("metadata is currently not supported by execute")
	}

	if args.Format != "json" && args.Format != "table" && args.Format != "csv" {
		return fmt.Errorf("invalid format: must be 'json', 'table', or 'csv'")
	}

	toolkit, err := NewToolkit(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create toolkit: %w", err)
	}
	defer toolkit.Close()

	if err := toolkit.StartSync(); err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}

	var queryArgs ditto.QueryArguments
	if args.Data != "" {
		if err := json.Unmarshal([]byte(args.Data), &queryArgs); err != nil {
			return fmt.Errorf("invalid JSON query parameters: %w", err)
		}
	}

	result, err := toolkit.Execute(args.Query, queryArgs)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer result.Close()

	err = printQueryResult(result, args)
	if err != nil {
		return err
	}

	return nil
}

func printQueryResult(result *ditto.QueryResult, args *ExecuteArgs) error {
	resultData, err := queryResultDataFromQueryResult(result)
	if err != nil {
		return fmt.Errorf("failed to transform query result: %w", err)
	}

	// Format output based on format option
	switch args.Format {
	case "json":
		jsonStr, err := toJSONString(resultData)
		if err != nil {
			return fmt.Errorf("failed to convert result to JSON: %w", err)
		}
		fmt.Println(jsonStr)
	case "table":
		fmt.Print(formatTableOutput(resultData))
	case "csv":
		csvStr, err := formatCSVOutput(resultData)
		if err != nil {
			return fmt.Errorf("failed to format CSV output: %w", err)
		}
		fmt.Print(csvStr)
	}
	return nil
}
