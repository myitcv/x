package main

import (
	"fmt"
	"io"
)

func mainUsage(f io.Writer) {
	fmt.Fprint(f, mainHelp)
}

var mainHelp = `
getprotoc is a go:generate generator that downloads a given version of the C++
protoc compiler.

Usage:
	getprotoc version

getprotoc takes a single argument: the version of protoc to fetch. GOOS and
GOARCH are used to determin the OS and arch. Downloads are placed in
$PWD/downloads/$GOOS/$GOARCH/$version.zip. You should probably .gitignore the
$PWD/downloads directory.

getprotoc is useless by itself; it should generally be followed by a go-bindata
directive that "vendors" a specific zip file.

`[1:]
