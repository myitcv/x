package main

import (
	"fmt"
	"io"
)

func mainUsage(f io.Writer) {
	fmt.Fprint(f, mainHelp)
}

var mainHelp = `
consttofile expands a package-level string constant to a file.

consttofile is best used via go generate directives in a main package:

  //go:generate consttofile myconst_txt

Running go generate (or any other program that understands go:generate
directives) will result in a single file, gen_myconst_consttofile.txt, that
contains the value of the string constant myconst_txt.
`[1:]
