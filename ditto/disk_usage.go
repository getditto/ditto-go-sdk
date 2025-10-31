package ditto

import (
	"fmt"

	"github.com/getditto/ditto-go-sdk/internal/cbor"
	"github.com/getditto/ditto-go-sdk/internal/ffi"
)

// FileSystemType represents different types of file systems used by Ditto.
//
// Ditto organizes its persistent data into different file system components,
// each serving a specific purpose in the synchronization and storage system.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type FileSystemType int

const (
	// FileSystemRoot represents the root file system containing all Ditto data.
	// This includes all sub-components and is typically the persistence directory.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FileSystemRoot FileSystemType = iota

	// FileSystemStore represents the main document store file system.
	// This contains the actual document data and indexes.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FileSystemStore

	// FileSystemAuth represents the authentication file system.
	// This stores authentication credentials and tokens.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FileSystemAuth

	// FileSystemReplication represents the replication file system.
	// This contains sync metadata and replication logs.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FileSystemReplication

	// FileSystemAttachments represents the attachments file system.
	// This stores binary attachments associated with documents.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FileSystemAttachments
)

// DiskUsageChild represents disk usage information for a component or directory.
//
// This structure forms a tree representing the disk usage hierarchy,
// with each node containing its size and potentially child nodes.
//
// Example:
//
//	usage := dittoInstance.DiskUsage()
//	root, err := usage.Exec()
//	if err == nil {
//		fmt.Printf("Total size: %d bytes\n", root.SizeInBytes)
//		for _, child := range root.Children {
//			fmt.Printf("  %s: %d bytes\n", child.Path, child.SizeInBytes)
//		}
//	}
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type DiskUsageChild struct {
	// Path is the relative path of this component from the root
	Path string `json:"path"`

	// SizeInBytes is the total size of this component including children
	SizeInBytes uint64 `json:"sizeInBytes"`

	// FileSystemType indicates the type of file system component (optional)
	FileSystemType FileSystemType `json:"fileSystemType,omitempty"`

	// Children contains nested components or directories
	Children []*DiskUsageChild `json:"children,omitempty"`

	// Metadata contains additional component-specific information
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DiskUsage provides methods to query and monitor disk space used by Ditto.
//
// This component allows you to:
//   - Query current disk usage
//   - Monitor usage changes over time
//   - Identify which components use the most space
//
// Access this via Ditto.DiskUsage().
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type DiskUsage struct {
	ditto     *Ditto
	component ffi.FsComponent
}

// DiskUsageCallback is the function signature for disk usage change notifications.
//
// The callback receives a DiskUsageChild structure containing the current
// disk usage information whenever it changes.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type DiskUsageCallback func(usage *DiskUsageChild)

// DiskUsageObserver represents an active observation of disk usage changes.
//
// Call Stop() to stop receiving updates when no longer needed.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type DiskUsageObserver struct {
	handle *ffi.DiskUsageHandle
}

// newDiskUsage creates a new DiskUsage instance
func newDiskUsage(ditto *Ditto) *DiskUsage {
	return &DiskUsage{
		ditto:     ditto,
		component: ffi.FsComponentRoot,
	}
}

// Exec calculates the current disk usage
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *DiskUsage) Exec() (*DiskUsageChild, error) {
	if d.ditto == nil || d.ditto.dittoHandle == nil {
		return nil, fmt.Errorf("invalid ditto instance")
	}

	// Get disk usage from FFI
	cborData, err := ffi.GetDiskUsage(d.ditto.dittoHandle, d.component)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	// Decode CBOR to map
	resultMap, err := cbor.DecodeToMap(cborData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode disk usage: %w", err)
	}

	// Convert to DiskUsageChild
	return parseDiskUsageChild(resultMap), nil
}

