// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

// Package gogenerate exposes some of the unexported internals of the go generate command as a convenience
// for the authors of go generate generators. See https://github.com/myitcv/gogenerate/wiki/Go-Generate-Notes
// for further notes on such generators. It also exposes some convenience functions that might be useful
// to authors of generators
//
package gogenerate

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rogpeppe/go-internal/imports"
)

// These constants correspond in name and value to the details given in
// go generate --help
const (
	GOARCH    = "GOARCH"
	GOFILE    = "GOFILE"
	GOLINE    = "GOLINE"
	GOOS      = "GOOS"
	GOPACKAGE = "GOPACKAGE"
	GOPATH    = "GOPATH"

	GoGeneratePrefix = "//go:generate"
)

const (
	// genStr is the string used in the prefix of generated files
	genStr = "gen"

	// sep is the separator used between the parts of a file name; the prefix used to identify
	// generated files, the name (body) and the suffix used to identify the generator
	sep = "_"

	// genFilePrefix is the prefix used on all generated files (which strictly speaking
	// is limited to Go files as far as this definition is concerned, but in practice need not be)
	genFilePrefix = genStr + sep
)

const (
	// FlagLog is the name of the common flag shared between go generate generators
	// to control logging verbosity.
	FlagLog = "gglog"

	// FlagOutDirPrefix is the prefix used for flags generated by OutPkgFlag.
	FlagOutDirPrefix = "outdir:"

	// FlagLicenseFile is the name of the common flag shared between go generate generators
	// to provide a license header file.
	FlagLicenseFile = "licenseFile"
)

type LogLevel string

// The various log levels supported by the flag returned by LogFlag()
const (
	LogInfo    LogLevel = "info"
	LogWarning LogLevel = "warning"
	LogError   LogLevel = "error"
	LogFatal   LogLevel = "fatal"
)

// FileIsGenerated determines wheter the Go file located at path is generated or not
// and if it is generated returns the base name of the generator that generated it
func FileIsGenerated(path string) (string, bool) {
	gen, ext, isGen := AnyFileIsGenerated(path)
	if isGen && ext == ".go" {
		return gen, true
	}
	return "", false
}

// AnyFileIsGenerated determines wheter the file located at path is generated or
// not and if it is generated returns the base name of the generator that
// generated it and the file extension
func AnyFileIsGenerated(path string) (string, string, bool) {
	fn := filepath.Base(path)
	extI := strings.LastIndex(fn, ".")
	if extI < 0 {
		return "", "", false
	}
	fn, ext := fn[:extI], fn[extI:]

	if !strings.HasPrefix(fn, genFilePrefix) {
		return "", ext, false
	}

	fn = strings.TrimPrefix(fn, genFilePrefix)

	if strings.HasSuffix(fn, "_test") {
		fn = strings.TrimSuffix(fn, "_test")
	}

	// deals with the edge case gen_.go or gen__test.go
	if fn == "" {
		return "", ext, false
	}

	parts := strings.Split(fn, sep)

	return parts[len(parts)-1], ext, true
}

// AnyFileGeneratedBy returns true if the base name of the supplied path is a
// file that would have been generated by the supplied cmd. Unlike
// FileGeneratedBy this is not limited to .go files. The extension of the file
// is also returned
func AnyFileGeneratedBy(path string, cmd string) bool {
	cmd = filepath.Base(cmd)

	c, _, ok := AnyFileIsGenerated(path)

	return ok && c == cmd
}

// FileGeneratedBy returns true if the base name of the supplied path is a Go
// file that would have been generated by the supplied cmd
func FileGeneratedBy(path string, cmd string) bool {
	cmd = filepath.Base(cmd)

	c, ok := FileIsGenerated(path)

	return ok && c == cmd
}

// NameFileFromFile uses the provided filename as a template and returns a generated filename consistent with
// the provided command
func NameFileFromFile(name string, cmd string) (string, bool) {
	dir := filepath.Dir(name)
	name = filepath.Base(name)

	if !strings.HasSuffix(name, ".go") {
		return "", false
	}

	name = strings.TrimSuffix(name, ".go")
	cmd = filepath.Base(cmd)

	var res string

	if strings.HasSuffix(name, "_test") {
		name = strings.TrimSuffix(name, "_test")
		res = NameTestFile(name, cmd)
	} else {
		res = NameFile(name, cmd)
	}

	return filepath.Join(dir, res), true
}

func nameBase(name string, cmd string) string {
	res := genStr

	if name != "" {
		res += sep + name
	}

	res += sep + cmd

	return res
}

// NameFile returns a file name that conforms with the pattern associated with
// files generated by the provided command
func NameFile(name string, cmd string) string {
	cmd = filepath.Base(cmd)

	return nameBase(name, cmd) + ".go"
}

// NameTestFile returns a file name that conforms with the pattern associated
// with files generated by the provided command
func NameTestFile(name string, cmd string) string {
	cmd = filepath.Base(cmd)

	return nameBase(name, cmd) + "_test.go"
}

type outputs []string

func (o *outputs) String() string {
	return fmt.Sprint(*o)
}

func (o *outputs) Set(value string) error {
	*o = append(*o, value)
	return nil
}

var outputsSet = make(map[string]bool)

// OutPkgFlag defines a new flag "outpkg:"+key that can accept a list of
// package specifications that represent output targets above and beyond the
// default of self.
func OutPkgFlag(key string) *outputs {
	if outputsSet[key] {
		panic(fmt.Errorf("already defined outpkg flag for key %q", key))
	}

	// safe doing this because we should be in init phase
	outputsSet[key] = true

	var res outputs

	flag.Var(&res, "outpkg:"+key, "list of output packages in addition to self")

	return &res
}

