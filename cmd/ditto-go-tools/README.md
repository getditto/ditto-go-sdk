# ditto-go-tools

Command-line utility for testing and interacting with the Ditto Go SDK. This tool provides a comprehensive set of commands for database operations, real-time data synchronization, and SDK testing.

## Overview

`ditto-go-tools` is a test and development utility that provides command-line access to Ditto SDK functionality. It's primarily used for:

- Testing SDK installations and connectivity
- Performing CRUD operations on Ditto collections
- Observing real-time data changes and peer presence
- Executing DQL (Ditto Query Language) queries
- Exporting diagnostic logs
- Validating SDK configurations
- Performance profiling with runtime traces and CPU/memory/heap profiling

## Building

### Prerequisites

- Go 1.24 or later
- Make (optional, for using Makefile targets)
- Ditto Go SDK dependencies

### Build Instructions

From this directory (`cmd/ditto-go-tools/`):

```bash
# Build using Make
make build

# Or build directly with Go
go build -o ditto-go-tools .
```

From the SDK root directory (`sdks/go/`):

```bash
# Build the entire SDK including ditto-go-tools
make build-all

# Or build just the tools
go build -o ditto-go-tools ./cmd/ditto-go-tools
```

The binary will be created as `ditto-go-tools` in the current directory.

## Installation

### Install Dependencies

```bash
# From this directory
go mod download
```

### Environment Setup

Set up your environment variables for authentication:

```bash
# For Playground authentication
export DITTO_DATABASE_ID="your-database-id"  # or DITTO_APP_ID (deprecated)
export DITTO_PLAYGROUND_TOKEN="your-playground-token"
export DITTO_AUTH_URL="https://cloud.ditto.live"  # Required for playground

# For Shared Secret authentication
export DITTO_SHARED_TOKEN="your-shared-token"
export DITTO_SECRET_KEY="your-secret-key"

# For custom authentication
export DITTO_AUTH_URL="https://your-auth-server.com"
export DITTO_AUTH_EVENT_HANDLER="path/to/handler"

# For offline-only license
export DITTO_LICENSE="your-license-token"

# For WebSocket connections
export DITTO_WEBSOCKET_URL="wss://your-server.com,wss://backup-server.com"
```

## Usage

### Basic Command Structure

```bash
ditto-go-tools [global-options] <command> [command-options]
```

### Global Options

| Option                 | Short | Description                                        | Default            |
|------------------------|-------|----------------------------------------------------|--------------------|
| `--db-id`              | -     | Ditto database ID                                  | -                  |
| `--app-id`             | -     | Ditto application ID (deprecated, use --db-id)     | -                  |
| `--auth-system`        | -     | Authentication system (see Authentication section) | `small-peers-only` |
| `--config`             | `-c`  | Path to TOML configuration file                    | -                  |
| `--log-level`          | `-l`  | Log level (error, warning, info, debug)            | `error`            |
| `--log-file`           | -     | Path to Ditto log file                             | -                  |
| `--playground`         | -     | Use playground mode (shortcut)                     | `false`            |
| `--playground-token`   | `-t`  | Playground authentication token                    | -                  |
| `--shared-token`       | `-s`  | Shared authentication token                        | -                  |
| `--secret-key`         | -     | Secret key for authentication                      | -                  |
| `--offline-only-license-token` | - | Offline-only license token                   | -                  |
| `--persistent-root`    | -     | Directory for persistent storage                   | Auto-generated     |
| `--auth-url`           | -     | Authentication server URL                          | -                  |
| `--websocket-urls`     | -     | Comma-separated WebSocket URLs                     | -                  |
| `--auth-event-handler` | -     | Path to authentication event handler               | -                  |
| `--enable-ble`         | -     | Enable Bluetooth LE transport                      | `true`             |
| `--enable-lan`         | -     | Enable LAN transport                               | `true`             |
| `--tcp-listener`       | -     | Enable TCP listener                                | `false`            |
| `--tcp-listener-port`  | -     | TCP listener port number                           | `0`                |
| `--name`               | `-n`  | Set device name                                    | `ditto-go-tools`   |
| `--no-dotenv`          | -     | Do not load .env file                              | `false`            |
| `--dotenv-path`        | -     | Path to .env file                                  | `./.env`           |
| `--no-envvars`         | -     | Do not read environment variables                  | `false`            |
| `--trace`              | -     | Enable runtime trace output to specified file      | -                  |
| `--cpuprofile`         | -     | Enable CPU profiling output to specified file      | -                  |
| `--memprofile`         | -     | Write memory profile to specified file at exit     | -                  |
| `--heapprofile`        | -     | Write heap profile to specified file at exit       | -                  |

