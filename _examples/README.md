# Ditto Go SDK Examples

This directory contains example programs demonstrating how to use the Ditto Go SDK.

## Prerequisites

- Go 1.24 or later
- A Ditto account (create one at https://portal.ditto.live/)

## Structure

The examples are organized as a separate Go module with individual subdirectories for each example:

```
_examples/
  go.mod                    # Separate module for examples
  go.sum
  sync-example/                # Real-time data synchronization
    main.go
  presence-example/            # Peer presence and discovery
    main.go
  transport-diagnostics-example/ # Network transport monitoring
    main.go
  auth-example/                # Authentication and logout
    main.go
  execute-example/             # DQL Execute operations
    main.go
  transaction-example/         # Transaction operations
    main.go
```

## Configuration

All example programs support configuration through environment variables or a `.env` file. The precedence order is:
1. Environment variables (highest priority)
2. `.env` file in the example directory
3. Default values in the code (lowest priority)

### Using Environment Variables

```bash
export DITTO_APP_ID="your-app-id"
export DITTO_AUTH_URL="https://your-app-id.cloud.ditto.live"
export DITTO_PLAYGROUND_TOKEN="your-playground-token"
```

### Using a .env File

Create a `.env` file in the example directory:

```env
DITTO_APP_ID=your-app-id
DITTO_AUTH_URL=https://your-app-id.cloud.ditto.live
DITTO_PLAYGROUND_TOKEN=your-playground-token
```

When environment variables or .env entries are not set, the examples will use default placeholder values.

You can obtain credentials from your app in the [Ditto Portal](https://portal.ditto.live/).

## Examples

### 1. sync-example - Real-time Data Synchronization

This example demonstrates how to:
- Connect to Ditto Cloud with authentication
- Set up real-time data observers
- Subscribe to sync data from other peers
- Handle authentication token expiration
- Export Ditto logs

#### Building and Running

```bash
# Navigate to the example directory
cd sync-example

# Build the example
go build

# Run with environment variables
DITTO_APP_ID="your-app-id" \
DITTO_AUTH_URL="https://your-app-id.cloud.ditto.live" \
DITTO_PLAYGROUND_TOKEN="your-token" \
./sync-example

# Or export them first
export DITTO_APP_ID="your-app-id"
export DITTO_AUTH_URL="https://your-app-id.cloud.ditto.live"
export DITTO_PLAYGROUND_TOKEN="your-token"
./sync-example

# Or run directly without building
go run main.go
```

The program will:
1. Connect to your Ditto app
2. Register an observer for the "tasks" collection
3. Display any updates received in real-time
4. Run for 60 seconds by default (or use `-wait` flag to specify duration, 0 or negative for forever)
5. Export logs to `./ditto-go-example-sync-example-logexport.jsonl.gz`

#### Customization

To observe a different collection or query, modify the observer registration in the source code:

```go
observerQuery := "SELECT * FROM your_collection WHERE your_condition"
var observerArgs ditto.QueryArguments = nil // or provide arguments
observer, err := d.Store().RegisterObserver(observerQuery, observerArgs,
    func(result *ditto.QueryResult) {
        defer result.Close()
        // Handle updates
    },
)
```

### 2. presence-example - Peer Presence and Discovery

This example demonstrates how to observe peer presence and handle connection requests between Ditto peers.

#### Features
- Connect to Ditto Cloud for presence synchronization
- Observe peer join/leave events in real-time
- Set and display peer metadata
- Handle connection requests between peers
- Support multiple instances with unique IDs
- Automatically generate random instance IDs

#### Building and Running

```bash
# Navigate to the example directory
cd presence-example

# Build the example
go build

# Run with environment variables and a specific instance ID
DITTO_APP_ID="your-app-id" \
DITTO_AUTH_URL="https://your-app-id.cloud.ditto.live" \
DITTO_PLAYGROUND_TOKEN="your-token" \
./presence-example -id mydevice1

# Run with an auto-generated random ID
./presence-example

# Run multiple instances in separate terminals to see peer discovery
./presence-example -id device1 &
./presence-example -id device2 &
./presence-example -id device3 &
```

The program will:
1. Connect to Ditto Cloud with authentication
2. Set peer metadata including device name and instance ID
3. Register a presence observer to detect peer changes
4. Display join/leave events for discovered peers
5. Show peer metadata and connection details
6. Run for 30 seconds by default (or use `-wait` flag to specify duration, 0 or negative for forever)
7. Export logs to `./ditto-go-example-presence-example-<instanceID>-logexport.jsonl.gz`

#### Command-line Options

- `-id <string>`: Specify a custom instance ID (optional, random if not provided)
- `-wait <int>`: Number of seconds to run before shutting down (default 30, 0 or negative for forever)
- `-persistence-directory <string>`: Directory for Ditto persistence (defaults to /tmp/ditto-go-example-presence-example-<id>)
- `-log-export-path <string>`: Path for exported log file (defaults to ./ditto-go-example-presence-example-<id>-logexport.jsonl.gz)

### 3. transport-diagnostics-example - Network Transport Monitoring

This example demonstrates how to monitor network transport status and connections:
- Connect to Ditto Cloud with authentication
- Query transport diagnostics every 5 seconds
- Display connection states for all transports
- Show which peers are connected via each transport type
- Export Ditto logs

#### Building and Running

```bash
# Navigate to the example directory
cd transport-diagnostics-example

# Build and run with environment variables
DITTO_APP_ID="your-app-id" \
DITTO_AUTH_URL="https://your-app-id.cloud.ditto.live" \
DITTO_PLAYGROUND_TOKEN="your-token" \
go run main.go
```

The program will:
1. Connect to your Ditto app
2. Start all available transports
3. Print transport diagnostics every 10 seconds in JSON format
4. Show connection states (connecting, connected, disconnecting, disconnected)
5. Display a summary of active transports and connections
6. Export logs to `./ditto-go-example-transport-diagnostics-example-logexport.jsonl.gz`

#### Output Format

The diagnostics show each transport with arrays of site IDs in different connection states:

```json
{
  "transports": [
    {
      "connection_type": "WebSocket",
      "connecting": [],
      "connected": [9037],
      "disconnecting": [],
      "disconnected": []
    }
  ]
}
```

### 4. auth-example - Authentication and Logout

This example demonstrates how to:
- Connect to Ditto Cloud with authentication
- Perform authentication login with a playground token
- Execute logout to clear authentication state
- Handle authentication callbacks

#### Building and Running

```bash
# Navigate to the example directory
cd auth-example

# Build and run with environment variables
DITTO_APP_ID="your-app-id" \
DITTO_AUTH_URL="https://your-app-id.cloud.ditto.live" \
DITTO_PLAYGROUND_TOKEN="your-token" \
go run main.go
```

The program will:
1. Connect to Ditto Cloud with authentication
2. Login using the playground token
3. Display authentication status and client info
4. Perform logout operation
5. Display logout confirmation
6. Demonstrate cleanup callbacks

### 5. execute-example - DQL Execute Operations

This example demonstrates how to:
- Use Ditto with offline identity (no sync needed)
- Execute DQL INSERT statements to add documents
- Execute SELECT queries with WHERE clauses and parameters
- Execute UPDATE statements to modify documents
- Execute DELETE statements with conditions
- Perform complex queries with multiple conditions
- Handle query results and iterate over items

#### Building and Running

```bash
# Navigate to the example directory
cd execute-example

# Build the example
go build

# Run the example (no environment variables needed for offline mode)
./execute-example

# Or run directly without building
go run main.go
```

The program will:
1. Create a Ditto instance with offline identity
2. Insert sample car documents into a collection
3. Demonstrate SELECT queries with various filters and sorting
4. Update documents using SET operations
5. Delete documents based on conditions
6. Show complex query patterns with multiple conditions
7. Clean up by removing the persistence directory

#### Example Operations

The execute-example example demonstrates:
- **INSERT**: Adding single and multiple documents with parameters
- **SELECT**: Querying with WHERE, ORDER BY, LIMIT clauses
- **UPDATE**: Modifying documents with SET operations and conditions
- **DELETE**: Removing documents based on criteria
- **Complex Queries**: Using IN operators, multiple conditions, and COUNT aggregation

### 6. transaction-example - Transaction Operations

This example demonstrates how to:
- Use Ditto with offline identity (no sync needed)
- Execute DQL statements within a transaction
- Perform read/write operations with transaction isolation
- Handle transaction commit and rollback
- Use transactions to ensure atomic operations

#### Building and Running

```bash
# Navigate to the example directory
cd transaction-example

# Build the example
go build

# Run the example (no environment variables needed for offline mode)
./transaction-example

# Or run directly without building
go run main.go
```

The program will:
1. Create a Ditto instance with offline identity
2. Insert initial car documents
3. Demonstrate read operations within a transaction
4. Perform write operations (insert/update) in a transaction
5. Show atomic commit of multiple changes
6. Clean up by removing the persistence directory

#### Example Operations

The transaction-example example demonstrates:
- **Read Transactions**: Executing SELECT queries within a transaction
- **Write Transactions**: Performing INSERT, UPDATE operations atomically
- **Transaction Isolation**: Ensuring consistency across multiple operations
- **Commit/Rollback**: Properly completing or canceling transactions

## Building All Examples

You can build all examples at once from the `_examples` directory:

```bash
# From the _examples directory
for dir in */; do
  echo "Building $dir"
  (cd "$dir" && go build)
done
```

## Data Persistence

All examples store data in temporary directories:
- **sync-example**: `/tmp/ditto-go-example-sync-example/`
- **presence-example**: `/tmp/ditto-go-example-presence-example-<instanceID>/`
- **transport-diagnostics-example**: `/tmp/ditto-go-example-transport-diagnostics-example/`
- **auth-example**: `/tmp/ditto-go-example-auth-example/`
- **execute-example**: `/tmp/ditto-go-example-execute-example/`
- **transaction-example**: `/tmp/ditto-go-example-transaction-example/`

These directories contain the local Ditto database and can be deleted to start fresh.

## Troubleshooting

### Authentication Errors
Ensure your app credentials are correct and your playground token hasn't expired.

### Transport Errors
Some peer-to-peer transports may not be available on all platforms:
- WiFi Aware is Android-only
- AWDL is Apple-only
- Bluetooth LE requires appropriate permissions

### Build Errors
The examples use a local replace directive in go.mod to use the local SDK. If you encounter issues:
```bash
cd _examples
go mod tidy
```

### Module Structure
These examples are organized as a separate Go module to prevent interference with the main SDK build process. The `go.mod` file contains a replace directive that points to the parent directory for local development.

## Additional Resources

- [Ditto Documentation](https://docs.ditto.live/)
- [Ditto Query Language (DQL)](https://docs.ditto.live/dql/dql)
- [Ditto Go SDK API Reference](https://pkg.go.dev/github.com/getditto/ditto-go-sdk/v5)
