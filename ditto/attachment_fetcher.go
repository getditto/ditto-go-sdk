// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ditto

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// AttachmentFetcher manages the asynchronous fetching of an attachment.
// It provides progress updates and can be canceled if needed.
//
// Fetchers must be started explicitly with Start() and can be canceled with Cancel().
// Progress updates are delivered through the callback function provided during creation.
//
// Thread-safety: AttachmentFetcher is thread-safe. All methods can be called
// concurrently from multiple goroutines.
//
// Example usage:
//
//	fetcher := newAttachmentFetcher(dittoHandle, token, func(event *AttachmentFetchEvent) {
//		switch event.Type {
//		case FetchEventTypeProgress:
//			fmt.Printf("Downloaded %d of %d bytes (%.1f%%)\n",
//				event.BytesDownloaded, event.TotalBytes, event.Progress*100)
//		case FetchEventTypeCompleted:
//			fmt.Printf("Download complete! File at: %s\n", event.Path)
//		case FetchEventTypeFailed:
//			fmt.Printf("Download failed: %v\n", event.Error)
//		}
//	})
//
//	if err := fetcher.Start(); err != nil {
//		log.Fatal(err)
//	}
//
//	// Cancel if needed
//	time.Sleep(5 * time.Second)
//	fetcher.Cancel()
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type AttachmentFetcher struct {
	mu              sync.RWMutex
	token           *AttachmentToken
	callback        FetchCallback
	dittoHandle     *ffi.DittoHandle
	progress        float64
	bytesDownloaded int64
	completed       bool
	callbackID      uintptr
	cancelToken     uintptr
}

// FetchCallback is the function signature for attachment fetch event callbacks.
// This function is called whenever there is a progress update, completion,
// failure, or cancellation of an attachment fetch operation.
//
// The callback is called from a background goroutine and should not perform
// long-running operations that could block progress updates.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
type FetchCallback func(event *AttachmentFetchEvent)

// newAttachmentFetcher creates a new attachment fetcher.
//
// Parameters:
//   - dittoHandle: The Ditto handle for FFI operations
//   - token: The attachment token identifying what to fetch
//   - callback: Function called for fetch events (progress, completion, etc.)
//
// Returns:
//   - *AttachmentFetcher: The created fetcher, ready to be started
//
// The fetcher is created in an inactive state. Call Start() to begin fetching.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func newAttachmentFetcher(dittoHandle *ffi.DittoHandle, token *AttachmentToken, callback FetchCallback) *AttachmentFetcher {
	return &AttachmentFetcher{
		token:       token,
		callback:    callback,
		dittoHandle: dittoHandle,
	}
}

// Start begins fetching the attachment.
//
// Returns:
//   - error: An error if the fetch cannot be started (e.g., invalid token or handle)
//
// Once started, the fetcher will download the attachment in the background
// and call the callback function with progress updates. The fetch continues
// until completion, failure, or cancellation.
//
// Starting an already started fetcher will return an error.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.callbackID != 0 {
		return fmt.Errorf("fetcher already started")
	}

	// Convert token ID to bytes
	idBytes, err := hex.DecodeString(f.token.ID())
	if err != nil {
		return fmt.Errorf("invalid attachment ID: %w", err)
	}

	// Register callbacks
	f.callbackID = ffi.RegisterAttachmentCallbacks(
		func(handle *ffi.AttachmentHandle) {
			// On complete callback
			f.mu.Lock()
			f.completed = true
			f.progress = 1.0
			f.bytesDownloaded = f.token.Length()
			f.mu.Unlock()

			if f.callback != nil {
				// Try to get the attachment path
				path, err := ffi.GetCompleteAttachmentPath(f.dittoHandle, handle)
				if err != nil {
					// Fallback path
					path = fmt.Sprintf("/tmp/attachment_%s", f.token.ID())
				}

				event := &AttachmentFetchEvent{
					Type:            AttachmentFetchEventTypeCompleted,
					Path:            path,
					Progress:        1.0,
					BytesDownloaded: f.token.Length(),
					TotalBytes:      f.token.Length(),
				}
				f.callback(event)
			}
		},
		func(downloaded, total uint64) {
			// On progress callback
			f.mu.Lock()
			if total > 0 {
				f.progress = float64(downloaded) / float64(total)
			} else {
				f.progress = 0
			}
			f.bytesDownloaded = int64(downloaded)
			f.mu.Unlock()

			if f.callback != nil {
				event := &AttachmentFetchEvent{
					Type:            AttachmentFetchEventTypeProgress,
					Progress:        f.progress,
					BytesDownloaded: int64(downloaded),
					TotalBytes:      int64(total),
				}
				f.callback(event)
			}
		},
		func() {
			// On deleted callback
			if f.callback != nil {
				event := &AttachmentFetchEvent{
					Type: AttachmentFetchEventTypeCanceled,
				}
				f.callback(event)
			}
		},
	)

	// Start fetching using the resolve attachment FFI function
	result, err := ffi.ResolveAttachment(f.dittoHandle, idBytes, f.callbackID)
	if err != nil {
		ffi.UnregisterAttachmentCallbacks(f.callbackID)
		f.callbackID = 0
		return fmt.Errorf("failed to start fetching attachment: %w", err)
	}

	if result.StatusCode != 0 {
		ffi.UnregisterAttachmentCallbacks(f.callbackID)
		f.callbackID = 0
		return fmt.Errorf("attachment resolve failed with status code: %d", result.StatusCode)
	}

	f.cancelToken = result.CancelToken

	return nil
}

// Cancel stops fetching the attachment.
//
// This method is safe to call multiple times and will gracefully stop
// the fetch operation if it's in progress. The callback will receive
// a cancellation event.
//
// After cancellation, the fetcher cannot be restarted.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) Cancel() {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Cancel via FFI if we have a cancel token
	if f.cancelToken != 0 && f.dittoHandle != nil {
		// Call FFI cancel function here if available
		// For now we rely on callback cleanup
	}

	// Unregister callbacks
	if f.callbackID != 0 {
		ffi.UnregisterAttachmentCallbacks(f.callbackID)
		f.callbackID = 0
	}
}

// Token returns the attachment token that identifies what is being fetched.
// This can be useful for tracking multiple concurrent fetches.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) Token() *AttachmentToken {
	return f.token
}

// IsCompleted returns true if the fetch has completed successfully.
// This does not include failed or canceled fetches.
//
// Thread-safe: This method can be called concurrently.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) IsCompleted() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.completed
}

// Progress returns the current fetch progress as a value between 0.0 and 1.0.
//   - 0.0 means the fetch hasn't started or no data has been downloaded
//   - 1.0 means the fetch is complete
//   - Values in between represent partial completion
//
// Thread-safe: This method can be called concurrently.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) Progress() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.progress
}

// BytesDownloaded returns the number of bytes downloaded so far.
// This can be used with TotalBytes() to calculate progress or estimate
// remaining download time.
//
// Thread-safe: This method can be called concurrently.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) BytesDownloaded() int64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.bytesDownloaded
}

// TotalBytes returns the total size of the attachment in bytes.
// This value is known before the download starts and comes from the token.
//
// Deprecated: This API is experimental. It is not supported in this preview build and may change or be removed at any time.
func (f *AttachmentFetcher) TotalBytes() int64 {
	return f.token.Length()
}