## Commands

### Information Commands

#### sdk-version
Display the Ditto SDK version.

```bash
ditto-go-tools sdk-version
# Output: SDK version: 5.0.0-dev
```

#### print-config
Display the current configuration settings.

```bash
ditto-go-tools --db-id test123 print-config
```

#### smoke-test
Verify that Ditto can initialize and establish sync connections.

```bash
ditto-go-tools --db-id test123 smoke-test
```

### Query Commands

#### execute
Execute a synchronous DQL query and display results.

```bash
# Basic query
ditto-go-tools execute -q "SELECT * FROM cars"

# Query with parameters
ditto-go-tools execute -q "SELECT * FROM cars WHERE year > :year" \
  -d '{"year": 2020}'

# Different output formats
ditto-go-tools execute -q "SELECT * FROM cars" -f json
ditto-go-tools execute -q "SELECT * FROM cars" -f table
ditto-go-tools execute -q "SELECT * FROM cars" -f csv

# Note: Attachment and metadata support are not yet implemented
```

Options:
- `-q, --query`: DQL query string (required)
- `-d, --data`: JSON parameters for the query
- `-f, --format`: Output format (json, table, csv) [default: json]
- `--attachment`: Path to attachment file (not yet implemented)
- `-m, --metadata`: JSON metadata (not yet implemented)

### Data Manipulation Commands

All CRUD operations are fully implemented and working.

#### insert
Insert one or more documents into a collection.

```bash
# Insert single document
ditto-go-tools insert --collection cars -d '{"_id": "car1", "make": "Toyota", "year": 2022}'

# Insert multiple documents (batch mode)
ditto-go-tools insert --collection cars -b -d '[
  {"_id": "car1", "make": "Toyota", "year": 2022},
  {"_id": "car2", "make": "Honda", "year": 2023}
]'

# Note: Attachment support is not yet implemented
```

Options:
- `--collection`: Collection name (required)
- `-d, --data`: JSON document data (required)
- `-b, --batch`: Insert multiple documents from JSON array
- `--attachment`: Path to attachment file (not yet implemented)

#### update
Update existing documents in a collection.

```bash
# Update by ID
ditto-go-tools update --collection cars -i car1 -d '{"year": 2023}'

# Update with query condition
ditto-go-tools update --collection cars -q "make == 'Toyota'" -d '{"serviced": true}'
```

Options:
- `--collection`: Collection name (required)
- `-i, --id`: Document ID to update
- `-q, --query`: Query condition for update
- `-d, --data`: JSON update data (required)

#### delete
Delete documents from a collection.

```bash
# Delete by ID
ditto-go-tools delete --collection cars -i car1

# Delete with query condition
ditto-go-tools delete --collection cars -q "year < 2000"
```

Options:
- `--collection`: Collection name (required)
- `-i, --id`: Document ID to delete
- `-q, --query`: Query condition for deletion

#### evict
Evict documents from local storage (keeps them in remote sync).

```bash
# Evict by ID
ditto-go-tools evict --collection cars -i car1

# Evict with query condition
ditto-go-tools evict --collection cars -q "archived == true"
```

Options:
- `--collection`: Collection name (required)
- `-i, --id`: Document ID to evict
- `-q, --query`: Query condition for eviction

#### count
Count documents in a collection.

```bash
# Count all documents
ditto-go-tools count --collection cars

# Count with query condition
ditto-go-tools count --collection cars -q "year > 2020"
```

Options:
- `--collection`: Collection name (required)
- `-q, --query`: Query condition for counting

#### find-by-id
Find and display a single document by its ID.

```bash
# Find document
ditto-go-tools find-by-id --collection cars -i car1

# Pretty-print output
ditto-go-tools find-by-id --collection cars -i car1 -f pretty
```

