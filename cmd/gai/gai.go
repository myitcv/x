// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/kisielk/gotool"
)

const (
	// goBuildPkgHeader is the line prefix used in the output of go install etc
	// the starts a block of line errors
	goBuildPkgHeader = "# "

	// maxGoType
	maxGoType = 50
)

var (
	fDebug = flag.Bool("v", false, "be very verbose about what gai is doing")
)

type multiFlag []string

func (i *multiFlag) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *multiFlag) String() string {
	return fmt.Sprint(*i)
}

var (
	fPkgs  multiFlag
	fTpkgs multiFlag
)

func init() {
	flag.Var(&fPkgs, "P", "a file containing package specs (may appear multiple times) that should be installed")
	flag.Var(&fTpkgs, "T", "a file containing package specs that should also, in addition to being installed,\n\talso have their tests type-checked (may appear multiple times)")
}

func usage() {
	fmt.Fprintf(os.Stderr, `
%v - install and type check packages and their tests

Usage:

  gai [-P <file1>] [-T <file2>] <package specs>...

  Package specs provided as arguments will be treated as if supplied via a -T flag.

Flags:

`, os.Args[0])
	flag.PrintDefaults()
	os.Exit(0)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("gai: ")

	flag.Usage = usage
	flag.Parse()

	defer func() {
		// if err, ok := recover().(error); ok {
		// 	log.Fatalln(err)
		// }
	}()

	wd, err := os.Getwd()
	if err != nil {
		fatalf("could not get working directory: %v", err)
	}

	var specs []string
	tspecs := make([]string, len(flag.Args()))

	copy(tspecs, flag.Args())

	for _, fn := range fPkgs {
		specs = append(specs, readLines(fn)...)
	}

	for _, fn := range fTpkgs {
		tspecs = append(tspecs, readLines(fn)...)
	}

	infof("command line specs: %v\n", specs)
	infof("command line tspecs: %v\n", tspecs)

	if len(specs) > 0 {
		specs = gotool.ImportPaths(specs)
	}
	tspecs = gotool.ImportPaths(tspecs)

	if len(specs) == 0 && len(tspecs) == 0 {
		fatalf("nothing to do; no specs provided?")
	}

	roots, nonCore, all := buildDeps(specs, tspecs)

	nonCorePkgs := make([]string, len(nonCore))

	for i, v := range nonCore {
		nonCorePkgs[i] = v.pkg.ImportPath
	}

	getPkgFails := func(vs []string) []string {
		var res []string
		for _, line := range vs {
			if strings.HasPrefix(line, goBuildPkgHeader) {
				line = strings.TrimPrefix(line, goBuildPkgHeader)
				res = append(res, line)
			}
		}
		return res
	}

	var failWork []*depPkg

	failedInstalls := getPkgFails(goDo(nonCorePkgs, "go", "install"))
	for _, v := range failedInstalls {
		p := all[v]

		p.buildStatus = statusFailed
		failWork = append(failWork, p)
	}

	var f *depPkg

	for len(failWork) != 0 {
		f, failWork = failWork[0], failWork[1:]
		for vv := range f.rdeps {
			if vv.buildStatus == statusPassed {
				vv.buildStatus = statusDepFailed
				failWork = append(failWork, vv)
			}
		}
	}

	res := make(chan string)
	gotyped := make(map[*depPkg]bool)

	gotFail := false

	done := make(chan bool)

	go func() {
		for {
			v, ok := <-res
			if !ok {
				break
			}

			if v != "" {
				gotFail = true
				fmt.Fprint(os.Stderr, v)
			}
		}

		close(done)
	}()

	var wg sync.WaitGroup

	togotype := make(map[*depPkg]struct{})
	for _, v := range failedInstalls {
		pkg, ok := all[v]

		if !ok {
			fatalf("failed install %v did not resolve to a package", pkg)
		}

		togotype[pkg] = struct{}{}
	}
	for _, v := range all {
		if v.test {
			togotype[v] = struct{}{}
		}
	}

	for v := range togotype {

		if v.buildStatus == statusDepFailed {
			continue
		}

		rd := v.pkg.Dir

		if filepath.IsAbs(rd) {
			r, err := filepath.Rel(wd, rd)
			if err != nil {
				fatalf("could not calculate filepath.Rel(%q, %q): %v", wd, rd, err)
			}

			rd = r
		}

		wg.Add(1)

		go func(ip, dir string, test bool) {
			out := ""
			var args []string

			if test {
				args = append(args, "-a")
			}

			r := goDo([]string{dir}, "gotype", args...)

			// sort the lines
			splits := make(linesByNumber, len(r))
			for i := range r {
				splits[i] = strings.SplitN(r[i], ":", 4)
			}

			sort.Sort(splits)

			for i := range r {
				r[i] = strings.Join(splits[i], ":")
			}

			if len(r) > 0 {
				out = fmt.Sprintf("# %v\n%v\n", ip, strings.Join(r, "\n"))
			}

			res <- out
			wg.Done()

		}(v.pkg.ImportPath, rd, v.test)

		if v.buildStatus == statusPassed {
			for d := range v.rdeps {

				if d.buildStatus == statusDepFailed {
					continue
				}

				if !gotyped[d] {
					gotyped[d] = true
					roots = append(roots, d)
				}
			}
		}
	}

	wg.Wait()

	close(res)

	<-done

	if gotFail {
		os.Exit(1)
	}
}

type status uint

const (
	statusPassed status = iota
	statusFailed
	statusDepFailed
)

type depPkg struct {
	pkg         *build.Package
	buildStatus status
	test        bool

	deps  map[*depPkg]struct{}
	rdeps map[*depPkg]struct{}
}

