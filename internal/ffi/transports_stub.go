//go:build darwin
// +build darwin

package ffi

/*
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include "dittoffi.h"

// Stub implementation for macOS/iOS transport initialization
// This function is called by ditto_sdk_transports_init on Apple platforms
bool ditto_transports_init(DittoSdkTransportsError_t *outError) {
    if (outError) {
        *outError = DITTO_SDK_TRANSPORTS_ERROR_NONE;
    }
    return true;
}

// Stub for BLE transport creation
void* ditto_transports_ble_create(void* ditto, DittoSdkTransportsError_t *outError) {
    if (outError) {
        *outError = DITTO_SDK_TRANSPORTS_ERROR_NONE;
    }
    // Return a dummy non-null pointer to indicate success
    return (void*)0x1;
}

// Stub for BLE transport destruction
bool ditto_transports_ble_destroy(void* ble, DittoSdkTransportsError_t *outError) {
    if (outError) {
        *outError = DITTO_SDK_TRANSPORTS_ERROR_NONE;
    }
    return true;
}
*/
import "C"
