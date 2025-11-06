// Copyright 2025 DittoLive Incorporated. All rights reserved.

package tools

import (
	"context"
	"fmt"
	"os"
	"runtime/trace"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// ExportLogsArgs holds arguments for the export-logs command
type ExportLogsArgs struct {
	OutputPath     string `json:"output_path"`
	ForceOverwrite bool   `json:"force_overwrite"`
}

// DoExportLogs exports logs as jsonl.gz archive
func DoExportLogs(ctx context.Context, config *Config, args *ExportLogsArgs) error {
	ctx, task := trace.NewTask(ctx, "DoExportLogs")
	defer task.End()
	defer trace.StartRegion(ctx, "DoExportLogs").End()

	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if args.OutputPath == "" {
		return fmt.Errorf("output path is required for export logs")
	}

	if args.ForceOverwrite {
		if err := os.Remove(args.OutputPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}

	// Export logs using the global ExportLog function
	lineCount, err := ditto.ExportLog(args.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to export logs: %w", err)
	}

	fmt.Printf("Exported %d log lines to %s\n", lineCount, args.OutputPath)
	return nil
}
