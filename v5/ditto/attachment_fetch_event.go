package ditto

import (
	"fmt"
)

// AttachmentFetchEvent represents an event that occurs during attachment fetching.
// Events are delivered to the callback function provided when creating a fetcher.
//
// Different event types provide different information:
//   - Progress events: Include Progress, BytesDownloaded, and TotalBytes
//   - Completed events: Include Path to the downloaded file
//   - Failed events: Include Error with failure details
//   - Canceled events: Indicate the fetch was canceled by the user
//
// Example handling:
//
//	func handleFetchEvent(event *AttachmentFetchEvent) {
//		switch event.Type {
//		case FetchEventTypeProgress:
//			percentage := event.Progress * 100
//			fmt.Printf("Progress: %.1f%%\n", percentage)
//		case FetchEventTypeCompleted:
//			fmt.Printf("Download completed: %s\n", event.Path)
//		case FetchEventTypeFailed:
//			fmt.Printf("Download failed: %v\n", event.Error)
//		case FetchEventTypeCanceled:
//			fmt.Println("Download canceled")
//		}
//	}
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type AttachmentFetchEvent struct {
	// Type indicates what kind of event this is
	Type FetchEventType
	// Progress indicates fetch completion (0.0 to 1.0), valid for progress events
	Progress float64
	// BytesDownloaded is the number of bytes downloaded so far, valid for progress events
	BytesDownloaded int64
	// TotalBytes is the total size of the attachment, valid for progress events
	TotalBytes int64
	// Error contains failure details, valid for failed events
	Error error
	// Path is the local file path where the attachment was saved, valid for completed events
	Path string
}

// FetchEventType represents the type of fetch event.
// Events are delivered in sequence: multiple progress events,
// followed by exactly one terminal event (completed, failed, or canceled).
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type FetchEventType int

const (
	// FetchEventTypeProgress indicates a fetch progress update.
	// These events are sent periodically during download to report progress.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FetchEventTypeProgress FetchEventType = iota

	// FetchEventTypeCompleted indicates the fetch completed successfully.
	// The attachment is now available locally at the path specified in the event.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FetchEventTypeCompleted

	// FetchEventTypeFailed indicates the fetch failed with an error.
	// The Error field contains details about what went wrong.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FetchEventTypeFailed

	// FetchEventTypeCanceled indicates the fetch was canceled by the user.
	// This happens when Cancel() is called on the fetcher.
	//
	// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
	FetchEventTypeCanceled
)

// AttachmentFetchEventType is an alias for FetchEventType for backward compatibility.
// New code should use FetchEventType.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type AttachmentFetchEventType = FetchEventType

// Backward compatibility constants.
// New code should use the FetchEventType* constants.
const (
	AttachmentFetchEventTypeProgress  = FetchEventTypeProgress
	AttachmentFetchEventTypeCompleted = FetchEventTypeCompleted
	AttachmentFetchEventTypeFailed    = FetchEventTypeFailed
	AttachmentFetchEventTypeCanceled  = FetchEventTypeCanceled
)

// NewProgressEvent creates a new progress event.
//
// Parameters:
//   - progress: The completion percentage (0.0 to 1.0)
//
// Returns:
//   - *AttachmentFetchEvent: A progress event
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewProgressEvent(progress float64) *AttachmentFetchEvent {
	return &AttachmentFetchEvent{
		Type:     FetchEventTypeProgress,
		Progress: progress,
	}
}

// NewCompletedEvent creates a new completion event.
//
// Parameters:
//   - path: The local file system path where the attachment was saved
//
// Returns:
//   - *AttachmentFetchEvent: A completion event
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewCompletedEvent(path string) *AttachmentFetchEvent {
	return &AttachmentFetchEvent{
		Type:     FetchEventTypeCompleted,
		Progress: 1.0,
		Path:     path,
	}
}

// NewFailedEvent creates a new failure event.
//
// Parameters:
//   - err: The error that caused the fetch to fail
//
// Returns:
//   - *AttachmentFetchEvent: A failure event
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewFailedEvent(err error) *AttachmentFetchEvent {
	return &AttachmentFetchEvent{
		Type:  FetchEventTypeFailed,
		Error: err,
	}
}

// NewCanceledEvent creates a new cancellation event.
//
// Returns:
//   - *AttachmentFetchEvent: A cancellation event
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewCanceledEvent() *AttachmentFetchEvent {
	return &AttachmentFetchEvent{
		Type: FetchEventTypeCanceled,
	}
}

// String returns a human-readable string representation of the event.
// Useful for logging and debugging.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (e *AttachmentFetchEvent) String() string {
	switch e.Type {
	case FetchEventTypeProgress:
		return fmt.Sprintf("Progress: %.2f%%", e.Progress*100)
	case FetchEventTypeCompleted:
		return fmt.Sprintf("Completed: %s", e.Path)
	case FetchEventTypeFailed:
		return fmt.Sprintf("Failed: %v", e.Error)
	case FetchEventTypeCanceled:
		return "Canceled"
	default:
		return "Unknown"
	}
}

// IsCompleted returns true if this is a completion event.
// Completed events indicate successful download of the
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (e *AttachmentFetchEvent) IsCompleted() bool {
	return e.Type == FetchEventTypeCompleted
}

// IsFailed returns true if this is a failure event.
// Failed events include an Error field with details about the failure.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (e *AttachmentFetchEvent) IsFailed() bool {
	return e.Type == FetchEventTypeFailed
}

// IsCanceled returns true if this is a cancellation event.
// Canceled events occur when the user calls Cancel() on the fetcher.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (e *AttachmentFetchEvent) IsCanceled() bool {
	return e.Type == FetchEventTypeCanceled
}

// NewAttachmentFetchEventProgress creates a new progress event.
// This is an alias for NewProgressEvent provided for backward compatibility.
// New code should use NewProgressEvent.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewAttachmentFetchEventProgress(progress float64) *AttachmentFetchEvent {
	return NewProgressEvent(progress)
}

// NewAttachmentFetchEventCompleted creates a new completion event.
// This is an alias for NewCompletedEvent provided for backward compatibility.
// New code should use NewCompletedEvent.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewAttachmentFetchEventCompleted(path string) *AttachmentFetchEvent {
	return NewCompletedEvent(path)
}

// NewAttachmentFetchEventDeleted creates a new cancellation event.
// This is an alias for NewCanceledEvent provided for backward compatibility.
// The name "Deleted" is maintained for compatibility but actually represents cancellation.
// New code should use NewCanceledEvent.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func NewAttachmentFetchEventDeleted() *AttachmentFetchEvent {
	return NewCanceledEvent()
}
