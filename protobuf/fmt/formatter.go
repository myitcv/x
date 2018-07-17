// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package fmt // import "myitcv.io/protobuf/fmt"

import (
	"fmt"
	"io"
	"strings"

	"myitcv.io/protobuf/ast"
	"myitcv.io/protobuf/parser"
)

type Formatter struct {
	Output io.Writer

	// TODO this is a bit gross - we can only be in one oneOf at any
	// point in time... seems hacky to store the state here (for indenting)
	indent int
	oneOf  *ast.Oneof
}

func (f *Formatter) Fmt(files []string, importPaths []string) {

	fs, err := parser.ParseFiles(files, importPaths)
	if err != nil {
		panic(err)
	}

	var fmtFiles []*ast.File

	for _, astFile := range fs.Files {
		for _, file := range files {
			if file == astFile.Name {
				fmtFiles = append(fmtFiles, astFile)
			}
		}
	}

	for _, file := range fmtFiles {
		f.FmtFile(file)
	}
}

func (f *Formatter) println(a ...interface{}) {
	fmt.Fprintf(f.Output, strings.Repeat("\t", f.indent))
	fmt.Fprintln(f.Output, a...)
}

func (f *Formatter) printf(format string, a ...interface{}) {
	fmt.Fprintf(f.Output, strings.Repeat("\t", f.indent)+format, a...)
}

func (f *Formatter) noIndentPrintf(format string, a ...interface{}) {
	fmt.Fprintf(f.Output, format, a...)
}
