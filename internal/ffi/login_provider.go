// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include "dittoffi.h"
#include <stdlib.h>

// Forward declare the C callbacks
extern void goLoginProviderRetain(void* ctx);
extern void goLoginProviderRelease(void* ctx);
extern void goLoginProviderExpiringCallback(void* ctx, uint32_t secondsUntilExpiration);

// Create a LoginProvider (using static inline to avoid duplicate symbols)
static inline CLoginProvider_t* createGoLoginProvider(void* ctx) {
    return ditto_auth_client_make_login_provider(
        ctx,
        goLoginProviderRetain,
        goLoginProviderRelease,
        goLoginProviderExpiringCallback
    );
}
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// LoginProviderCallback is the type for login provider expiration callbacks
type LoginProviderCallback func(secondsUntilExpiration uint32)

// providerContext stores the callback and reference count for a provider
type providerContext struct {
	callback LoginProviderCallback
	refCount int
	id       uintptr
}

var (
	loginProviderMu     sync.RWMutex
	loginProviders              = make(map[uintptr]*providerContext)
	nextLoginProviderID uintptr = 1
)

//export goLoginProviderRetain
func goLoginProviderRetain(ctx unsafe.Pointer) {
	loginProviderMu.Lock()
	defer loginProviderMu.Unlock()

	id := uintptr(ctx)
	if pctx, ok := loginProviders[id]; ok {
		pctx.refCount++
	}
}

//export goLoginProviderRelease
func goLoginProviderRelease(ctx unsafe.Pointer) {
	loginProviderMu.Lock()
	defer loginProviderMu.Unlock()

	id := uintptr(ctx)
	if pctx, ok := loginProviders[id]; ok {
		pctx.refCount--
		if pctx.refCount <= 0 {
			delete(loginProviders, id)
		}
	}
}

//export goLoginProviderExpiringCallback
func goLoginProviderExpiringCallback(ctx unsafe.Pointer, secondsUntilExpiration C.uint32_t) {
	loginProviderMu.RLock()
	id := uintptr(ctx)
	var callback LoginProviderCallback
	if pctx, ok := loginProviders[id]; ok {
		callback = pctx.callback
	}
	loginProviderMu.RUnlock()

	if callback != nil {
		callback(uint32(secondsUntilExpiration))
	}
}

// setLoginProviderInternal is an internal function to break CGO pointer checking
//
//go:noinline
func setLoginProviderInternal(id uintptr) *C.CLoginProvider_t {
	return C.createGoLoginProvider(unsafe.Pointer(id))
}

// SetLoginProvider sets the login provider for authentication
func SetLoginProvider(handle *DittoHandle, callback LoginProviderCallback) error {
	loginProviderMu.Lock()
	id := nextLoginProviderID
	nextLoginProviderID++

	// Create and store provider context
	pctx := &providerContext{
		callback: callback,
		refCount: 1, // Initial reference count
		id:       id,
	}
	loginProviders[id] = pctx
	loginProviderMu.Unlock()

	// Use separate function to call C code
	provider := setLoginProviderInternal(id)
	if provider == nil {
		// Clean up on failure
		loginProviderMu.Lock()
		delete(loginProviders, id)
		loginProviderMu.Unlock()
		return fmt.Errorf("failed to create login provider")
	}

	// Set the login provider
	C.ditto_auth_set_login_provider(handle.load(), provider)

	// Important: After setting the login provider, the Ditto core takes ownership
	// and will manage the lifecycle. We should not manually free it here as it
	// would be freed by the FFI layer when appropriate.

	return nil
}
