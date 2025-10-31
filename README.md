# Prerequisites

- A Go installation. The SDK has been tested with Go versions: 1.24 - 1.25

# Creating a New Project

```
mkdir my_project_name; cd my_project_name
go mod init example.com/my_project_name
```

## Adding the ditto dependency


Include a `replace` directive in your `go.mod` to locate the Ditto SDK:

```
$ go mod edit -replace github.com/getditto/ditto-go-sdk=path/to/ditto-go-sdk
```

Or edit `go.mod` by hand:

```
replace github.com/getditto/ditto-go-sdk => path/to/ditto-go-sdk
```

Now `go get` the SDK to add it to `go.mod`. It will use the replacement path.

```
$ go get github.com/getditto/ditto-go-sdk
```

Or edit `go.mod` by hand:

```
require github.com/getditto/ditto-go-sdk v0.0.0
```

After importing `github.com/getditto/ditto-go-sdk/ditto` in your source code, run `go mod tidy` to ensure all indirect dependencies are properly required.

## CGo

The Ditto SDK for Go uses CGo. The Go toolchain enables CGo by default, but your project must
not be built with CGo disabled, such as with `CGO_ENABLED=0`.

## Library Linking

The Ditto SDK for Go uses static linking by default on all platforms, creating self-contained binaries.

### Default Behavior (Static Linking)

On macOS and Linux, the Ditto FFI library is statically linked into your binary:
- No additional files need to be distributed with your application
- The binary is self-contained and portable
- Works with `go run`, `go test`, and `go build` without additional configuration

Simply build and run your application:
```bash
$ go build -o my_binary .
$ ./my_binary
```

### Dynamic Linking Option (Linux Only)

For special use cases, Linux also supports dynamic linking with `libdittoffi.so`.
To use dynamic linking on Linux, you would need to:
1. Modify `internal/ffi/bindings.go` to use dynamic linking directives
2. Set `LD_LIBRARY_PATH` for `go run` and `go test`
3. Distribute `libdittoffi.so` with your application

Note: Static linking is recommended for most use cases as it simplifies deployment.
