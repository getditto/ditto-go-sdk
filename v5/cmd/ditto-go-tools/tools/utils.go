package tools

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/ditto"
)

// collectionNameRe is used in isValidCollectionName
var collectionNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// GetSDKVersion returns the Ditto SDK version
func GetSDKVersion() string {
	return ditto.Version()
}

// underscoreToHyphen returns a copy of s with any underscore (_) characters converted to
// hyphen-minus (-).
func underscoreToHyphen(s string) string {
	return strings.ReplaceAll(s, "_", "-")
}

var logLevelMap = map[string]ditto.LogLevel{
	"error":   ditto.LogLevelError,
	"warning": ditto.LogLevelWarning,
	"info":    ditto.LogLevelInfo,
	"debug":   ditto.LogLevelDebug,
}

// toLogLevel converts string to Ditto log level
// Returns an error if the given string is not a valid log level.
func toLogLevel(s string) (ditto.LogLevel, error) {
	if s == "" {
		return ditto.LogLevelError, fmt.Errorf("log level cannot be empty")
	}

	if level, ok := logLevelMap[strings.ToLower(s)]; ok {
		return level, nil
	}

	return ditto.LogLevelError, fmt.Errorf("valid log levels are error, warning, info, and debug")
}

// toJSONString converts a DQL query result to a JSON string
// TODO: This needs to be adapted once we understand the Go SDK's QueryResult structure
func toJSONString(result any) (string, error) {
	// For now, we'll use a generic approach with JSON marshaling
	// This will need to be updated when we integrate with the actual Go SDK
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result to JSON: %w", err)
	}
	return string(data), nil
}

// QueryResultData represents the structured data from a query result
//
// TODO: add ModifiedDocumentIDs field
type QueryResultData struct {
	// Items is an array of JSON string data for the items.
	Items []string `json:"items"`
}

// queryResultDataFromQueryResult converts a ditto.QueryResult to our internal QueryResultData format
func queryResultDataFromQueryResult(result *ditto.QueryResult) (*QueryResultData, error) {
	var items []string

	// Convert items from the QueryResult to JSON strings
	for _, item := range result.Items() {
		jsonString := item.JSONString()
		items = append(items, jsonString)
	}

	return &QueryResultData{
		Items: items,
	}, nil
}

// formatTableOutput formats query results as a table
func formatTableOutput(resultData *QueryResultData) string {
	var sb strings.Builder

	if len(resultData.Items) == 0 {
		sb.WriteString("No results found.\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Results: %d item(s)\n", len(resultData.Items)))
	sb.WriteString(strings.Repeat("-", 80) + "\n")

	for i, item := range resultData.Items {
		sb.WriteString(fmt.Sprintf("Item %d:\n", i+1))

		// Try to parse and pretty-print the JSON
		var itemData any
		if err := json.Unmarshal([]byte(item), &itemData); err == nil {
			if prettyJSON, err := json.MarshalIndent(itemData, "", "  "); err == nil {
				sb.WriteString(string(prettyJSON) + "\n")
			} else {
				sb.WriteString(item + "\n")
			}
		} else {
			sb.WriteString(item + "\n")
		}

		if i < len(resultData.Items)-1 {
			sb.WriteString(strings.Repeat("-", 40) + "\n")
		}
	}

	return sb.String()
}

// formatCSVOutput formats query results as CSV using the standard library
func formatCSVOutput(resultData *QueryResultData) (string, error) {
	if len(resultData.Items) == 0 {
		return "", nil
	}

	// Parse all items to collect unique keys
	allKeys := make(map[string]bool)
	var parsedItems []map[string]any

	for _, item := range resultData.Items {
		var itemData map[string]any
		if err := json.Unmarshal([]byte(item), &itemData); err != nil {
			return "", fmt.Errorf("failed to parse item for CSV: %w", err)
		}
		parsedItems = append(parsedItems, itemData)

		for key := range itemData {
			allKeys[key] = true
		}
	}

	// Sort keys for consistent output
	var keys []string
	for key := range allKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write CSV header
	if err := writer.Write(keys); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write CSV rows
	for _, item := range parsedItems {
		row := make([]string, len(keys))
		for i, key := range keys {
			if value, exists := item[key]; exists {
				if value == nil {
					row[i] = ""
				} else if str, ok := value.(string); ok {
					row[i] = str
				} else {
					// Convert non-string values to JSON
					if jsonValue, err := json.Marshal(value); err == nil {
						row[i] = string(jsonValue)
					} else {
						row[i] = ""
					}
				}
			} else {
				row[i] = ""
			}
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.String(), nil
}

// timeoutOrInterruptSignal waits for a SIGINT or SIGTERM, or for a timeout to expire,
// then sends an event on the returned channel.
//
// If timeout is 0 or negative, then timeout is ignored and only an interrupt signal will be detected.
//
// Returns the notification channel and a cancel function to stop waiting and clean up resources.
func timeoutOrInterrupt(timeout time.Duration) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		interrupted := make(chan os.Signal, 1)
		signal.Notify(interrupted, syscall.SIGINT, syscall.SIGTERM)
		var timedout <-chan time.Time
		if timeout > 0 {
			timedout = time.After(timeout)
		}

		select {
		case <-interrupted:
		case <-timedout:
		}

		signal.Stop(interrupted) // Stop receiving signals
		close(done)
	}()

	return done
}

// isValidCollectionName checks whether a given string is a valid unquoted collection name.
//
// This should be called whenever building a DQL query string that includes a user-supplied collection name.
//
// To keep things simple, and avoid DQL injection problems, this tool doesn't support quoted collection names.
func isValidCollectionName(name string) bool {
	// Collection names must meet these requirements:
	// - Start with a letter or underscore
	// - Followed by zero or more letters, digits, or underscores
	// - Length must be less than 100 characters
	// - Cannot start with "$TS_" (reserved prefix)

	if len(name) == 0 || len(name) >= 100 {
		return false
	}

	if strings.HasPrefix(name, "$TS_") {
		return false
	}

	matched := collectionNameRe.MatchString(name)

	return matched
}
