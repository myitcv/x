package main

import (
	"fmt"
	"io"
)

func mainUsage(f io.Writer) {
	fmt.Fprint(f, mainHelp)
}

var mainHelp = `
The protoc command is a Go modules-based wrapper around the C++ protoc command.

Usage:
    protoc [-Ipkg pkg]... [-go-out options] protofile...

protoc also ensures, using gobin -m, that protoc-gen-go is available to the
underlying C++ protoc command. gobin is therefore assumed to be on PATH.

The -Ipkg flag takes a package path. The directory corresponding to the package
is passed to the underlying protoc as a -I value. The -Ipkg flag may be
repeated.

The -go-out flag is passed verbatim to the underlying protoc command. As
documented at
https://github.com/golang/protobuf#using-protocol-buffers-with-go, the -go-out
flag can be used to control the output directory for generated Go code.

protoc maintains a cache of C++ protoc installations and protoc-gen-go
binaries.  By default, protoc uses the directories
protoc-cache/$goos/$goarch/$version under your user cache directory. See the
documentation for os.UserCacheDir for OS-specific details on how to configure
its location. Setting PROTOCCACHE overrides the default.

The -silent flag does not exist in the underlying C++ protoc command. It allows
protoc to exit without error and without calling the underlying C++ protoc
command if any of the input files do not exist. This is particularly useful
when protoc is being used as a go:generate directive and the input file(s)
are the result of a generation step in another package.

`[1:]
