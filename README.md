**NOTE** The **Public Preview** of the Ditto Go SDK can be used for exploratory development and prototypes. Refer to the documentation for supported features. Future versions of the Ditto Go SDK may not be compatible with the current preview API.


## Adding the Ditto Go SDK to a Go Application

In the directory containing your application's `go.mod` file, run these commands to add the Ditto Go SDK and its dependencies:

```shell
go get github.com/getditto/ditto-go-sdk/v5@latest
go mod tidy
```

Then, in any Go source file that uses the Ditto SDK, add this import:

```go
import "github.com/getditto/ditto-go-sdk/v5/ditto"
```

API documentation for the Go SDK can be found at [https://software.ditto.live/go/Ditto/5.0.0/docs/github.com/getditto/ditto-go-sdk/v5/ditto/index.html](https://software.ditto.live/go/Ditto/5.0.0/docs/github.com/getditto/ditto-go-sdk/v5/ditto/index.html).


## Installing the Ditto Go SDK Native Library

A native library provides Ditto Edge Sync Platform functionality, and must be downloaded and linked into any application that uses the Ditto Go SDK.


### Downloading and Unpacking the Ditto Library

Download `Ditto.tar.gz` and unpack an archive containing the `libdittoffi.so` (Linux) or `libdittoffi.dylib` (macOS) shared library for your operating system and CPU architecture.

#### Linux x86-64

```shell
curl -O https://software.ditto.live/go/Ditto/5.0.0/dist/libdittoffi-linux-x86_64.tar.gz && tar xvfz libdittoffi-linux-x86_64.tar.gz
```

#### Linux aarch64

```shell
curl -O https://software.ditto.live/go/Ditto/5.0.0/dist/libdittoffi-linux-aarch64.tar.gz && tar xvfz libdittoffi-linux-aarch64.tar.gz
```

#### macOS aarch64

```shell
curl -O https://software.ditto.live/go/Ditto/5.0.0/dist/libdittoffi-macos-aarch64.tar.gz && tar xvfz libdittoffi-macos-aarch64.tar.gz
```


### Linking to the Ditto Shared Library at Build Time

When you build your application that uses the Ditto Go SDK, the linker needs to be able to find the native library.

#### Linux

The shared `libdittoffi.so` library will automatically be linked to by the Go SDK if it is in one of these locations:

- `/usr/local/lib`
- `/usr/lib`

Copy the unpacked `libdittoffi.so` to one of these locations for automatic linking.

If your `libdittoffi.so` is located somewhere else, you can pass that path to `go build` via the `-ldflags` option when you build your application.

```shell
go build -ldflags='-extldflags "-L/path/to/my/lib/directory"'
```

#### macOS

The shared `libdittoffi.dylib` library will automatically be linked to by the Go SDK if it is in one of these locations:

- `/usr/local/lib`
- `/usr/lib`

Copy the unpacked `libdittoffi.dylib` to one of these locations for automatic linking.

If your `libdittoffi.dylib` is located somewhere else, you can pass that path to `go build` via the `-ldflags` option when you build your application.

```shell
go build -ldflags='-extldflags "-L/path/to/my/lib/directory"'
```


### Linking to the Ditto Shared Library at Run Time

When an app that uses the Ditto Go SDK is run, the system must be able to find and load the native library.

#### Linux

Ubuntu/Debian will search in these directories at runtime by default:

- `/lib`
- `/lib64`
- `/usr/lib`
- `/usr/lib64`
- `/usr/local/lib`

If you don't deploy the `libdittoffi.so` library in one of those directories, set the `LD_LIBRARY_PATH` environment variable to the appropriate directory when running your application. For example:

```shell
LD_LIBRARY_PATH=/path/to/my/lib/directory ./myapp
```

or

```shell
export LD_LIBRARY_PATH=/path/to/my/lib/directory
./myapp
```


#### macOS

macOS will search in these directories at runtime by default:

- `/usr/lib`
- `/usr/local/lib`

If you don't deploy the `libdittoffi.dylib` library in one of those directories, set the `DYLD_LIBRARY_PATH` environment variable to the appropriate directory when running your application. For example:

```shell
DYLD_LIBRARY_PATH=/path/to/my/lib/directory ./myapp
```

or

```shell
export DYLD_LIBRARY_PATH=/path/to/my/lib/directory
./myapp
```