Options:
- `--collection`: Collection name (required)
- `-i, --id`: Document ID (required)
- `-f, --format`: Output format (json, pretty) [default: json]

### Observation Commands

#### observe
Observe real-time changes to query results.

```bash
# Observe single query
ditto-go-tools observe -q "SELECT * FROM cars WHERE year > 2020"

# Observe multiple queries
ditto-go-tools observe \
  -q "SELECT * FROM cars" \
  -q "SELECT * FROM trucks"

# With timeout (in seconds)
ditto-go-tools observe -q "SELECT * FROM cars" -t 60

# Note: Save attachments feature is not yet implemented
```

Options:
- `-q, --query`: DQL query (required, can specify multiple)
- `-t, --timeout`: Timeout in seconds (0 for no timeout)
- `-s, --save`: Save attachments to files (not yet implemented)

#### presence
Observe local and/or remote peer metadata.

```bash
# Observe remote peers (default)
ditto-go-tools presence

# Observe all peers (local and remote)
ditto-go-tools presence -p all

# Observe only local peer
ditto-go-tools presence -p local

# With timeout (in seconds)
ditto-go-tools presence --timeout 30
```

Options:
- `-p, --peer-scope`: Peer scope (remote, local, all) [default: remote]
- `--timeout`: Timeout in seconds (0 for no timeout)

### Diagnostic Commands

#### export-logs
Export diagnostic logs as a compressed JSONL archive.

```bash
# Export logs
ditto-go-tools export-logs -o logs.jsonl.gz

# Force overwrite existing file
ditto-go-tools export-logs -o logs.jsonl.gz -f
```

Options:
- `-o, --output`: Output file path (required)
- `-f, --force-overwrite`: Overwrite existing file

## Configuration

### Configuration Precedence

The tool loads configuration from multiple sources with the following precedence (highest to lowest):

1. **Command-line flags** - Highest precedence, always wins
2. **Environment variables** - Applied unless `--no-envvars` is specified
3. **.env file** - Loaded from current directory unless `--no-dotenv` is specified (or custom path via `--dotenv-path`)
4. **TOML configuration file** - If specified via `--config` flag
5. **Default values** - Built-in defaults

Example:
```bash
# Disable .env file loading
./ditto-go-tools --no-dotenv <command>

# Use a specific .env file from a different location
./ditto-go-tools --dotenv-path /path/to/my/.env <command>

# Disable environment variable reading
./ditto-go-tools --no-envvars <command>

# Use only CLI flags and defaults
./ditto-go-tools --no-dotenv --no-envvars --db-id "test-db" <command>
```

### Configuration File (TOML)

You can use a TOML configuration file to set default values:

```toml
# config.toml
database_id = "your-database-id"
auth_system = "online-playground"
playground_token = "your-token"
log_level = "info"
log_file = "/var/log/ditto.log"
persistent_root = "/var/ditto/data"
enable_ble = true
enable_lan = true
tcp_listener = false
name = "my-device"
```

Load the configuration:

```bash
ditto-go-tools --config config.toml <command>
```

### Authentication Methods

The tool supports multiple authentication systems:

#### 1. Small Peers Only (default)
For peer-to-peer only mode without cloud sync.

```bash
ditto-go-tools --auth-system small-peers-only \
  --db-id "your-database-id" \
  <command>
```

#### 2. Online Playground
For development and testing with Ditto's playground environment.

```bash
ditto-go-tools --auth-system online-playground \
  --db-id "your-database-id" \
  --playground-token "your-token" \
  --auth-url "https://cloud.ditto.live" \
  <command>

# Or use the shortcut flag
ditto-go-tools --playground \
  --db-id "your-database-id" \
  --playground-token "your-token" \
  --auth-url "https://cloud.ditto.live" \
  <command>
```

#### 3. Shared Secret
For production deployments with shared key authentication.

```bash
ditto-go-tools --auth-system shared-secret \
  --db-id "your-database-id" \
  --shared-token "your-token" \
  --secret-key "your-secret" \
  <command>
```

#### 4. Online with Authentication
For custom authentication servers.

```bash
ditto-go-tools --auth-system online-with-authentication \
  --auth-url "https://auth.example.com" \
  <command>
```

## Examples

### Example 1: Basic Data Operations

