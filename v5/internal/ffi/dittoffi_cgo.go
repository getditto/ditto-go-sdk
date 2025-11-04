package ffi

// This file contains CGO configuration directives for linking against libdittoffi.
// These directives apply to the entire ffi package.

/*
#cgo CFLAGS: -I${SRCDIR}
#cgo darwin CFLAGS: -mmacosx-version-min=11.0
#cgo darwin LDFLAGS: -L${SRCDIR}/../../build -L/usr/local/lib -L/usr/lib -ldittoffi
#cgo darwin LDFLAGS: -lc++ -framework Security -framework CoreFoundation -framework SystemConfiguration -framework IOKit -mmacosx-version-min=11.0
#cgo linux LDFLAGS: -L${SRCDIR}/../../build -L/usr/local/lib -L/usr/lib -ldittoffi
#cgo linux LDFLAGS: -lstdc++ -lm -ldl -lpthread
#cgo windows LDFLAGS: -L${SRCDIR}/../../build -ldittoffi
#cgo windows LDFLAGS: -lws2_32 -luserenv -lntdll -lbcrypt
*/
import "C"
