// Copyright 2025 DittoLive Incorporated. All rights reserved.

package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/trace"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/getditto/ditto-go-sdk/v5/ditto"
	"github.com/mattn/go-runewidth"
	"github.com/pterm/pterm"
)

// termHighlightColor returns a string for highlighted text in the terminal
func termHighlightColor(a ...interface{}) string {
	return pterm.LightBlue(a...)
}

// DQLCommandProcessor represents the state of the DQL REPL
type DQLCommandProcessor struct {
	ctx           context.Context
	config        *Config
	toolkit       *Toolkit
	commands      map[string]*DQLCommand
	rl            *readline.Instance
	subscriptions []*ditto.SyncSubscription
}

// DQLCommand represents a special REPL command
type DQLCommand struct {
	Name        string
	Description string
	Handler     func(repl *DQLCommandProcessor, args string) error
}

// DoDQLInterpreter runs an interactive DQL REPL
func DoDQLInterpreter(ctx context.Context, config *Config) error {
	defer trace.StartRegion(ctx, "DoDQLInterpreter").End()

	if config.NoColor {
		pterm.DisableColor()
	}

	toolkit, err := NewToolkit(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create toolkit: %w", err)
	}
	defer toolkit.Close()

	if err := toolkit.StartSync(); err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}

	repl := &DQLCommandProcessor{
		ctx:      ctx,
		config:   config,
		toolkit:  toolkit,
		commands: make(map[string]*DQLCommand),
	}

	repl.registerDQLCommands()

	rl, err := readline.New(pterm.Sprintf("%s> ", termHighlightColor("DQL")))
	if err != nil {
		return fmt.Errorf("failed to initialize readline: %w", err)
	}
	repl.rl = rl
	defer rl.Close()

	if err := repl.displayWelcomeBanner(); err != nil {
		pterm.Warning.Printf("Failed to display banner: %v\n", err)
	}

	// Main REPL loop
	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// Ctrl+C - just show prompt again
				continue
			}
			if err == io.EOF {
				// Ctrl+D - exit gracefully
				break
			}
			return fmt.Errorf("readline error: %w", err)
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if strings.ToLower(input) == ".exit" {
			break
		}

		if err := repl.processInput(input); err != nil {
			pterm.Error.Println(err)
		}
	}

	// Clean up subscriptions before exiting
	if count := repl.unsubscribe(); count > 0 {
		pterm.Info.Printf("Canceled %d subscription(s)\n", count)
	}

	pterm.Info.Println("Goodbye!")
	return nil
}

// registerDQLCommands registers all special commands
func (repl *DQLCommandProcessor) registerDQLCommands() {
	repl.commands[".help"] = &DQLCommand{
		Name:        ".help",
		Description: "Show this help message",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdHelp(args) },
	}

	repl.commands[".system"] = &DQLCommand{
		Name:        ".system",
		Description: "Show system information (document counts, indexes)",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdSystem(args) },
	}

	repl.commands[".collections"] = &DQLCommand{
		Name:        ".collections",
		Description: "List all collections with document counts",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdCollections(args) },
	}

	repl.commands[".subscribe"] = &DQLCommand{
		Name:        ".subscribe",
		Description: "Subscribe to a DQL query for sync",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdSubscribe(args) },
	}

	repl.commands[".unsubscribe"] = &DQLCommand{
		Name:        ".unsubscribe",
		Description: "Cancel all active subscriptions",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdUnsubscribe(args) },
	}

	repl.commands[".export"] = &DQLCommand{
		Name:        ".export",
		Description: "Export query results to exports/export_<timestamp>.ndjson",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdExport(args) },
	}

	repl.commands[".bench"] = &DQLCommand{
		Name:        ".bench",
		Description: "Benchmark a query (20 runs)",
		Handler:     func(r *DQLCommandProcessor, args string) error { return r.cmdBench(args) },
	}

	repl.commands[".exit"] = &DQLCommand{
		Name:        ".exit",
		Description: "Exit the DQL terminal",
		Handler:     nil, // Handled specially in main loop
	}
}

// unsubscribe cancels all active subscriptions and returns the count
func (repl *DQLCommandProcessor) unsubscribe() int {
	count := len(repl.subscriptions)
	for _, sub := range repl.subscriptions {
		sub.Cancel()
	}
	repl.subscriptions = nil
	return count
}

