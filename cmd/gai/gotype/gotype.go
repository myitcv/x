// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main // import "myitcv.io/cmd/gai/gotype"

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

var (
	// main operation modes
	allFiles  = flag.Bool("a", false, "use all (incl. _test.go) files when processing a directory")
	allErrors = flag.Bool("e", false, "report all errors (not just the first 10)")
	verbose   = flag.Bool("v", false, "verbose mode")
	gccgo     = flag.Bool("gccgo", false, "use gccgoimporter instead of gcimporter")

	// debugging support
	sequential    = flag.Bool("seq", false, "parse sequentially, rather than in parallel")
	printAST      = flag.Bool("ast", false, "print AST (forces -seq)")
	printTrace    = flag.Bool("trace", false, "print parse trace (forces -seq)")
	parseComments = flag.Bool("comments", false, "parse comments (ignored unless -ast or -trace is provided)")
)

var (
	fset       = token.NewFileSet()
	errorCount = 0
	parserMode parser.Mode
	sizes      types.Sizes
)

func initParserMode() {
	if *allErrors {
		parserMode |= parser.AllErrors
	}
	if *printTrace {
		parserMode |= parser.Trace
	}
	if *parseComments && (*printAST || *printTrace) {
		parserMode |= parser.ParseComments
	}
}

func initSizes() {
	wordSize := 8
	maxAlign := 8
	switch build.Default.GOARCH {
	case "386", "arm":
		wordSize = 4
		maxAlign = 4
		// add more cases as needed
	}
	sizes = &types.StdSizes{WordSize: int64(wordSize), MaxAlign: int64(maxAlign)}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: gotype [flags] [path ...]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "In case no paths are provided, gotype reads from stdin.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "In case a single directory is provided, gotype processes files in that directory.")
	fmt.Fprintln(os.Stderr, "If the -a flag is provided gotype also processes the _test.go files in the directory.")
	fmt.Fprintln(os.Stderr, "Additionally, if the directory contains an xtest package, gotype separately processes")
	fmt.Fprintln(os.Stderr, "the xtest package (it assumes the package itself is installed).")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Otherwise, gotype assumes the paths are processed as files belonging to the same package.")
	fmt.Fprintln(os.Stderr, "")
	flag.PrintDefaults()
	os.Exit(2)
}

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	if list, ok := err.(scanner.ErrorList); ok {
		errorCount += len(list)
		return
	}
	errorCount++
}

// parse may be called concurrently
func parse(filename string, src interface{}) (*ast.File, error) {
	if *verbose {
		fmt.Println(filename)
	}
	file, err := parser.ParseFile(fset, filename, src, parserMode) // ok to access fset concurrently
	if *printAST {
		ast.Print(fset, file)
	}
	return file, err
}

func parseStdin() (*ast.File, error) {
	src, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return parse("<standard input>", src)
}

func parseFiles(filenames []string) ([]*ast.File, error) {
	files := make([]*ast.File, len(filenames))

	if *sequential {
		for i, filename := range filenames {
			var err error
			files[i], err = parse(filename, nil)
			if err != nil {
				return nil, err // leave unfinished goroutines hanging
			}
		}
	} else {
		type parseResult struct {
			file *ast.File
			err  error
		}

		out := make(chan parseResult)
		for _, filename := range filenames {
			go func(filename string) {
				file, err := parse(filename, nil)
				out <- parseResult{file, err}
			}(filename)
		}

		for i := range filenames {
			res := <-out
			if res.err != nil {
				return nil, res.err // leave unfinished goroutines hanging
			}
			files[i] = res.file
		}
	}

	return files, nil
}

func parseDir(dirname string) ([]*ast.File, []*ast.File, error) {
	ctxt := build.Default
	pkginfo, err := ctxt.ImportDir(dirname, 0)
	if _, nogo := err.(*build.NoGoError); err != nil && !nogo {
		return nil, nil, err
	}
	filenames := append(pkginfo.GoFiles, pkginfo.CgoFiles...)
	if *allFiles {
		filenames = append(filenames, pkginfo.TestGoFiles...)
	}

	xtestfilenames := pkginfo.XTestGoFiles

	// complete file names
	for i, filename := range filenames {
		filenames[i] = filepath.Join(dirname, filename)
	}

	for i, filename := range xtestfilenames {
		xtestfilenames[i] = filepath.Join(dirname, filename)
	}

	files, err := parseFiles(filenames)

	if err != nil {
		return nil, nil, err
	}

	xtestfiles, err := parseFiles(xtestfilenames)

	if err != nil {
		return nil, nil, err
	}

	return files, xtestfiles, nil
}

func getPkgFiles(args []string) ([]*ast.File, []*ast.File, error) {
	if len(args) == 0 {
		// stdin
		file, err := parseStdin()
		if err != nil {
			return nil, nil, err
		}
		return []*ast.File{file}, nil, nil
	}

	if len(args) == 1 {
		// possibly a directory
		path := args[0]
		info, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}
		if info.IsDir() {
			return parseDir(path)
		}
	}

	// list of files
	files, err := parseFiles(args)

	return files, nil, err
}

func checkPkgFiles(files []*ast.File) {
	compiler := "gc"
	if *gccgo {
		compiler = "gccgo"
	}
	type bailout struct{}
	conf := types.Config{
		FakeImportC: true,
		Error: func(err error) {
			if !*allErrors && errorCount >= 10 {
				panic(bailout{})
			}
			report(err)
		},
		Importer: importer.For(compiler, nil),
		Sizes:    sizes,
	}

	defer func() {
		switch p := recover().(type) {
		case nil, bailout:
			// normal return or early exit
		default:
			// re-panic
			panic(p)
		}
	}()

	const path = "pkg" // any non-empty string will do for now
	conf.Check(path, fset, files, nil)
}

func printStats(d time.Duration) {
	fileCount := 0
	lineCount := 0
	fset.Iterate(func(f *token.File) bool {
		fileCount++
		lineCount += f.LineCount()
		return true
	})

	fmt.Printf(
		"%s (%d files, %d lines, %d lines/s)\n",
		d, fileCount, lineCount, int64(float64(lineCount)/d.Seconds()),
	)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *printAST || *printTrace {
		*sequential = true
	}
	initParserMode()
	initSizes()

	start := time.Now()

	files, xtestfiles, err := getPkgFiles(flag.Args())
	if err != nil {
		report(err)
		os.Exit(2)
	}

	checkPkgFiles(files)
	if errorCount > 0 {
		os.Exit(2)
	}

	if len(xtestfiles) > 0 {
		fmt.Printf("checking xtest package...")
		checkPkgFiles(xtestfiles)
		if errorCount > 0 {
			os.Exit(2)
		}
	}

	if *verbose {
		printStats(time.Since(start))
	}
}
