// package vgo provides some utility types, functions etc to support vgo
//
// For now it is a copy of the WIP myitcv.io/vgo
package vgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"go/importer"
	"go/types"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Loader supports loading of vgo-build cached packages. NewLoader returns a
// correctly initialised *Loader.  A Loader must not be copied once created.
type Loader struct {
	mu sync.Mutex

	Debug io.Writer

	dir       string
	compiler  string
	resCache  map[string]map[string]*types.Package
	importers map[string]types.ImporterFrom
	test      bool
}

func NewLoader(dir string) *Loader {
	res := &Loader{
		dir:       dir,
		compiler:  "gc",
		resCache:  make(map[string]map[string]*types.Package),
		importers: make(map[string]types.ImporterFrom),
	}

	return res
}

func NewTestLoader(dir string) *Loader {
	res := NewLoader(dir)
	res.test = true
	return res
}

var _ types.ImporterFrom = new(Loader)

func (l *Loader) Import(path string) (*types.Package, error) {
	return nil, fmt.Errorf("did not expect this method to be used; we implement types.ImporterFrom")
}

func (l *Loader) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if mode != 0 {
		panic(fmt.Errorf("unknown types.ImportMode %v", mode))
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// TODO optimise mutex usage later... keep it simple for now
	dirCache, ok := l.resCache[dir]
	if ok {
		if p, ok := dirCache[path]; ok {
			return p, nil
		}
	} else {
		// ensures dirCache is now set
		dirCache = make(map[string]*types.Package)
		l.resCache[dir] = dirCache
	}

	// res cache miss
	imp, ok := l.importers[dir]
	if !ok {
		// we need to load the results for this dir and build an importer

		// resolve the package found in dir
		_, err := build.ImportDir(dir, 0)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve %v to a package: %v", dir, err)
		}

		// now run vgo depbuildlist with the import path
		args := []string{"vgo", "deplist", "-build"}

		if l.test {
			args = append(args, "-test")
		}

		args = append(args, ".")

		combined := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		stdout := new(bytes.Buffer)

		mu := new(sync.Mutex)

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = l.dir
		cmd.Stderr = newSyncMultiWriter(mu, stderr, combined)
		cmd.Stdout = newSyncMultiWriter(mu, stdout, combined)

		l.debugf("dir: %v, running %v\n", cmd.Dir, strings.Join(cmd.Args, " "))

		if err := cmd.Run(); err != nil {
			l.debugf("failed: %v\n%v\n", err, combined.String())
			return nil, fmt.Errorf("unable to run %v: %v [%q]", strings.Join(cmd.Args, " "), err, combined.String())
		}

		l.debugf("output:\n%v\n", combined.String())

		// parse the JSON

		dec := json.NewDecoder(bytes.NewBuffer(stdout.Bytes()))

		type vgoDeplistResult struct {
			ImportPath  string
			PackageFile string
			Incomplete  bool
		}

		lookup := make(map[string]vgoDeplistResult)

		for {
			var d vgoDeplistResult

			if err := dec.Decode(&d); err != nil {
				if err == io.EOF {
					break
				}

				return nil, fmt.Errorf("failed to parse vgo output: %v\noutput was:\n%v", err, combined.String())
			}

			lookup[d.ImportPath] = d
		}

		i := importer.For(l.compiler, func(path string) (io.ReadCloser, error) {
			d, ok := lookup[path]
			if !ok {
				return nil, fmt.Errorf("failed to resolve import path %q", path)
			}

			if d.Incomplete {
				return nil, fmt.Errorf("import path %v failed to compile", path)
			}

			file := d.PackageFile

			f, err := os.Open(file)
			if err != nil {
				return nil, fmt.Errorf("failed to open file %v: %v", file, err)
			}

			return f, nil
		})

		from, ok := i.(types.ImporterFrom)
		if !ok {
			return nil, fmt.Errorf("failed to get an importer that implements go/types.ImporterFrom; got %T", i)
		}

		imp = from
		l.importers[dir] = imp
	}

	p, err := imp.ImportFrom(path, dir, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to import: %v", err)
	}

	dirCache[path] = p

	return p, nil
}

func (l *Loader) debugf(format string, args ...interface{}) {
	if l.Debug != nil {
		fmt.Fprintf(l.Debug, format, args...)
	}
}

type syncWriter struct {
	mu *sync.Mutex
	u  io.Writer
}

func newSyncMultiWriter(mu *sync.Mutex, ws ...io.Writer) syncWriter {
	return syncWriter{
		mu: mu,
		u:  io.MultiWriter(ws...),
	}
}

func (s syncWriter) Write(b []byte) (int, error) {
	s.mu.Lock()
	n, err := s.u.Write(b)
	s.mu.Unlock()
	return n, err
}