// processInput processes a line of input from the REPL
func (repl *DQLCommandProcessor) processInput(input string) error {
	defer trace.StartRegion(repl.ctx, "processInput").End()

	// Check if it's a special command
	if strings.HasPrefix(input, ".") {
		cmdParts := strings.SplitN(input, " ", 2)
		cmdName := strings.ToLower(cmdParts[0])
		cmdArgs := ""
		if len(cmdParts) > 1 {
			cmdArgs = strings.TrimSpace(cmdParts[1])
		}

		cmd, ok := repl.commands[cmdName]
		if !ok {
			pterm.Error.Printf("Invalid command: %s\n", cmdName)
			pterm.Info.Printf("Type %s for available commands.\n", termHighlightColor(".help"))
			return nil
		}

		if cmd.Handler != nil {
			return cmd.Handler(repl, cmdArgs)
		}
		return nil
	}

	// Execute as DQL query
	return repl.executeQuery(input)
}

// executeQuery executes a DQL query and displays the results
func (repl *DQLCommandProcessor) executeQuery(query string) error {
	defer trace.StartRegion(repl.ctx, "executeQuery").End()

	start := time.Now()
	result, err := repl.toolkit.Execute(query, nil)
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer result.Close()

	pterm.Info.Printf("%s %dms\n",
		termHighlightColor("execute-time:"),
		elapsed.Milliseconds())

	upperQuery := strings.ToUpper(strings.TrimSpace(query))
	isExplain := strings.HasPrefix(upperQuery, "EXPLAIN")
	isProfile := strings.HasPrefix(upperQuery, "PROFILE")

	// Collect items into a slice for counting and display
	//
	// Note: items will not necessarily be of type ditto.Document,
	// because EXPLAIN and PROFILE and other DQL commands could have
	// different kinds of results.
	var items []interface{}
	for _, item := range result.Items() {
		// Parse JSON string back to interface{} for formatting
		var parsed interface{}
		if err := json.Unmarshal([]byte(item.JSONString()), &parsed); err == nil {
			items = append(items, parsed)
		} else {
			items = append(items, item.JSONString())
		}
	}

	if isExplain || isProfile {
		// Display formatted JSON for EXPLAIN/PROFILE
		for _, item := range items {
			formatted, err := json.MarshalIndent(item, "", "  ")
			if err != nil {
				pterm.Error.Printf("Failed to format result: %v\n", err)
				continue
			}
			fmt.Println(string(formatted))
		}
	} else {
		// Regular query - show count and ask if user wants to print results
		pterm.Info.Printf("Result count: %d\n", len(items))

		if len(items) > 0 {
			fmt.Print("Print results? (y/n, default: n): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "y" || response == "yes" {
				for i, item := range items {
					pterm.Printf("%s%d%s\n",
						termHighlightColor("["),
						i+1,
						termHighlightColor("]"))
					formatted, err := json.MarshalIndent(item, "", "  ")
					if err != nil {
						pterm.Error.Printf("Failed to format result: %v\n", err)
						continue
					}
					fmt.Println(string(formatted))
				}
			}
		}
	}

	return nil
}

// getCollectionCounts returns a map of collection names to document counts
func (repl *DQLCommandProcessor) getCollectionCounts() (map[string]int, error) {
	defer trace.StartRegion(repl.ctx, "getCollectionCounts").End()

	collections := make(map[string]int)
	collectionsQuery := "SELECT * FROM COLLECTION __collections"
	result, err := repl.toolkit.Execute(collectionsQuery, nil)
	if err != nil {
		return collections, err
	}
	defer result.Close()

	for _, item := range result.Items() {
		var collectionMap map[string]interface{}
		if err := json.Unmarshal([]byte(item.JSONString()), &collectionMap); err == nil {
			if name, ok := collectionMap["name"].(string); ok {
				// Count documents in this collection
				countQuery := fmt.Sprintf("SELECT COUNT(*) AS _count FROM %s", name)
				countResult, err := repl.toolkit.Execute(countQuery, nil)
				if err == nil {
					for _, countItem := range countResult.Items() {
						var countMap map[string]interface{}
						if err := json.Unmarshal([]byte(countItem.JSONString()), &countMap); err == nil {
							if count, ok := countMap["_count"].(float64); ok {
								collections[name] = int(count)
							}
						}
						break // Only need first item
					}
					countResult.Close()
				}
			}
		}
	}

	return collections, nil
}