func newDepPkg(pkg *build.Package, test bool) *depPkg {
	return &depPkg{
		pkg:  pkg,
		test: test,

		deps:  make(map[*depPkg]struct{}),
		rdeps: make(map[*depPkg]struct{}),
	}
}

func buildDeps(specs []string, tspecs []string) ([]*depPkg, []*depPkg, map[string]*depPkg) {
	// toDo represents the list of package whose imports (and testimports) we need to walk
	toDo := make([]*depPkg, 0, len(specs)+len(tspecs))
	seen := make(map[string]*depPkg)

	wd, err := os.Getwd()
	if err != nil {
		fatalf("could not get working directory: %v", err)
	}

	infof("specs: %v\n", specs)
	infof("tspecs: %v\n", tspecs)

	loadPkg := func(s string, test bool) (*depPkg, bool) {
		if s == "C" {
			return nil, false
		}

		if pkg, ok := seen[s]; ok {
			return pkg, false
		}

		bpkg, err := build.Import(s, wd, 0)
		if err != nil {
			fatalf("could not import %v relative to %v: %v", s, wd, err)
		}

		if pkg, ok := seen[bpkg.ImportPath]; ok {
			return pkg, false
		}

		res := newDepPkg(bpkg, test)

		seen[s] = res
		seen[bpkg.ImportPath] = res

		return res, true
	}

	for _, v := range tspecs {
		if pkg, isnew := loadPkg(v, true); isnew {
			toDo = append(toDo, pkg)
		}
	}

	for _, v := range specs {
		if pkg, isnew := loadPkg(v, false); isnew {
			toDo = append(toDo, pkg)
		}
	}

	var nonCore []*depPkg

	// clearly this only supports us "injecting" packages that need TestImports
	// to be walked at the beginning

	var pkg *depPkg

	for len(toDo) != 0 {
		pkg, toDo = toDo[0], toDo[1:]

		if pkg.pkg.Goroot {
			continue
		}

		nonCore = append(nonCore, pkg)

		var toCheck []string
		toCheck = append(toCheck, pkg.pkg.Imports...)

		if pkg.test {
			toCheck = append(toCheck, pkg.pkg.TestImports...)
		}

		for _, v := range toCheck {
			if v == "C" {
				continue
			}

			dpkg, isnew := loadPkg(v, false)

			if isnew {
				toDo = append(toDo, dpkg)
			}

			pkg.deps[dpkg] = struct{}{}
			dpkg.rdeps[pkg] = struct{}{}
		}
	}

	// find the roots
	var roots []*depPkg
	for _, v := range nonCore {

		hasNonCore := false

		for d := range v.deps {
			if !d.pkg.Goroot {
				hasNonCore = true
			}
		}

		if !hasNonCore {
			roots = append(roots, v)
		}
	}

	return roots, nonCore, seen
}

func goDo(specs []string, c string, args ...string) []string {
	// we don't have a good way of detecting whether a failure of go install
	// is "good" or "bad"; exit codes might tell us something but this does
	// not appear to be documented; maybe rsc's work on the cmd/go suite will
	// change this situation

	var cArgs []string
	cArgs = append(cArgs, args...)
	cArgs = append(cArgs, specs...)

	cmd := exec.Command(c, cArgs...)
	cmdStr := fmt.Sprintf("%v %v", c, strings.Join(cArgs, " "))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fatalf("could not create stderr pipe for %v: %v", cmdStr, err)
	}

	var res []string

	cmd.Start()

	sc := bufio.NewScanner(stderr)
	for sc.Scan() {
		line := sc.Text()
		res = append(res, line)
	}

	if err := sc.Err(); err != nil {
		fatalf("scan error reading %v: %v", cmdStr, err)
	}

	// we don't care for the failed results...
	cmd.Wait()

	infof("running %v %v", cmdStr, res)

	return res
}

type linesByNumber [][]string

func (l linesByNumber) Len() int      { return len(l) }
func (l linesByNumber) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l linesByNumber) Less(i, j int) bool {
	li, lj := l[i], l[j]

	// each line will have at least one part
	if i := strings.Compare(l[i][0], l[j][0]); i < 0 {
		return true
	}

	// in case they both had just a single part
	linei, err := strconv.ParseUint(li[1], 10, 64)
	if err != nil {
		return i < j
	}
	linej, err := strconv.ParseUint(lj[1], 10, 64)
	if err != nil {
		return i < j
	}
	if linei < linej {
		return true
	}

	coli, err := strconv.ParseUint(li[1], 10, 64)
	if err != nil {
		return i < j
	}
	colj, err := strconv.ParseUint(lj[1], 10, 64)
	if err != nil {
		return i < j
	}
	if coli < colj {
		return true
	}

	return i < j
}

func readLines(file string) []string {
	var fi *os.File
	var err error

	if file == "-" {
		fi = os.Stdin
	} else {
		fi, err = os.Open(file)
		if err != nil {
			fatalf("could not open %v: %v", file, err)
		}
	}

	sc := bufio.NewScanner(fi)
	var res []string

	for sc.Scan() {
		res = append(res, sc.Text())
	}
	if err = sc.Err(); err != nil {
		fatalf("could not scan %v: %v", file, err)
	}

	return res
}

func resolvePkgSpec(spec []string) []string {
	var res []string

	args := append([]string{"list", "-e"}, spec...)

	gl := exec.Command("go", args...)

	// we only care for stdout
	out, err := gl.Output()
	if err != nil {
		fatalf("could not run go list: %v", err)
	}

	buf := bytes.NewBuffer(out)

	sc := bufio.NewScanner(buf)

	for sc.Scan() {
		res = append(res, sc.Text())
	}

	return res
}

func infof(format string, args ...interface{}) {
	if *fDebug {
		log.Printf(format, args...)
	}
}

func fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}