```bash
# Set up authentication (use environment variables or flags)
export DITTO_DATABASE_ID="test-db"

# Initialize and test connection
ditto-go-tools smoke-test

# Insert some data
ditto-go-tools insert --collection products -d '{"_id": "p1", "name": "Widget", "price": 9.99}'

# Query the data
ditto-go-tools execute -q "SELECT * FROM products" -f table

# Update the price
ditto-go-tools update --collection products -i p1 -d '{"price": 12.99}'

# Count items
ditto-go-tools count --collection products
```

### Example 2: Real-time Observation

```bash
# In terminal 1: Start observing changes
ditto-go-tools observe -q "SELECT * FROM sensors WHERE active == true"

# In terminal 2: Insert data that triggers the observer
ditto-go-tools insert --collection sensors -d '{"_id": "s1", "active": true, "temp": 22.5}'
```

### Example 3: Peer Presence Monitoring

```bash
# Monitor peer connections for 60 seconds
ditto-go-tools presence -p all --timeout 60
```

### Example 4: Complex Query with Parameters

```bash
# Query with multiple conditions and parameters
ditto-go-tools execute \
  -q "SELECT * FROM orders WHERE status == :status AND total > :minTotal ORDER BY created_at DESC LIMIT :limit" \
  -d '{"status": "pending", "minTotal": 100, "limit": 10}' \
  -f table
```

## Output Formats

### JSON Format (default)
Outputs raw JSON for programmatic processing:
```json
{
  "items": [
    {"_id": "1", "name": "Item 1", "value": 100}
  ]
}
```

### Table Format
Human-readable table output with formatted columns.

### CSV Format
Comma-separated values for spreadsheet import:
```csv
_id,name,value
1,Item 1,100
```

## Debugging and Performance Analysis

### Runtime Trace

The `--trace` flag enables Go runtime tracing, which captures detailed execution information for performance analysis and debugging.

#### Basic Usage

```bash
# Enable trace output to a file
ditto-go-tools --trace trace.out <command>

# Example: Trace a query execution
ditto-go-tools --trace query_trace.out execute -q "SELECT * FROM products"
```

#### Analyzing Trace Files

Use the `go tool trace` command to analyze trace files:

```bash
# Open trace in web browser for interactive analysis
go tool trace trace.out
```