// displayWelcomeBanner displays the welcome banner with collection counts
func (repl *DQLCommandProcessor) displayWelcomeBanner() error {
	defer trace.StartRegion(repl.ctx, "displayWelcomeBanner").End()

	// Get SDK version
	sdkVersion := GetSDKVersion()

	// Get collection document counts
	collections, _ := repl.getCollectionCounts()

	// Get working directory
	workingDir, _ := os.Getwd()

	// Build banner
	boxWidth := 80
	topBorder := "╭" + strings.Repeat("─", boxWidth-2) + "╮"
	bottomBorder := "╰" + strings.Repeat("─", boxWidth-2) + "╯"

	centerText := func(text string) string {
		// Remove ANSI codes and calculate display width (handles wide chars, emoji, etc.)
		cleanText := pterm.RemoveColorFromString(text)
		visibleLen := runewidth.StringWidth(cleanText)
		padding := (boxWidth - 4 - visibleLen) / 2
		rightPadding := boxWidth - 4 - visibleLen - padding
		return termHighlightColor("│") + " " + strings.Repeat(" ", padding) + text + strings.Repeat(" ", rightPadding) + " " + termHighlightColor("│")
	}

	pterm.Println(termHighlightColor(topBorder))
	pterm.Println(centerText(""))
	pterm.Println(centerText(termHighlightColor("Ditto DQL Sandbox")))
	pterm.Println(centerText(fmt.Sprintf("ditto-go-tools · Ditto SDK %s", sdkVersion)))
	pterm.Println(centerText(""))

	if len(collections) > 0 {
		pterm.Println(centerText(termHighlightColor("Collections")))
		for name, count := range collections {
			pterm.Println(centerText(fmt.Sprintf("%s (%d docs)", name, count)))
		}
		pterm.Println(centerText(""))
	}

	pterm.Println(centerText(termHighlightColor("Working Directory")))
	pterm.Println(centerText(workingDir))
	pterm.Println(centerText(""))
	pterm.Println(centerText(fmt.Sprintf("Type %s for available commands", termHighlightColor(".help"))))
	pterm.Println(centerText(""))
	pterm.Println(termHighlightColor(bottomBorder))
	pterm.Println("")

	return nil
}

// padCommand pads a command string to the specified visible width, accounting for ANSI color codes.
// Uses runewidth.StringWidth to properly handle Unicode characters, emoji, and wide characters (CJK).
func padCommand(cmd string, width int) string {
	colored := termHighlightColor(cmd)
	visibleLen := runewidth.StringWidth(pterm.RemoveColorFromString(colored))
	padding := width - visibleLen
	if padding < 0 {
		padding = 0
	}
	return colored + strings.Repeat(" ", padding)
}

// cmdHelp displays help information
func (repl *DQLCommandProcessor) cmdHelp(args string) error {
	defer trace.StartRegion(repl.ctx, "cmdHelp").End()

	const cmdWidth = 30

	pterm.Println("\nGeneral commands:")
	pterm.Printf("  %s - Show this help message\n", padCommand(".help", cmdWidth))
	pterm.Printf("  %s - Show system information (document counts, indexes)\n", padCommand(".system", cmdWidth))
	pterm.Printf("  %s - List all collections with document counts\n", padCommand(".collections", cmdWidth))
	pterm.Printf("  %s - Exit the DQL terminal\n", padCommand(".exit", cmdWidth))

	pterm.Println("\nSync commands:")
	pterm.Printf("  %s - Subscribe to a query for sync\n", padCommand(".subscribe <query>", cmdWidth))
	pterm.Printf("  %s - Cancel all active subscriptions\n", padCommand(".unsubscribe", cmdWidth))

	pterm.Println("\nBenchmark commands:")
	pterm.Printf("  %s - Benchmark a query (20 runs)\n", padCommand(".bench <query>", cmdWidth))

	pterm.Println("\nUtility commands:")
	pterm.Printf("  %s - Export query results to exports/export_<timestamp>.ndjson\n", padCommand(".export <query>", cmdWidth))

	pterm.Println("\nDQL queries:")
	pterm.Println("  - Enter any valid DQL query to execute")
	pterm.Println("  - Queries starting with EXPLAIN will show execution plan")

	pterm.Println("\nExample queries:")
	pterm.Println("  SELECT * FROM movies LIMIT 10")
	pterm.Println("  SELECT title FROM movies WHERE year > 2020")
	pterm.Println("  EXPLAIN SELECT * FROM movies WHERE genre = \"Action\"")
	pterm.Println("")

	return nil
}

