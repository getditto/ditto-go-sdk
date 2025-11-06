// Copyright 2025 DittoLive Incorporated. All rights reserved.

package ffi

/*
#include "dittoffi.h"
*/
import "C"
import "runtime"

// Platform constants from FFI header
const (
	PlatformWindows = C.PLATFORM_WINDOWS
	PlatformMac     = C.PLATFORM_MAC
	PlatformIOS     = C.PLATFORM_IOS
	PlatformTvOS    = C.PLATFORM_TVOS
	PlatformAndroid = C.PLATFORM_ANDROID
	PlatformLinux   = C.PLATFORM_LINUX
	PlatformWeb     = C.PLATFORM_WEB
	PlatformUnknown = C.PLATFORM_UNKNOWN
)

// Language constants from FFI header
const (
	LanguageSwift      = C.LANGUAGE_SWIFT
	LanguageObjectiveC = C.LANGUAGE_OBJECTIVE_C
	LanguageCPlusPlus  = C.LANGUAGE_C_PLUS_PLUS
	LanguageCSharp     = C.LANGUAGE_C_SHARP
	LanguageJavaScript = C.LANGUAGE_JAVA_SCRIPT
	LanguageUnknown    = C.LANGUAGE_UNKNOWN
	LanguageRust       = C.LANGUAGE_RUST
	LanguageKotlin     = C.LANGUAGE_KOTLIN
	LanguageJava       = C.LANGUAGE_JAVA
	LanguageFlutter    = C.LANGUAGE_FLUTTER
	LanguageGo         = C.LANGUAGE_GO
)

// DetectPlatform returns the appropriate platform constant based on runtime.GOOS
func DetectPlatform() C.Platform_t {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMac
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	case "android":
		return PlatformAndroid
	case "ios":
		return PlatformIOS
	default:
		return PlatformUnknown
	}
}

// GetLanguage returns the FFI language constant for Go
func GetLanguage() C.Language_t {
	return LanguageGo
}
