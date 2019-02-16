package main

import (
	"fmt"
	"io"
)

func mainUsage(f io.Writer) {
	fmt.Fprint(f, mainHelp)
}

var mainHelp = `
helpflagtopkgdoc ensures that your package docs stay up to date with your
-help flag output.

helpflagtopkgdoc is best used via go generate directives in a main package:

  //go:generate helpflagtopkgdoc

Running go generate (or any other program that understands go:generate
directives) will result in a single file, gen_helpflagtopkgdoc.go, that
contains the output of -help as a package document.
`[1:]
