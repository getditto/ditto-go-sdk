package ditto

import (
	"github.com/getditto/ditto-go-sdk/v5/internal/ffi"
)

// AuthenticationProvider encapsulates the authentication provider used to login.
//
// To define a custom authentication provider, wrap the string like this:
//
//	mySpecialProvider := ditto.AuthenticationProvider("my-special-provider")
type AuthenticationProvider string

// DevelopmentAuthenticationProvider returns the built-in development authentication provider
// to be used together with development authentication tokens.
func DevelopmentAuthenticationProvider() AuthenticationProvider {
	providerName := ffi.GetDevelopmentProvider()
	return AuthenticationProvider(providerName)
}

// String returns the raw underlying authentication provider name as a string.
//
// This method allows AuthenticationProvider to be used directly as a string
// in contexts that expect a provider name.
func (p AuthenticationProvider) String() string { return string(p) }
