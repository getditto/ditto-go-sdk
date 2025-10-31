package ditto

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/getditto/ditto-go-sdk/internal/ffi"
)

// SmallPeerInfo is the entrypoint for small peer user info collection. Small peer info consists of information
// gathered into a system collection on a regular interval and optionally synced to the Big Peer
// for device dashboard and debugging purposes.
//
// You don't create this class directly, but can access it from a particular [Ditto]
// instance via its [Ditto.SmallPeerInfo] method.
type SmallPeerInfo struct {
	mu          sync.RWMutex
	dittoHandle *ffi.DittoHandle
}

// newSmallPeerInfo creates a new SmallPeerInfo instance.
//
// Parameters:
//   - dittoHandle: The Ditto handle for FFI operations
//
// Returns:
//   - *SmallPeerInfo: A new SmallPeerInfo instance
func newSmallPeerInfo(dittoHandle *ffi.DittoHandle) *SmallPeerInfo {
	return &SmallPeerInfo{
		dittoHandle: dittoHandle,
	}
}

// IsEnabled indicates whether small peer info collection is currently enabled, defaults
// to `true`.
//
// Note: whether the background ingestion process is enabled or not is a
// separate decision to whether this information is allowed to sync to other
// peers (including the big peer). This is controlled by [sync scopes].
//
// [sync scopes]: https://docs.ditto.live/sdk/latest/sync/sync-scopes
func (s *SmallPeerInfo) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.dittoHandle.IsValid() {
		LogError("SmallPeerInfo.IsEnabled called with invalid handle")
		return false
	}

	return ffi.SmallPeerInfoGetIsEnabled(s.dittoHandle)
}

// SetEnabled sets whether small peer info collection is currently enabled.
func (s *SmallPeerInfo) SetEnabled(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dittoHandle.IsValid() {
		return ErrDittoClosed
	}

	ffi.SetSmallPeerInfoEnabled(s.dittoHandle, enabled)

	return nil
}

// Metadata Returns the JSON metadata being associated with the small peer info.
//
// Returns:
//   - map[string]any: The current metadata dictionary
//   - an empty map if there is an error retrieving or unmarshalling the metadata or if no metadata has been set.
func (s *SmallPeerInfo) Metadata() map[string]any {
	jsonStr := s.MetadataJSONString()

	if jsonStr == "" {
		// Empty JSON string means no metadata set
		return make(map[string]any)
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &metadata); err != nil {
		LogErrorF("failed to unmarshal metadata: %v", err)
		return make(map[string]any)
	}
	return metadata
}

// MetadataJSONString Returns the JSON metadata being associated with the small peer info.
//
// Returns:
//   - non-empty JSON string: The current metadata dictionary
//   - empty string if there is an error retrieving the metadata or if no metadata has been set.
func (s *SmallPeerInfo) MetadataJSONString() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Handle case where dittoHandle is nil - return empty map (safe for read)
	if !s.dittoHandle.IsValid() {
		return ""
	}

	// If we have a valid handle, we should get real data from FFI
	return ffi.SmallPeerInfoGetMetadata(s.dittoHandle)
}

// SetMetadata sets the JSON metadata to be associated with the small peer info.
//
// The metadata is a free-form, user-provided dictionary that is serialized into
// JSON and is inserted into the small peer info system doc at each collection
// interval. The dictionary has no schema except for the following constraints:
//
//  1. All contained values must be JSON serializable.
//  2. The size of the dictionary serialized as JSON may not exceed 128 KB.
//  3. The dictionary may only be nested up to 64 levels deep.
func (s *SmallPeerInfo) SetMetadata(metadata map[string]any) error {
	// Convert metadata to JSON
	var jsonStr string
	if metadata == nil {
		// Handle nil metadata by setting empty JSON object
		jsonStr = "{}"
	} else {
		jsonBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		jsonStr = string(jsonBytes)
	}

	return s.SetMetadataJSONString(jsonStr)
}

// SetMetadataJSONString sets the JSON metadata to be associated with the small peer info.
//
// The metadata is a free-form, user-provided dictionary that is serialized into
// JSON and is inserted into the small peer info system doc at each collection
// interval. The dictionary has no schema except for the following constraints:
//
//  1. All contained values must be JSON serializable.
//  2. The size of the dictionary serialized as JSON may not exceed 128 KB.
//  3. The dictionary may only be nested up to 64 levels deep.
func (s *SmallPeerInfo) SetMetadataJSONString(jsonStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.dittoHandle.IsValid() {
		return ErrDittoClosed
	}

	if err := ffi.SmallPeerInfoSetMetadata(s.dittoHandle, jsonStr); err != nil {
		return err
	}

	return nil
}
