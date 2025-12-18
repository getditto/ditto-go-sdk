package ffi

import (
	"runtime"
	"sync"
)

// freer abstracts a C pointer type to associate it with the C function that will free it.
type freer interface {
	free()
}

// HandleCleaner holds a C pointer type that needs to be freed by calling a C function.
// The inner value must be set by calling Initialize, which will add a runtime.Cleanup to ensure the
// C pointer is freed when the HandleCleaner is garbage collected.
// See DittoHandle for an example of its use.
type HandleCleaner[H freer] struct {
	inner    H
	cleanup  runtime.Cleanup
	freeOnce sync.Once
}

// Initialize sets the inner pointer to value, and registers a cleanup function with the runtime to
// free the pointer when the HandleCleaner is garbage collected.
func (c *HandleCleaner[H]) Initialize(value H) {
	c.inner = value
	// When 'c' is garbage collected, the runtime will invoke `H.free(value)` (aka `value.free()`).
	// Note that the argument to the cleanup function must not be the same as the pointer being cleaned (c) because
	// that reference held by the runtime would prevent c from becoming unreachable. See AddCleanup for more details.
	//
	// While we cannot have the runtime call c.Free() to use freeOnce and ensure value.free() is only called once,
	// the Cleanup itself will only be called once, when c is garbage collected. Calling cleanup.Stop() in Free()
	// removes the Cleanup and preserves the guarantee that value.free() is only called once.
	c.cleanup = runtime.AddCleanup(c, H.free, value)
}

// Free explicitly frees the C pointer by invoking the inner pointer's freer.
// Free may be called multiple times, but the freer will only be called once.
// After Free, the inner pointer is set to nil.
func (c *HandleCleaner[H]) Free() {
	c.freeOnce.Do(func() {
		c.cleanup.Stop()
		c.inner.free()
		var zero H
		c.inner = zero
	})
}
