// package vgo provides some utility types, functions etc to support vgo
package vgo // import "myitcv.io/vgo"

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
		bpkg, err := build.ImportDir(dir, 0)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve %v to a package: %v", dir, err)
		}

		// now run vgo depbuildlist with the import path
		args := []string{"vgo", "deplist", "-build"}

		if l.test {
			args = append(args, "-test")
		}

		args = append(args, bpkg.ImportPath)

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = l.dir

		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("unable to run %v: %v [%q]", strings.Join(cmd.Args, " "), err, string(out))
		}

		// parse the JSON

		lookup := make(map[string]string)

		dec := json.NewDecoder(bytes.NewBuffer(out))

		for {
			var d struct {
				ImportPath  string
				PackageFile string
			}

			if err := dec.Decode(&d); err != nil {
				if err == io.EOF {
					break
				}

				return nil, fmt.Errorf("failed to parse vgo output: %v\noutput was:\n%v", err, string(out))
			}

			lookup[d.ImportPath] = d.PackageFile
		}

		i := importer.For(l.compiler, func(path string) (io.ReadCloser, error) {
			file, ok := lookup[path]
			if !ok {
				return nil, fmt.Errorf("failed to resolve import path %q", path)
			}

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
