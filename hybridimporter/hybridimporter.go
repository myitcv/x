// Package hybridimporter is an implementation of go/types.ImporterFrom that
// uses depdency export information where it can, falling back to a source-file
// based importer otherwise.
package hybridimporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"

	"myitcv.io/hybridimporter/srcimporter"
)

type pkgInfo struct {
	ImportPath string
	Export     string
	Stale      bool
	Name       string
}

// New returns a go/types.ImporterFrom that uses build cache package files if they
// are available (i.e. compile), dropping back to a src-based importer otherwise.
// path is effectively optional, because if not specified it defaults to ".", i.e.
// the package in dir.
func New(ctxt *build.Context, fset *token.FileSet, dir, path string) (*srcimporter.Importer, error) {
	if path == "" {
		path = "."
	}
	cmd := exec.Command("go", "list", "-deps", "-test", "-json", "-e", "-export", path)
	cmd.Dir = dir

	// Because of https://github.com/golang/go/issues/25842 we first need to
	// check whether we can parse the output - and even then, only the output in
	// stdout - if we can, for now, we take that as a sign of success. When
	// #25842 is resolve we can add back the check for the exit code indicating
	// success/failure, and also read CombinedOutput

	out, _ := cmd.Output()

	// TODO because of https://github.com/golang/go/issues/25842 in Go 1.11 we
	// need to have this fallback. But it also happens to work for Go 1.10
	// (which does not have the extended list capability). So we'll leave this
	// for now....
	// if err != nil {
	// 	if ad, err := filepath.Abs(dir); err == nil {
	// 		dir = ad
	// 	}
	// 	return nil, fmt.Errorf("failed to %v in %v: %v\n%v", strings.Join(cmd.Args, " "), dir, err, string(out))
	// }

	lookups := make(map[string]io.ReadCloser)

	dec := json.NewDecoder(bytes.NewReader(out))

	for {
		var p pkgInfo
		err := dec.Decode(&p)
		if err != nil {
			if io.EOF == err {
				break
			}
			return nil, fmt.Errorf("failed to parse list in %v: %v", dir, err)
		}
		if p.ImportPath == "unsafe" || p.Export == "" || p.Name == "main" {
			continue
		}
		fi, err := os.Open(p.Export)
		if err != nil {
			return nil, fmt.Errorf("failed to open %v: %v", p.Export, err)
		}
		lookups[p.ImportPath] = fi
	}

	lookup := func(path string) (io.ReadCloser, error) {
		rc, ok := lookups[path]
		if !ok {
			return nil, fmt.Errorf("failed to resolve %v", path)
		}

		return rc, nil
	}

	gc := importer.For("gc", lookup)

	tpkgs := make(map[string]*types.Package)

	for path := range lookups {
		p, err := gc.Import(path)
		if err != nil {
			return nil, fmt.Errorf("failed to gc import %v: %v", path, err)
		}
		tpkgs[path] = p
	}

	return srcimporter.New(ctxt, fset, tpkgs), nil
}