// cmdSystem displays system information
func (repl *DQLCommandProcessor) cmdSystem(args string) error {
	defer trace.StartRegion(repl.ctx, "cmdSystem").End()

	pterm.Println("")

	// 1. Ditto SDK Version
	pterm.DefaultSection.Println("Ditto SDK Version")
	sdkVersion := GetSDKVersion()
	pterm.Printf("  Version: %s\n", sdkVersion)
	pterm.Println("")

	// 2. System Information
	pterm.DefaultSection.Println("System Information")
	hostname, _ := os.Hostname()
	execPath, _ := os.Executable()
	pterm.Printf("  Platform: %s\n", runtime.GOOS)
	pterm.Printf("  Architecture: %s\n", runtime.GOARCH)
	pterm.Printf("  Hostname: %s\n", hostname)
	pterm.Printf("  Executable: %s\n", execPath)
	pterm.Println("")

	// 3. CPU Information
	pterm.DefaultSection.Println("CPU Information")
	pterm.Printf("  CPUs: %d\n", runtime.NumCPU())
	pterm.Println("")

	// 4. Memory Information
	pterm.DefaultSection.Println("Memory Information")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// General statistics
	pterm.Println("  General:")
	pterm.Printf("    Alloc: %d MB\n", m.Alloc/1024/1024)
	pterm.Printf("    TotalAlloc: %d MB\n", m.TotalAlloc/1024/1024)
	pterm.Printf("    Sys: %d MB\n", m.Sys/1024/1024)
	pterm.Printf("    Lookups: %d\n", m.Lookups)
	pterm.Printf("    Mallocs: %d\n", m.Mallocs)
	pterm.Printf("    Frees: %d\n", m.Frees)

	// Heap statistics
	pterm.Println("  Heap:")
	pterm.Printf("    HeapAlloc: %d MB\n", m.HeapAlloc/1024/1024)
	pterm.Printf("    HeapSys: %d MB\n", m.HeapSys/1024/1024)
	pterm.Printf("    HeapIdle: %d MB\n", m.HeapIdle/1024/1024)
	pterm.Printf("    HeapInuse: %d MB\n", m.HeapInuse/1024/1024)
	pterm.Printf("    HeapReleased: %d MB\n", m.HeapReleased/1024/1024)
	pterm.Printf("    HeapObjects: %d\n", m.HeapObjects)

	// Stack and allocator statistics
	pterm.Println("  Stack and Allocator:")
	pterm.Printf("    StackInuse: %d MB\n", m.StackInuse/1024/1024)
	pterm.Printf("    StackSys: %d MB\n", m.StackSys/1024/1024)
	pterm.Printf("    MSpanInuse: %d MB\n", m.MSpanInuse/1024/1024)
	pterm.Printf("    MSpanSys: %d MB\n", m.MSpanSys/1024/1024)
	pterm.Printf("    MCacheInuse: %d MB\n", m.MCacheInuse/1024/1024)
	pterm.Printf("    MCacheSys: %d MB\n", m.MCacheSys/1024/1024)
	pterm.Printf("    BuckHashSys: %d MB\n", m.BuckHashSys/1024/1024)
	pterm.Printf("    GCSys: %d MB\n", m.GCSys/1024/1024)
	pterm.Printf("    OtherSys: %d MB\n", m.OtherSys/1024/1024)

	// Garbage collector statistics
	pterm.Println("  Garbage Collector:")
	pterm.Printf("    NextGC: %d MB\n", m.NextGC/1024/1024)
	pterm.Printf("    LastGC: %s\n", time.Unix(0, int64(m.LastGC)).Format(time.RFC3339))
	pterm.Printf("    PauseTotalNs: %d ms\n", m.PauseTotalNs/1000000)
	pterm.Printf("    NumGC: %d\n", m.NumGC)
	pterm.Printf("    NumForcedGC: %d\n", m.NumForcedGC)
	pterm.Printf("    GCCPUFraction: %.6f\n", m.GCCPUFraction)
	pterm.Printf("    EnableGC: %t\n", m.EnableGC)
	pterm.Printf("    DebugGC: %t\n", m.DebugGC)
	pterm.Println("")

	// 5. Go Runtime Information
	pterm.DefaultSection.Println("Go Runtime Information")
	pterm.Printf("  Version: %s\n", runtime.Version())
	pterm.Printf("  Goroutines: %d\n", runtime.NumGoroutine())
	pterm.Printf("  NumCgoCall: %d\n", runtime.NumCgoCall())
	pterm.Println("")

	// 6. Storage Information
	pterm.DefaultSection.Println("Storage Information")
	workingDir, _ := os.Getwd()
	pterm.Printf("  Working Directory: %s\n", workingDir)
	if repl.config.PersistentRoot != "" {
		pterm.Printf("  Ditto Directory: %s\n", repl.config.PersistentRoot)
	}
	pterm.Println("")

	pterm.Println("")
	return nil
}

