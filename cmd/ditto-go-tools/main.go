// Copyright 2025 DittoLive Incorporated. All rights reserved.

// ditto-go-tools is a command-line testing utility for the Ditto Go SDK.
//
// It provides interactive commands for testing SDK features including:
//   - Executing DQL queries with multiple output formats
//   - Observing real-time data changes
//   - Managing documents (insert, update, delete, evict, count, find)
//   - Monitoring peer presence
//   - Exporting diagnostic logs
//   - Running smoke tests
//
// The tool supports multiple authentication modes (small-peers-only, online-playground,
// shared-secret, online-with-authentication) and can load configuration from environment
// variables, .env files, TOML config files, or command-line flags.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"ditto-go-tools/tools"
)

var config *tools.Config
var noDotenv bool
var noEnvvars bool
var dotenvPath string
var traceFile *os.File
var cpuprofileFile *os.File

func main() {
	ctx, task := trace.NewTask(context.Background(), "ditto-go-tools")
	defer task.End()

	config = tools.NewDefaultConfig()

	// Build auth system help text using constants
	authSystemHelp := fmt.Sprintf(
		"Authentication system (%s, %s, %s, %s)",
		tools.AuthSystemSmallPeersOnly,
		tools.AuthSystemOnlineWithAuthentication,
		tools.AuthSystemOnlinePlayground,
		tools.AuthSystemSharedSecret,
	)

	var rootCmd = &cobra.Command{
		Use:   "ditto-go-tools",
		Short: "Ditto Go SDK Tools",
		Long:  "Command-line tools for the Ditto Go SDK",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if !noDotenv {
				loadEnvFile()
			}

			if !noEnvvars {
				applyEnvironmentVariables(cmd)
			}

			if config.Trace != "" {
				var err error
				traceFile, err = os.Create(config.Trace)
				if err != nil {
					return fmt.Errorf("failed to create trace file: %w", err)
				}
				if err := trace.Start(traceFile); err != nil {
					traceFile.Close()
					return fmt.Errorf("failed to start trace: %w", err)
				}
				fmt.Printf("info: trace output enabled to %s\n", config.Trace)
			}

			if config.CPUProfile != "" {
				var err error
				cpuprofileFile, err = os.Create(config.CPUProfile)
				if err != nil {
					return fmt.Errorf("failed to create CPU profile file: %w", err)
				}
				if err := pprof.StartCPUProfile(cpuprofileFile); err != nil {
					cpuprofileFile.Close()
					return fmt.Errorf("failed to start CPU profile: %w", err)
				}
				fmt.Printf("info: CPU profiling enabled to %s\n", config.CPUProfile)
			}

			return nil
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(
		&config.DatabaseID, "app-id", config.DatabaseID,
		"Ditto app ID (deprecated, use --db-id)",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.DatabaseID, "db-id", config.DatabaseID,
		"Ditto database ID",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.AuthSystem, "auth-system", config.AuthSystem,
		authSystemHelp,
	)
	rootCmd.PersistentFlags().StringVarP(
		&config.ConfigFile, "config", "c", config.ConfigFile,
		"Path to TOML configuration file",
	)
	rootCmd.PersistentFlags().StringVarP(
		&config.LogLevel, "log-level", "l", config.LogLevel,
		"Log level (error, warn, info, debug)",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.LogFile, "log-file", config.LogFile,
		"Ditto log file",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.PlaygroundToken, "playground-token", config.PlaygroundToken,
		"Playground token",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.SharedToken, "shared-token", config.SharedToken,
		"Shared token",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.SecretKey, "secret-key", config.SecretKey,
		"Secret key",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.OfflineOnlyLicenseToken, "offline-only-license-token", config.OfflineOnlyLicenseToken,
		fmt.Sprintf("Offline-only license token for %s authentication", tools.AuthSystemSmallPeersOnly),
	)
	rootCmd.PersistentFlags().StringVar(
		&config.PersistentRoot, "persistent-root", config.PersistentRoot,
		"Persistent root directory",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.AuthURL, "auth-url", config.AuthURL,
		"Authentication URL for server connections",
	)
	rootCmd.PersistentFlags().StringSliceVar(
		&config.WebsocketURLs, "websocket-urls", config.WebsocketURLs,
		"WebSocket URLs",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.AuthEventHandler, "auth-event-handler", config.AuthEventHandler,
		"Authentication event handler",
	)
	rootCmd.PersistentFlags().BoolVar(
		&config.EnableBLE, "enable-ble", config.EnableBLE,
		"Enable Bluetooth LE",
	)
	rootCmd.PersistentFlags().BoolVar(
		&config.EnableLAN, "enable-lan", config.EnableLAN,
		"Enable LAN",
	)
	rootCmd.PersistentFlags().BoolVar(
		&config.TCPListener, "tcp-listener", config.TCPListener,
		"Enable TCP listener",
	)
	rootCmd.PersistentFlags().Uint16Var(
		&config.TCPListenerPort, "tcp-listener-port", config.TCPListenerPort,
		"TCP listener port",
	)
	rootCmd.PersistentFlags().StringVarP(
		&config.Name, "name", "n", config.Name,
		"Set device name",
	)
	rootCmd.PersistentFlags().BoolVar(
		&noDotenv, "no-dotenv", false,
		"Do not load .env file",
	)
	rootCmd.PersistentFlags().StringVar(
		&dotenvPath, "dotenv-path", "",
		"Path to .env file (defaults to ./.env)",
	)
	rootCmd.PersistentFlags().BoolVar(
		&noEnvvars, "no-envvars", false,
		"Do not read environment variables",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.Trace, "trace", "",
		"Enable runtime trace output to the specified file",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.CPUProfile, "cpuprofile", "",
		"Enable CPU profiling output to the specified file",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.MemProfile, "memprofile", "",
		"Write memory allocation profile to the specified file at exit",
	)
	rootCmd.PersistentFlags().StringVar(
		&config.HeapProfile, "heapprofile", "",
		"Write heap profile to the specified file at exit",
	)
	rootCmd.PersistentFlags().BoolVar(
		&config.NoColor, "no-color", false,
		"Disable terminal color output",
	)

	// SDK version command
	var sdkVersionCmd = &cobra.Command{
		Use:   "sdk-version",
		Short: "Print the Ditto SDK version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("SDK version: %s\n", tools.GetSDKVersion())
			return nil
		},
	}
	// Skip validation and env loading for this command
	sdkVersionCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	rootCmd.AddCommand(sdkVersionCmd)

	// Print config command
	var printConfigCmd = &cobra.Command{
		Use:   "print-config",
		Short: "Print the configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.String())
		},
	}
	rootCmd.AddCommand(printConfigCmd)

	// Smoke test command
	var smokeTestCmd = &cobra.Command{
		Use:   "smoke-test",
		Short: "Verify Ditto can initialize and sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoSmokeTest(ctx, config)
		},
	}
	rootCmd.AddCommand(smokeTestCmd)

	// Execute command
	var executeArgs tools.ExecuteArgs
	executeArgs.Format = "json"
	var executeCmd = &cobra.Command{
		Use:   "execute",
		Short: "Execute a synchronous query",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoExecute(ctx, config, &executeArgs)
		},
	}
	executeCmd.Flags().StringVarP(&executeArgs.Query, "query", "q", "", "DQL query (required)")
	executeCmd.Flags().StringVarP(&executeArgs.Data, "data", "d", "", "JSON query parameters")
	executeCmd.Flags().StringVar(&executeArgs.Attachment, "attachment", "", "Attachment parameter")
	executeCmd.Flags().StringVarP(&executeArgs.Metadata, "metadata", "m", "", "Metadata parameter")
	executeCmd.Flags().StringVarP(
		&executeArgs.Format, "format", "f", executeArgs.Format, "Output format (json, table, csv)",
	)
	executeCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(executeCmd)

	// Observe command
	var observeArgs tools.ObserveArgs
	var observeCmd = &cobra.Command{
		Use:   "observe",
		Short: "Observe data for specified query",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoObserve(ctx, config, &observeArgs)
		},
	}
	observeCmd.Flags().StringSliceVarP(
		&observeArgs.Query, "query", "q", nil, "DQL query (required, can specify multiple)",
	)
	observeCmd.Flags().Int32VarP(&observeArgs.TimeoutSec, "timeout", "t", 0, "Timeout in seconds to stop observing")
	observeCmd.Flags().BoolVarP(&observeArgs.SaveAttachment, "save", "s", false, "Save attachment to file")
	observeCmd.MarkFlagRequired("query")
	rootCmd.AddCommand(observeCmd)

	// Presence command
	var presenceArgs tools.PresenceArgs
	presenceArgs.PeerScope = tools.PeerScopeAll
	var presenceCmd = &cobra.Command{
		Use:   "presence",
		Short: "Observe local and/or remote peer metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoPresence(ctx, config, &presenceArgs)
		},
	}
	presenceCmd.Flags().StringVarP(
		&presenceArgs.PeerScope, "peer-scope", "p", presenceArgs.PeerScope, "Peer scope (remote, local, or all)",
	)
	presenceCmd.Flags().Int32Var(&presenceArgs.TimeoutSec, "timeout", 0, "Timeout in seconds to stop observing")
	rootCmd.AddCommand(presenceCmd)

	// Export logs command
	var exportLogsArgs tools.ExportLogsArgs
	var exportLogsCmd = &cobra.Command{
		Use:   "export-logs",
		Short: "Export logs as jsonl.gz archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoExportLogs(ctx, config, &exportLogsArgs)
		},
	}
	exportLogsCmd.Flags().StringVarP(&exportLogsArgs.OutputPath, "output", "o", "", "Output file path (required)")
	exportLogsCmd.Flags().BoolVarP(
		&exportLogsArgs.ForceOverwrite, "force-overwrite", "f", false, "Overwrite any existing file at the output path",
	)
	exportLogsCmd.MarkFlagRequired("output")
	rootCmd.AddCommand(exportLogsCmd)

	// Insert command
	var insertArgs tools.InsertArgs
	var insertCmd = &cobra.Command{
		Use:   "insert",
		Short: "Insert documents into a collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoInsert(ctx, config, &insertArgs)
		},
	}
	insertCmd.Flags().StringVar(&insertArgs.Collection, "collection", "", "Collection name (required)")
	insertCmd.Flags().StringVarP(&insertArgs.Data, "data", "d", "", "JSON data for the document(s) (required)")
	insertCmd.Flags().BoolVarP(&insertArgs.Batch, "batch", "b", false, "Insert multiple documents from JSON array")
	insertCmd.Flags().StringVar(&insertArgs.Attachment, "attachment", "", "Path to attachment file")
	insertCmd.Flags().BoolVar(&insertArgs.Upsert, "upsert", false, "Use ON ID CONFLICT DO UPDATE (upsert mode)")
	insertCmd.MarkFlagRequired("collection")
	insertCmd.MarkFlagRequired("data")
	rootCmd.AddCommand(insertCmd)

	// Update command
	var updateArgs tools.UpdateArgs
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update documents in a collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoUpdate(ctx, config, &updateArgs)
		},
	}
	updateCmd.Flags().StringVar(&updateArgs.Collection, "collection", "", "Collection name (required)")
	updateCmd.Flags().StringVarP(&updateArgs.ID, "id", "i", "", "Document ID to update")
	updateCmd.Flags().StringVarP(&updateArgs.Query, "query", "q", "", "Query condition for update")
	updateCmd.Flags().StringVarP(&updateArgs.Data, "data", "d", "", "JSON data for the update (required)")
	updateCmd.MarkFlagRequired("collection")
	updateCmd.MarkFlagRequired("data")
	rootCmd.AddCommand(updateCmd)

	// Delete command
	var deleteArgs tools.DeleteArgs
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete documents from a collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoDelete(ctx, config, &deleteArgs)
		},
	}
	deleteCmd.Flags().StringVar(&deleteArgs.Collection, "collection", "", "Collection name (required)")
	deleteCmd.Flags().StringVarP(&deleteArgs.ID, "id", "i", "", "Document ID to delete")
	deleteCmd.Flags().StringVarP(&deleteArgs.Query, "query", "q", "", "Query condition for deletion")
	deleteCmd.MarkFlagRequired("collection")
	rootCmd.AddCommand(deleteCmd)

	// Evict command
	var evictArgs tools.EvictArgs
	var evictCmd = &cobra.Command{
		Use:   "evict",
		Short: "Evict documents from local storage",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoEvict(ctx, config, &evictArgs)
		},
	}
	evictCmd.Flags().StringVar(&evictArgs.Collection, "collection", "", "Collection name (required)")
	evictCmd.Flags().StringVarP(&evictArgs.ID, "id", "i", "", "Document ID to evict")
	evictCmd.Flags().StringVarP(&evictArgs.Query, "query", "q", "", "Query condition for eviction")
	evictCmd.MarkFlagRequired("collection")
	rootCmd.AddCommand(evictCmd)

	// Count command
	var countArgs tools.CountArgs
	var countCmd = &cobra.Command{
		Use:   "count",
		Short: "Count documents in a collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoCount(ctx, config, &countArgs)
		},
	}
	countCmd.Flags().StringVar(&countArgs.Collection, "collection", "", "Collection name (required)")
	countCmd.Flags().StringVarP(&countArgs.Query, "query", "q", "", "Query condition for counting")
	countCmd.MarkFlagRequired("collection")
	rootCmd.AddCommand(countCmd)

	// Find by ID command
	var findByIDArgs tools.FindByIDArgs
	findByIDArgs.Format = "json"
	var findByIDCmd = &cobra.Command{
		Use:   "find-by-id",
		Short: "Find a single document by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoFindByID(ctx, config, &findByIDArgs)
		},
	}
	findByIDCmd.Flags().StringVar(&findByIDArgs.Collection, "collection", "", "Collection name (required)")
	findByIDCmd.Flags().StringVarP(&findByIDArgs.ID, "id", "i", "", "Document ID (required)")
	findByIDCmd.Flags().StringVarP(
		&findByIDArgs.Format, "format", "f", findByIDArgs.Format, "Output format (json, pretty)",
	)
	findByIDCmd.MarkFlagRequired("collection")
	findByIDCmd.MarkFlagRequired("id")
	rootCmd.AddCommand(findByIDCmd)

	// DQL REPL command
	var dqlCmd = &cobra.Command{
		Use:   "dql",
		Short: "Start interactive DQL REPL",
		Long:  "Start an interactive Read-Eval-Print Loop for executing DQL queries and special commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tools.DoDQLInterpreter(ctx, config)
		},
	}
	rootCmd.AddCommand(dqlCmd)

	// Setup defer for trace and profiling cleanup
	defer func() {
		if traceFile != nil {
			trace.Stop()
			traceFile.Close()
			fmt.Printf("info: trace output saved to %s\n", config.Trace)
		}
		if cpuprofileFile != nil {
			pprof.StopCPUProfile()
			cpuprofileFile.Close()
			fmt.Printf("info: CPU profile saved to %s\n", config.CPUProfile)
		}
		if config.MemProfile != "" {
			f, err := os.Create(config.MemProfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to create memory profile: %v\n", err)
			} else {
				runtime.GC() // get up-to-date statistics
				if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
					fmt.Fprintf(os.Stderr, "error: failed to write memory profile: %v\n", err)
				} else {
					fmt.Printf("info: memory allocation profile saved to %s\n", config.MemProfile)
				}
				f.Close()
			}
		}
		if config.HeapProfile != "" {
			f, err := os.Create(config.HeapProfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: failed to create heap profile: %v\n", err)
			} else {
				runtime.GC() // get up-to-date statistics
				if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
					fmt.Fprintf(os.Stderr, "error: failed to write heap profile: %v\n", err)
				} else {
					fmt.Printf("info: heap profile saved to %s\n", config.HeapProfile)
				}
				f.Close()
			}
		}
	}()

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// loadEnvFile loads environment variables from .env file if it exists
func loadEnvFile() {
	path := dotenvPath
	if path == "" {
		path = ".env"
	}

	if err := godotenv.Load(path); err != nil {
		if dotenvPath != "" || !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: failed to load .env from %s: %v\n", path, err)
		}
	} else {
		fmt.Printf("info: loaded environment variables from %s\n", path)
	}
}

