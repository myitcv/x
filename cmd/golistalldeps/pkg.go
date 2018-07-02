package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os/exec"
	"strings"
)

type Package struct {
	Dir           string // directory containing package sources
	ImportPath    string // import path of package in dir
	ImportComment string // path in import comment on package statement
	Name          string // package name
	Doc           string // package documentation string
	Target        string // install path
	Shlib         string // the shared library that contains this package (only set when -linkshared)
	Goroot        bool   // is this package in the Go root?
	Standard      bool   // is this package part of the standard Go library?
	Stale         bool   // would 'go install' do anything for this package?
	StaleReason   string // explanation for Stale==true
	Root          string // Go root or Go path dir containing this package
	ConflictDir   string // this directory shadows Dir in $GOPATH
	BinaryOnly    bool   // binary-only package: cannot be recompiled from sources

	// Source files
	GoFiles        []string // .go source files (excluding CgoFiles, TestGoFiles, XTestGoFiles)
	CgoFiles       []string // .go sources files that import "C"
	IgnoredGoFiles []string // .go sources ignored due to build constraints
	CFiles         []string // .c source files
	CXXFiles       []string // .cc, .cxx and .cpp source files
	MFiles         []string // .m source files
	HFiles         []string // .h, .hh, .hpp and .hxx source files
	FFiles         []string // .f, .F, .for and .f90 Fortran source files
	SFiles         []string // .s source files
	SwigFiles      []string // .swig files
	SwigCXXFiles   []string // .swigcxx files
	SysoFiles      []string // .syso object files to add to archive

	// Cgo directives
	CgoCFLAGS    []string // cgo: flags for C compiler
	CgoCPPFLAGS  []string // cgo: flags for C preprocessor
	CgoCXXFLAGS  []string // cgo: flags for C++ compiler
	CgoFFLAGS    []string // cgo: flags for Fortran compiler
	CgoLDFLAGS   []string // cgo: flags for linker
	CgoPkgConfig []string // cgo: pkg-config names

	// Dependency information
	Imports []string // import paths used by this package
	Deps    []string // all (recursively) imported dependencies

	// Error information
	Incomplete bool            // this package or a dependency has an error
	Error      *PackageError   // error loading package
	DepsErrors []*PackageError // errors loading dependencies

	TestGoFiles  []string // _test.go files in package
	TestImports  []string // imports from TestGoFiles
	XTestGoFiles []string // _test.go files outside package
	XTestImports []string // imports from XTestGoFiles
}

type PackageError struct {
	ImportStack []string // shortest path from package named on command line to this one
	Pos         string   // position of error (if present, file:line:col)
	Err         string   // the error itself
}

func goList(spec []string) []*Package {
	if len(spec) == 0 {
		return nil
	}

	var res []*Package
	args := append([]string{"list", "-e", "-json"}, spec...)

	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("running go %v\n%v", strings.Join(args, " "), string(out))
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p Package
		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("reading go list output: %v", err)
		}
		pkgInfo[p.ImportPath] = &p
		res = append(res, &p)
	}

	return res
}