// cmdCollections displays all collections with document counts
func (repl *DQLCommandProcessor) cmdCollections(_ string) error {
	defer trace.StartRegion(repl.ctx, "cmdCollections").End()

	pterm.Println("")
	pterm.DefaultSection.Println("Collections")

	collections, err := repl.getCollectionCounts()
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}

	if len(collections) == 0 {
		pterm.Info.Println("  No collections found")
	} else {
		for name, count := range collections {
			pterm.Printf("  %s: %d documents\n", termHighlightColor(name), count)
		}
	}

	pterm.Println("")
	return nil
}

// cmdSubscribe registers a sync subscription for a DQL query
func (repl *DQLCommandProcessor) cmdSubscribe(args string) error {
	defer trace.StartRegion(repl.ctx, "cmdSubscribe").End()

	if args == "" {
		return fmt.Errorf("usage: .subscribe <query>")
	}

	// Register the subscription
	subscription, err := repl.toolkit.RegisterSubscription(args, nil)
	if err != nil {
		return fmt.Errorf("failed to register subscription: %w", err)
	}

	// Add to our list of subscriptions
	repl.subscriptions = append(repl.subscriptions, subscription)

	pterm.Success.Printf("✓ Subscription registered (total: %d)\n", len(repl.subscriptions))
	pterm.Info.Printf("  Query: %s\n", args)
	return nil
}

// cmdUnsubscribe cancels all active subscriptions
func (repl *DQLCommandProcessor) cmdUnsubscribe(_ string) error {
	defer trace.StartRegion(repl.ctx, "cmdUnsubscribe").End()

	count := repl.unsubscribe()
	if count == 0 {
		pterm.Info.Println("No active subscriptions")
		return nil
	}

	pterm.Success.Printf("✓ Canceled %d subscription(s)\n", count)
	return nil
}

// cmdExport exports query results to an NDJSON file
func (repl *DQLCommandProcessor) cmdExport(args string) error {
	defer trace.StartRegion(repl.ctx, "cmdExport").End()

	if args == "" {
		return fmt.Errorf("usage: .export <query>")
	}

	// Execute the query
	result, err := repl.toolkit.Execute(args, nil)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}
	defer result.Close()

	// Collect items
	var items []string
	for _, item := range result.Items() {
		items = append(items, item.JSONString())
	}

	if len(items) == 0 {
		pterm.Warning.Println("No results to export")
		return nil
	}

	// Create exports directory
	exportsDir := "exports"
	if err := os.MkdirAll(exportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create exports directory: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("export_%s.ndjson", timestamp)
	filepath := fmt.Sprintf("%s/%s", exportsDir, filename)

	// Write NDJSON file
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	for _, jsonStr := range items {
		file.Write([]byte(jsonStr))
		file.Write([]byte("\n"))
	}

	// Get file size
	fileInfo, _ := file.Stat()
	fileSizeKB := fileInfo.Size() / 1024

	// Display summary
	pterm.Println("")
	pterm.Success.Printf("📋 Export complete\n")
	pterm.Printf("   Documents exported: %d\n", len(items))
	pterm.Printf("   File: %s\n", filepath)
	pterm.Printf("   Size: %d KB\n", fileSizeKB)
	pterm.Printf("   Query: %s\n", args)
	pterm.Println("")

	return nil
}