// applyEnvironmentVariables reads environment variables and applies them to the configuration
func applyEnvironmentVariables(cmd *cobra.Command) {
	root := cmd.Root()

	for _, e := range []struct{ env, flag string }{
		// Support both DITTO_APP_ID and DITTO_DATABASE_ID, with DATABASE_ID taking precedence
		{env: "DITTO_APP_ID", flag: "db-id"},
		{env: "DITTO_DATABASE_ID", flag: "db-id"},
		{env: "DITTO_PLAYGROUND_TOKEN", flag: "playground-token"},
		{env: "DITTO_SHARED_TOKEN", flag: "shared-token"},
		{env: "DITTO_SECRET_KEY", flag: "secret-key"},
		{env: "DITTO_AUTH_URL", flag: "auth-url"},
		{env: "DITTO_AUTH_EVENT_HANDLER", flag: "auth-event-handler"},
		{env: "DITTO_LICENSE", flag: "offline-only-license-token"},
	} {
		if val := os.Getenv(e.env); val != "" {
			if flag := root.PersistentFlags().Lookup(e.flag); flag != nil {
				flag.Value.Set(val)
			}
		}
	}

	if wsUrls := os.Getenv("DITTO_WEBSOCKET_URL"); wsUrls != "" {
		if flag := root.PersistentFlags().Lookup("websocket-urls"); flag != nil {
			config.WebsocketURLs = strings.Split(wsUrls, ",")
		}
	}
}