// ExecForComponent calculates disk usage for a specific component
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *DiskUsage) ExecForComponent(component FileSystemType) (*DiskUsageChild, error) {
	if d.ditto == nil || d.ditto.dittoHandle == nil {
		return nil, fmt.Errorf("invalid ditto instance")
	}

	// Map FileSystemType to ffi.FsComponent
	var ffiComponent ffi.FsComponent
	switch component {
	case FileSystemRoot:
		ffiComponent = ffi.FsComponentRoot
	case FileSystemStore:
		ffiComponent = ffi.FsComponentStore
	case FileSystemAuth:
		ffiComponent = ffi.FsComponentAuth
	case FileSystemReplication:
		ffiComponent = ffi.FsComponentReplication
	case FileSystemAttachments:
		ffiComponent = ffi.FsComponentAttachment
	default:
		return nil, fmt.Errorf("invalid file system type")
	}

	// Get disk usage from FFI
	cborData, err := ffi.GetDiskUsage(d.ditto.dittoHandle, ffiComponent)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	// Decode CBOR to map
	resultMap, err := cbor.DecodeToMap(cborData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode disk usage: %w", err)
	}

	// Convert to DiskUsageChild
	return parseDiskUsageChild(resultMap), nil
}

// RegisterObserver registers a callback for disk usage changes
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *DiskUsage) RegisterObserver(callback DiskUsageCallback) *DiskUsageObserver {
	if d.ditto == nil || d.ditto.dittoHandle == nil {
		LogError("Invalid ditto instance for disk usage observer")
		return &DiskUsageObserver{handle: nil}
	}

	// Create a wrapper callback that decodes CBOR and calls the user callback
	wrapperCallback := func(cborData []byte) {
		resultMap, err := cbor.DecodeToMap(cborData)
		if err != nil {
			return
		}
		child := parseDiskUsageChild(resultMap)
		callback(child)
	}

	// Register the callback with FFI
	handle, err := ffi.RegisterDiskUsageCallback(d.ditto.dittoHandle, d.component, wrapperCallback)
	if err != nil {
		LogError("Failed to register disk usage observer: " + err.Error())
		return &DiskUsageObserver{handle: nil}
	}

	return &DiskUsageObserver{
		handle: handle,
	}
}

// Cancel stops observing disk usage changes
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (o *DiskUsageObserver) Cancel() {
	if o != nil && o.handle != nil {
		ffi.ReleaseDiskUsageCallback(o.handle)
		o.handle = nil
	}
}

// parseDiskUsageChild converts a map to DiskUsageChild
func parseDiskUsageChild(m map[string]any) *DiskUsageChild {
	child := &DiskUsageChild{
		Metadata: make(map[string]any),
	}

	if path, ok := m["path"].(string); ok {
		child.Path = path
	}

	// Handle different numeric types for size
	if size, ok := m["size_in_bytes"].(uint64); ok {
		child.SizeInBytes = size
	} else if size, ok := m["size_in_bytes"].(int64); ok {
		child.SizeInBytes = uint64(size)
	} else if size, ok := m["size_in_bytes"].(float64); ok {
		child.SizeInBytes = uint64(size)
	} else if size, ok := m["size"].(uint64); ok {
		child.SizeInBytes = size
	} else if size, ok := m["size"].(int64); ok {
		child.SizeInBytes = uint64(size)
	} else if size, ok := m["size"].(float64); ok {
		child.SizeInBytes = uint64(size)
	}

	// Parse file system type if present
	if fsType, ok := m["file_system_type"].(int); ok {
		child.FileSystemType = FileSystemType(fsType)
	} else if fsType, ok := m["file_system_type"].(float64); ok {
		child.FileSystemType = FileSystemType(int(fsType))
	}

	// Parse children recursively
	if children, ok := m["children"].([]any); ok {
		for _, item := range children {
			if childMap, ok := item.(map[string]any); ok {
				child.Children = append(child.Children, parseDiskUsageChild(childMap))
			}
		}
	}

	// Copy any additional metadata
	for k, v := range m {
		if k != "path" && k != "size_in_bytes" && k != "size" &&
			k != "children" && k != "file_system_type" {
			child.Metadata[k] = v
		}
	}

	return child
}

// DiskUsage returns the DiskUsage API for this Ditto instance
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *Ditto) DiskUsage() *DiskUsage {
	if d.diskUsage == nil {
		d.diskUsage = newDiskUsage(d)
	}
	return d.diskUsage
}

// GetTotalSize recursively calculates the total size including all children
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *DiskUsageChild) GetTotalSize() uint64 {
	total := d.SizeInBytes
	for _, child := range d.Children {
		total += child.GetTotalSize()
	}
	return total
}

// FindByPath searches for a child with the given path
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (d *DiskUsageChild) FindByPath(path string) *DiskUsageChild {
	if d.Path == path {
		return d
	}
	for _, child := range d.Children {
		if found := child.FindByPath(path); found != nil {
			return found
		}
	}
	return nil
}