This opens a web interface (typically at http://127.0.0.1:port) with several views:

1. **View trace** - Timeline of goroutine execution
2. **Goroutine analysis** - Statistics about goroutine behavior
3. **Network blocking profile** - Network-related blocking
4. **Synchronization blocking profile** - Mutex/channel blocking
5. **Syscall blocking profile** - System call blocking
6. **Scheduler latency profile** - Scheduling delays

#### Advanced Trace Analysis

```bash
# Trace a long-running observe command
ditto-go-tools --trace observe_trace.out observe -q "SELECT * FROM collection" -t 60

# Trace presence monitoring with all peers
ditto-go-tools --trace presence_trace.out presence -p all --timeout 30

# Trace batch insert operations
ditto-go-tools --trace batch_trace.out insert --collection data -b \
  -d '[{"_id":"1"},{"_id":"2"},{"_id":"3"}]'
```

#### What Trace Can Help Diagnose

- **Performance bottlenecks**: Identify slow functions and operations
- **Goroutine leaks**: Find goroutines that aren't properly terminated
- **Concurrency issues**: Analyze parallel execution and synchronization
- **Memory allocation patterns**: See where memory is being allocated
- **CPU utilization**: Understand CPU usage patterns
- **Network delays**: Identify network-related performance issues
- **Blocking operations**: Find where the program is waiting

#### Tips for Using Trace

1. **Keep trace files small**: Long-running traces can generate large files
2. **Use timeouts**: For observe/presence commands, set a reasonable timeout
3. **Compare traces**: Trace the same operation under different conditions
4. **Focus on specific operations**: Trace individual commands rather than entire sessions

Example workflow:
```bash
# 1. Capture trace for slow operation
ditto-go-tools --trace slow_query.out execute -q "SELECT * FROM large_collection"

# 2. Analyze in browser
go tool trace slow_query.out

# 3. Look for:
#    - Long-running goroutines
#    - High CPU usage areas
#    - Blocking system calls
#    - Memory allocation hotspots
```

### CPU Profiling

The `--cpuprofile` flag enables CPU profiling, which captures where the program spends its execution time. This is useful for identifying performance bottlenecks and hot code paths.

#### Basic Usage

```bash
# Enable CPU profiling
ditto-go-tools --cpuprofile cpu.prof <command>

# Example: Profile a query execution
ditto-go-tools --db-id test --cpuprofile cpu.prof execute -q "SELECT * FROM products"
```

#### Analyzing CPU Profiles

Use `go tool pprof` to analyze CPU profiles:

```bash
# Interactive analysis (command-line)
go tool pprof cpu.prof

# Common pprof commands:
# - top: Show functions consuming most CPU time
# - list <function>: Show annotated source code for a function
# - web: Generate graphical call graph (requires graphviz)
# - pdf: Generate PDF call graph

# Show top CPU consumers
go tool pprof -top cpu.prof

# Show cumulative time (functions + their callees)
go tool pprof -top -cum cpu.prof

# Generate web-based UI
go tool pprof -http=:8080 cpu.prof
```

#### What CPU Profiling Can Help With

- **Hot code paths**: Find functions consuming the most CPU time
- **Algorithm optimization**: Identify inefficient algorithms
- **Loop optimization**: Discover expensive loops
- **Performance regression**: Compare profiles before/after changes

#### Example Workflow

```bash
# 1. Capture CPU profile for an operation
ditto-go-tools --db-id test --cpuprofile slow_operation.prof \
  execute -q "SELECT * FROM large_collection WHERE complex_condition"

# 2. Analyze top CPU consumers
go tool pprof -top -cum slow_operation.prof

# 3. Look at specific hot function
go tool pprof -list "FunctionName" slow_operation.prof

# 4. Generate visual call graph
go tool pprof -web slow_operation.prof
```

### Memory Profiling

The `--memprofile` flag writes a memory allocation profile at program exit. This shows all memory allocations made during program execution, useful for finding allocation hotspots and memory inefficiencies.

#### Basic Usage

```bash
# Enable memory allocation profiling
ditto-go-tools --memprofile mem.prof <command>

# Example: Profile memory allocations during batch insert
ditto-go-tools --db-id test --memprofile mem.prof insert --collection data -b \
  -d '[{"_id":"1","data":"..."},{"_id":"2","data":"..."}]'
```

#### Analyzing Memory Profiles

```bash
# Show top memory allocators
go tool pprof -top mem.prof

# Show allocations with cumulative counts
go tool pprof -top -cum mem.prof

# Show allocation count vs allocation size
go tool pprof -alloc_space mem.prof  # Total bytes allocated
go tool pprof -alloc_objects mem.prof  # Total number of allocations

# Web-based UI
go tool pprof -http=:8080 mem.prof
```

#### What Memory Profiling Can Help With

- **Allocation hotspots**: Find code allocating excessive memory
- **Memory waste**: Identify unnecessary allocations
- **GC pressure**: Discover sources of garbage collector pressure
- **Memory optimization**: Guide memory usage improvements

### Heap Profiling

The `--heapprofile` flag writes a heap profile at program exit. Unlike memory profiling (which shows all allocations), heap profiling shows **live objects** still in memory, useful for finding memory leaks and understanding memory usage patterns.

#### Basic Usage

```bash
# Enable heap profiling
ditto-go-tools --heapprofile heap.prof <command>

# Example: Profile heap usage during long-running observation
ditto-go-tools --db-id test --heapprofile heap.prof \
  observe -q "SELECT * FROM collection" -t 60
```

#### Analyzing Heap Profiles

```bash
# Show live objects in memory
go tool pprof -top heap.prof

# Show by allocation count
go tool pprof -alloc_objects heap.prof

# Show by allocated space
go tool pprof -alloc_space heap.prof

# Show objects still in use (inuse)
go tool pprof -inuse_space heap.prof
go tool pprof -inuse_objects heap.prof

# Web-based UI
go tool pprof -http=:8080 heap.prof
```

#### What Heap Profiling Can Help With

- **Memory leaks**: Find objects not being freed
- **Memory retention**: Identify objects held longer than necessary
- **Memory footprint**: Understand overall memory usage
- **Object lifecycle**: See what objects remain in memory

### Combining Profiling Options

You can use multiple profiling options together to get a complete performance picture:

```bash
# Profile CPU, memory allocations, and heap together
ditto-go-tools --db-id test \
  --cpuprofile cpu.prof \
  --memprofile mem.prof \
  --heapprofile heap.prof \
  execute -q "SELECT * FROM large_dataset"

# With runtime trace as well
ditto-go-tools --db-id test \
  --trace trace.out \
  --cpuprofile cpu.prof \
  --memprofile mem.prof \
  --heapprofile heap.prof \
  observe -q "SELECT * FROM sensors" -t 30
```

### Profiling Tips

1. **Focus on realistic workloads**: Profile operations representative of actual usage
2. **Compare profiles**: Take before/after profiles when optimizing
3. **Look at cumulative metrics**: Use `-cum` flag to see time including callees
4. **Check sample counts**: Low sample counts may not be statistically significant
5. **Profile in production-like environments**: Development machines may behave differently
6. **Combine with trace**: Use `--trace` with profiling for complete analysis

### Profiling Example: Investigating Slow Query

```bash
# 1. Capture all profiling data
ditto-go-tools --db-id myapp \
  --cpuprofile cpu.prof \
  --memprofile mem.prof \
  --heapprofile heap.prof \
  --trace trace.out \
  execute -q "SELECT * FROM orders WHERE created_at > :date" \
  -d '{"date": "2024-01-01"}'

# 2. Check CPU usage
go tool pprof -top -cum cpu.prof

# 3. Check memory allocations
go tool pprof -top mem.prof

# 4. Check live heap objects
go tool pprof -top heap.prof

# 5. Check execution timeline
go tool trace trace.out

# 6. Generate comparison report
# (After making optimizations, repeat and compare)
go tool pprof -base cpu_before.prof cpu_after.prof
```

### Debug Logging

Enable debug logging for detailed operation information:

```bash
# Enable debug logs to console
ditto-go-tools --log-level debug <command>

# Write debug logs to file
ditto-go-tools --log-level debug --log-file debug.log <command>

# Combine with profiling for comprehensive analysis
ditto-go-tools --log-level debug --cpuprofile cpu.prof <command>
```

## Implementation Status

### Fully Implemented ✅
- All basic commands (sdk-version, print-config, smoke-test)
- Query execution with multiple output formats (JSON, table, CSV)
- All CRUD operations (insert, update, delete, evict, count, find-by-id)
- Real-time observation (observe command)
- Presence monitoring (presence command)
- Log export functionality (export-logs command)
- Configuration via CLI flags, environment variables, .env files, and TOML
- Multiple authentication systems
- Log level configuration (error, warning, info, debug)
- Runtime trace support for performance analysis
- CPU profiling (--cpuprofile)
- Memory allocation profiling (--memprofile)
- Heap profiling (--heapprofile)

### Not Yet Implemented ⚠️
- Attachment handling for queries and documents
- Metadata support for queries
- Save attachments feature in observe command
- Some authentication systems (full shared secret implementation)

## Testing

Run the test suite:
```bash
# Unit tests
go test ./...

# With race detection
go test -race ./...

# Integration tests (requires valid credentials)
DITTO_DATABASE_ID=xxx DITTO_PLAYGROUND_TOKEN=xxx go test -tags=integration ./...
```

## Contributing

### Adding New Commands

To add a new command:

1. Define the command structure in `main.go`
2. Add argument struct in `tools/toolkit.go`
3. Implement the `DoCommandName` function in `tools/toolkit.go`
4. Add any utility functions to `tools/utils.go`
5. Update this README with documentation

### Development Workflow

```bash
# Make changes
vim tools/toolkit.go

# Build
make build

# Test
./ditto-go-tools <new-command>

# Run with trace for performance analysis
./ditto-go-tools --trace perf.out <new-command>
go tool trace perf.out
```

## License

This tool is part of the Ditto Go SDK and is subject to Ditto's licensing terms.

## Support

For issues, questions, or contributions related to the Go SDK and tools:
- Linear Project: [Go SDK Project](https://linear.app/ditto/project/go-sdk-3fae8755cff3/overview)
- Documentation: [docs.ditto.live](https://docs.ditto.live)
