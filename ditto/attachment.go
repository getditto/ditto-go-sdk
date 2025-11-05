package ditto

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// Attachment represents a file attachment that can be synced via Ditto.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type Attachment struct {
	id          string
	token       *AttachmentToken
	metadata    map[string]any
	size        int64
	path        string
	handle      *ffi.AttachmentHandle
	dittoHandle *ffi.DittoHandle // Reference to Ditto handle for FFI calls
}

// ID returns the unique identifier of the attachment.
// The ID is a hex-encoded string that uniquely identifies this attachment
// across all peers in the Ditto network.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) ID() string {
	return a.id
}

// Len returns the size of the attachment in bytes.
// This represents the total size of the binary data.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Len() int64 {
	return a.size
}

// newAttachment creates a new attachment from a file path.
//
// Parameters:
//   - dittoHandle: The Ditto handle for FFI operations
//   - path: The file system path to the file to attach
//   - metadata: Optional metadata to associate with the attachment (can be nil)
//
// Returns:
//   - *Attachment: The created attachment
//   - error: An error if the file cannot be read or attachment creation fails
//
// The file is copied into Ditto's storage, so the original file can be deleted
// after this function returns successfully.
//
// Example:
//
//	metadata := map[string]any{
//		"contentType": "application/pdf",
//		"uploadedBy": "user123",
//	}
//	attachment, err := newAttachment(ditto.Handle, "/path/to/document.pdf", metadata)
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func newAttachment(dittoHandle *ffi.DittoHandle, path string, metadata map[string]any) (*Attachment, error) {
	// Check if file exists and get its size
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Create attachment using FFI
	attachment, err := ffi.NewAttachmentFromFile(dittoHandle, path, true) // Copy the file
	if err != nil {
		return nil, err
	}

	// Convert ID to hex string
	idStr := hex.EncodeToString(attachment.ID)
	if idStr == "" {
		// Generate a temporary ID if FFI didn't provide one
		// This might happen with mock or incomplete FFI implementations
		idStr = fmt.Sprintf("temp_%d", time.Now().UnixNano())
	}

	// Create token
	token := NewAttachmentToken(idStr, int64(attachment.Length), metadata)

	return &Attachment{
		id:          idStr,
		token:       token,
		metadata:    metadata,
		size:        fileInfo.Size(),
		path:        path,
		handle:      attachment.Handle,
		dittoHandle: dittoHandle,
	}, nil
}

// Token returns the attachment token that can be stored in documents.
// The token contains the attachment ID, size, and metadata, and is used
// to reference the attachment from within documents.
//
// The token should be stored in document fields where you want to reference
// this  When other peers sync the document, they can use the token
// to fetch the attachment data on-demand.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Token() *AttachmentToken {
	return a.token
}

// Metadata returns the attachment's metadata dictionary.
// Metadata can contain any JSON-serializable data that describes the attachment,
// such as content type, original filename, creation date, or custom attributes.
//
// The returned map is a reference to the internal metadata. Modifications to
// the returned map will affect the attachment's metadata.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Metadata() map[string]any {
	return a.metadata
}

// Size returns the size of the attachment in bytes.
// This is equivalent to Len() and provided for convenience.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Size() int64 {
	return a.size
}

// Path returns the local file system path where the attachment is stored.
// This path is managed by Ditto and should not be modified directly.
// The path may be empty if the attachment was created from bytes or
// if it hasn't been fetched yet.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Path() string {
	return a.path
}

// CreatedAt returns when the attachment was created.
// This timestamp is typically stored in the attachment's metadata.
// If no creation time is stored, returns the current time.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) CreatedAt() time.Time {
	// Would be stored in the attachment metadata
	if created, ok := a.metadata["createdAt"].(time.Time); ok {
		return created
	}
	return time.Now()
}

// UpdatedAt returns when the attachment was last updated.
// This timestamp is typically stored in the attachment's metadata.
// If no update time is stored, returns the current time.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) UpdatedAt() time.Time {
	// Would be stored in the attachment metadata
	if updated, ok := a.metadata["updatedAt"].(time.Time); ok {
		return updated
	}
	return time.Now()
}

