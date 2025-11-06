// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include "dittoffi.h"
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// AuthClientLoginWithToken logs in with a token
func AuthClientLoginWithToken(handle *DittoHandle, token string, provider string) error {
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))

	cProvider := C.CString(provider)
	defer C.free(unsafe.Pointer(cProvider))

	result := C.ditto_auth_client_login_with_token(handle.load(), cToken, cProvider)
	if result != 0 {
		return fmt.Errorf("login failed with error code %d", result)
	}

	return nil
}

// AuthClientLoginWithCredentials logs in with username and password
func AuthClientLoginWithCredentials(handle *DittoHandle, username, password, provider string) error {
	cUsername := C.CString(username)
	defer C.free(unsafe.Pointer(cUsername))

	cPassword := C.CString(password)
	defer C.free(unsafe.Pointer(cPassword))

	cProvider := C.CString(provider)
	defer C.free(unsafe.Pointer(cProvider))

	result := C.ditto_auth_client_login_with_credentials(handle.load(), cUsername, cPassword, cProvider)
	if result != 0 {
		return fmt.Errorf("login with credentials failed with error code %d", result)
	}

	return nil
}

// AuthClientLogout logs out the current user
func AuthClientLogout(handle *DittoHandle) error {
	result := C.ditto_auth_client_logout(handle.load())
	if result != 0 {
		return fmt.Errorf("logout failed with error code %d", result)
	}

	return nil
}

// AuthClientIsWebValid checks if web authentication is valid
func AuthClientIsWebValid(handle *DittoHandle) bool {
	result := C.ditto_auth_client_is_web_valid(handle.load())
	return result != 0
}

// AuthClientGetAppID gets the app ID
func AuthClientGetAppID(handle *DittoHandle) string {
	cAppID := C.ditto_auth_client_get_app_id(handle.load())
	if cAppID == nil {
		return ""
	}
	defer C.ditto_c_string_free(cAppID)

	return C.GoString(cAppID)
}

// AuthClientGetSiteID gets the site ID
func AuthClientGetSiteID(handle *DittoHandle) uint64 {
	return uint64(C.ditto_auth_client_get_site_id(handle.load()))
}

// GetDevelopmentProvider returns the name of the development authentication provider
func GetDevelopmentProvider() string {
	cProvider := C.dittoffi_DITTO_DEVELOPMENT_PROVIDER()
	if cProvider == nil {
		return ""
	}
	// The string is static, so we don't need to free it
	return C.GoString(cProvider)
}

// AuthClientLoginWithTokenAndFeedback logs in with a token and returns client info
func AuthClientLoginWithTokenAndFeedback(handle *DittoHandle, token string, provider string) (string, error) {
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))

	cProvider := C.CString(provider)
	defer C.free(unsafe.Pointer(cProvider))

	result := C.ditto_auth_client_login_with_token_and_feedback(handle.load(), cToken, cProvider)

	// Check if there was an error
	if result.status_code != 0 {
		return "", fmt.Errorf("login failed with error code %d", result.status_code)
	}

	// Get the client info JSON string
	if result.c_string == nil {
		return "", nil
	}

	clientInfo := C.GoString(result.c_string)
	C.ditto_c_string_free(result.c_string)

	return clientInfo, nil
}

// AuthClientGetUserID gets the authenticated user ID
func AuthClientGetUserID(handle *DittoHandle) string {
	cUserID := C.ditto_auth_client_user_id(handle.load())
	if cUserID == nil {
		return ""
	}
	defer C.ditto_c_string_free(cUserID)

	return C.GoString(cUserID)
}