// LogFlag defines a command line string flag named according to the constant
// FlagLog and returns a pointer to the string the flag sets
func LogFlag() *string {
	return flag.String(FlagLog, string(LogFatal), "log level; one of info, warning, error, fatal")
}

// LicenseFileFlag defines a command line string flag named according to the
// constant FlagLicenseFile and returns a pointer ot the string that flag set
func LicenseFileFlag() *string {
	return flag.String(FlagLicenseFile, "", "file that contains a license header to be inserted at the top of each generated file")
}

// CommentLicenseHeader is a convenience function to be used in conjunction
// with LicenseFileFlag; if a filename is provided it reads the contents of the
// file and returns a line-commented transformation of the contents with a
// final blank newline
func CommentLicenseHeader(file *string) (string, error) {
	if file == nil || *file == "" {
		return "", nil
	}

	fi, err := os.Open(*file)
	if err != nil {
		return "", fmt.Errorf("could not open file %q: %v", *file, err)
	}

	res := bytes.NewBuffer(nil)

	lastLineEmpty := false
	scanner := bufio.NewScanner(fi)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lastLineEmpty = line == ""
		fmt.Fprintln(res, "//", line)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan file %v: %v", *file, err)
	}

	// ensure we have a space before package
	if !lastLineEmpty {
		fmt.Fprintln(res)
	}

	return res.String(), nil
}

// DefaultLogLevel is provided simply as a convenience along with LogFlag to ensure a default LogLevel
// in a flag variable. This is necessary because of the interplay between go generate argument parsing
// and the advice given for log levels via gg.
func DefaultLogLevel(f *string, ll LogLevel) {
	if f != nil && *f == "" {
		*f = string(ll)
	}
}

// FilesContainingCmd returns a map of Go file name (defined by go list as
// GoFiles + CgoFiles + TestGoFiles + XTestGoFiles) in the directory dir, to a
// count of the number of times directive command appears in that file (after
// quote and variable expansion as described by go generate -help). Commands
// can be gobin commands or plain PATH-based command calls. When comparing
// PATH-based commands, the filepath.Base of each is compared. The file names
// will, by definition, be relative to dir
func FilesContainingCmd(dir string, command string, tags map[string]bool) (map[string]int, error) {
	pkgs, err := getCandidateFiles(dir, tags)
	if err != nil {
		return nil, err
	}

	command = strings.TrimSpace(command)
	matches := map[string]int{}
	cmdstr := filepath.Base(filepath.FromSlash(command))
	cmdIsPathBased := string(os.PathSeparator) == "/" || strings.Index(command, string(os.PathSeparator)) == -1

	for pname, files := range pkgs {
		for _, f := range files {
			fname := filepath.Base(f)
			checkMatch := func(line int, args []string) error {
				gencmd := filepath.Base(args[0])
				if gencmd == "gobin" && cmdIsPathBased {
					// NOTE: We want to only deal with gobin cmd for now

					// NOTE: Create flagset similar to the gobin cmd to parse all flags and consume
					// output of Args() to determine if the import command is a part of the args
					fs := flag.NewFlagSet("gobin", 0)
					fs.Bool("m", false, "resolve dependencies via the main module (as given by go env GOMOD)")
					fs.String("mod", "", "provide additional control over updating and use of go.mod")
					frun := fs.Bool("run", false, "run the provided main package")
					fs.Bool("p", false, "print gobin install cache location for main packages")
					fs.Bool("v", false, "print the module path and version for main packages")
					fs.Bool("d", false, "stop after installing main packages to the gobin install cache")
					fs.Bool("u", false, "check for the latest tagged version of main packages")
					fs.Bool("nonet", false, "prevent network access")
					fs.Bool("debug", false, "print debug information")

					err := fs.Parse(args[1:])
					if err != nil {
						return fmt.Errorf("unable to parse gobin flags: %v", err)
					}

					patterns := fs.Args()[0]
					cmdPath := strings.Split(patterns, "@")[0]
					if *frun && cmdPath == command {
						matches[fname] += 1
					}

				} else if gencmd == cmdstr {
					matches[fname] += 1
				}

				return nil
			}

			err := DirFunc(pname, dir, fname, checkMatch)
			if err != nil {
				return nil, err
			}
		}
	}

	return matches, nil
}

func getCandidateFiles(dir string, tags map[string]bool) (map[string][]string, error) {
	packages := make(map[string][]string, 0)

	filterFn := func(fi os.FileInfo) bool {
		return imports.MatchFile(fi.Name(), tags)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, filterFn, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("could not run ParseDir: %v", err)
	}

	for _, p := range pkgs {
		files := []string{}
		for name, f := range p.Files {
			var comments bytes.Buffer
			for _, cg := range f.Comments {
				if cg == f.Doc || cg.Pos() > f.Package {
					break
				}

				for _, cm := range cg.List {
					comments.WriteString(cm.Text + "\n")
				}

				comments.WriteString("\n")
			}

			if imports.ShouldBuild(comments.Bytes(), tags) {
				files = append(files, name)
			}
		}

		if len(files) > 0 {
			sort.SliceStable(files, func(i, j int) bool { return files[i] < files[j] })
			packages[p.Name] = files
		}
	}

	return packages, nil
}