// Handle returns the underlying FFI attachment handle.
// This is primarily for internal use and should not be accessed directly
// in application code.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Handle() *ffi.AttachmentHandle {
	return a.handle
}

// Data retrieves the attachment's binary content as a byte slice.
//
// This method loads the entire attachment into memory. For large attachments,
// consider using Reader() or CopyToPath() instead to avoid high memory usage.
//
// Returns:
//   - []byte: The attachment's binary data
//   - error: An error if the data cannot be retrieved
//
// Example:
//
//	data, err := attachment.Data()
//	if err != nil {
//	    log.Printf("Failed to get attachment data: %v", err)
//	} else {
//	    // Process the binary data
//	    fmt.Printf("Attachment size: %d bytes\n", len(data))
//	}
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Data() ([]byte, error) {
	if a.handle == nil {
		return nil, fmt.Errorf("attachment handle is nil")
	}
	if a.dittoHandle == nil {
		return nil, fmt.Errorf("ditto handle is nil")
	}

	// Get the data using FFI
	data, err := ffi.GetCompleteAttachmentData(a.dittoHandle, a.handle)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment data: %w", err)
	}

	return data, nil
}

// CopyToPath copies the attachment to a specified file path.
//
// This method efficiently copies the attachment data to the destination path
// without loading the entire content into memory. The destination directory
// must exist, and the file will be overwritten if it already exists.
//
// Parameters:
//   - destinationPath: The file path where the attachment should be copied
//
// Returns:
//   - error: An error if the copy operation fails
//
// Example:
//
//	err := attachment.CopyToPath("/path/to/destination.pdf")
//	if err != nil {
//	    log.Printf("Failed to copy attachment: %v", err)
//	}
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) CopyToPath(destinationPath string) error {
	if a.handle == nil {
		return fmt.Errorf("attachment handle is nil")
	}
	if a.dittoHandle == nil {
		return fmt.Errorf("ditto handle is nil")
	}

	// Get the attachment's current path
	sourcePath, err := ffi.GetCompleteAttachmentPath(a.dittoHandle, a.handle)
	if err != nil {
		return fmt.Errorf("failed to get attachment path: %w", err)
	}

	// Open the source file
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		err = errors.Join(err, source.Close())
	}()

	// Create the destination file
	destination, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		err = errors.Join(err, destination.Close())
	}()

	// Copy the file
	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// Reader returns an io.ReadSeeker for reading the attachment's content.
//
// This method provides a reader that can be used to read the attachment data
// without loading it entirely into memory. The reader supports seeking, which
// allows for partial reads and repositioning within the attachment.
//
// The caller is responsible for closing the returned reader when done.
//
// Returns:
//   - io.ReadSeeker: A reader for the attachment content
//   - error: An error if the reader cannot be created
//
// Example:
//
//	reader, err := attachment.Reader()
//	if err != nil {
//	    log.Printf("Failed to get reader: %v", err)
//	    return
//	}
//	defer reader.(io.Closer).Close()
//
//	// Read first 1024 bytes
//	buffer := make([]byte, 1024)
//	n, err := reader.Read(buffer)
//	if err != nil && err != io.EOF {
//	    log.Printf("Failed to read: %v", err)
//	}
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Reader() (io.ReadSeeker, error) {
	if a.handle == nil {
		return nil, fmt.Errorf("attachment handle is nil")
	}
	if a.dittoHandle == nil {
		return nil, fmt.Errorf("ditto handle is nil")
	}

	// Get the attachment's path
	path, err := ffi.GetCompleteAttachmentPath(a.dittoHandle, a.handle)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment path: %w", err)
	}

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open attachment file: %w", err)
	}

	// The file implements io.ReadSeeker
	return file, nil
}

// Free releases the attachment resources.
// This should be called when the attachment is no longer needed to free
// the underlying FFI resources. After calling Free, the attachment should
// not be used.
//
// It is safe to call Free multiple times; subsequent calls will be no-ops.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (a *Attachment) Free() {
	if a.handle != nil {
		ffi.FreeAttachmentHandle(a.handle)
		a.handle = nil
	}
}