// cmdBench benchmarks a query
func (repl *DQLCommandProcessor) cmdBench(args string) error {
	defer trace.StartRegion(repl.ctx, "cmdBench").End()

	if args == "" {
		return fmt.Errorf("usage: .bench <query>")
	}

	// Remove surrounding quotes if present
	query := strings.Trim(args, "\"'")

	const runs = 20
	pterm.Info.Printf("Benchmarking query: %s\n", query)
	pterm.Info.Printf("Runs: %d\n\n", runs)

	// Create progress bar
	progressbar, _ := pterm.DefaultProgressbar.WithTotal(runs).WithTitle("Running benchmark").Start()

	// Run benchmark
	times := make([]time.Duration, runs)
	var resultCount int

	for i := 0; i < runs; i++ {
		start := time.Now()
		result, err := repl.toolkit.Execute(query, nil)
		elapsed := time.Since(start)

		if err != nil {
			progressbar.Stop()
			return fmt.Errorf("query error on run %d: %w", i+1, err)
		}

		times[i] = elapsed
		if i == 0 {
			// Count items on first run
			count := 0
			for range result.Items() {
				count++
			}
			resultCount = count
		}
		result.Close()

		progressbar.Increment()
	}

	progressbar.Stop()

	// Calculate statistics
	stats := calculateBenchmarkStats(times)

	// Display results
	pterm.Println("")
	pterm.DefaultSection.Println("Benchmark Results")
	pterm.Printf("  Result count: %d\n", resultCount)
	pterm.Printf("  Total runs: %d\n", runs)
	pterm.Println("")

	pterm.Println("  Timing Statistics:")
	pterm.Printf("    Mean:   %.2f ms\n", stats.Mean)
	pterm.Printf("    Median: %.2f ms\n", stats.Median)
	pterm.Printf("    Min:    %.2f ms\n", stats.Min)
	pterm.Printf("    Max:    %.2f ms\n", stats.Max)
	pterm.Printf("    StdDev: %.2f ms\n", stats.StdDev)
	pterm.Printf("    95%%:    %.2f ms\n", stats.P95)
	pterm.Printf("    99%%:    %.2f ms\n", stats.P99)
	pterm.Println("")

	totalTime := float64(0)
	for _, t := range times {
		totalTime += t.Seconds()
	}
	queriesPerSec := float64(runs) / totalTime

	pterm.Println("  Throughput:")
	pterm.Printf("    Queries/sec: %.2f\n", queriesPerSec)
	pterm.Printf("    Total time:  %.2f s\n", totalTime)
	pterm.Println("")

	return nil
}

// BenchmarkStats holds benchmark statistics
type BenchmarkStats struct {
	Mean   float64
	Median float64
	Min    float64
	Max    float64
	StdDev float64
	P95    float64
	P99    float64
}

// calculateBenchmarkStats calculates statistics from benchmark times
func calculateBenchmarkStats(times []time.Duration) BenchmarkStats {
	if len(times) == 0 {
		return BenchmarkStats{}
	}

	// Convert to milliseconds
	ms := make([]float64, len(times))
	for i, t := range times {
		ms[i] = float64(t.Milliseconds())
	}

	// Sort for median and percentiles
	sorted := make([]float64, len(ms))
	copy(sorted, ms)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate mean
	sum := 0.0
	min := sorted[0]
	max := sorted[len(sorted)-1]
	for _, v := range ms {
		sum += v
	}
	mean := sum / float64(len(ms))

	// Calculate median
	median := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}

	// Calculate standard deviation
	variance := 0.0
	for _, v := range ms {
		variance += (v - mean) * (v - mean)
	}
	stdDev := 0.0
	if len(ms) > 1 {
		stdDev = variance / float64(len(ms)-1)
		if stdDev > 0 {
			stdDev = 1.0 / stdDev // Simplified sqrt approximation
			stdDev = (stdDev + variance/stdDev) / 2.0
		}
	}

	// Calculate percentiles
	p95Idx := int(float64(len(sorted)) * 0.95)
	p99Idx := int(float64(len(sorted)) * 0.99)
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}

	return BenchmarkStats{
		Mean:   mean,
		Median: median,
		Min:    min,
		Max:    max,
		StdDev: stdDev,
		P95:    sorted[p95Idx],
		P99:    sorted[p99Idx],
	}
}
