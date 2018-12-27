package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/build"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"myitcv.io/gogenerate"
)

var (
	pkgs = make(map[string]*pkg)
)

func resolve(ip string) *pkg {
	p, ok := pkgs[ip]
	if ok {
		return p
	}

	p = &pkg{
		ImportPath: ip,
	}
	pkgs[ip] = p
	return p
}

func loadPkgs(specs []string) pkgSet {
	loadOrder, toolsAndOutPkgs := readPkgs(specs, false, false)

	// now ensure we have loaded any tools that were not part of the original
	// package spec; skipping loading them if we have previously loaded them.
	// We skip scanning for any directives... these are external tools
	var toolAndOutSpecs []string
	for _, ip := range toolsAndOutPkgs {
		if p := pkgs[ip]; p.pendingVal == nil {
			toolAndOutSpecs = append(toolAndOutSpecs, ip)
		}
	}

	toolLoadOrder, _ := readPkgs(toolAndOutSpecs, true, true)

	loadOrder = append(toolLoadOrder, loadOrder...)

	loaded := make(pkgSet)
	for _, l := range loadOrder {
		loaded[l] = true
		if l.inPkgSpec {
			l.pendingVal = nil
		}
	}

	for _, p := range loadOrder {
		p.pending()
		if !p.inPkgSpec {
			continue
		}
		for d := range p.deps {
			if d.rdeps == nil {
				d.rdeps = make(pkgSet)
			}
			d.rdeps[p] = true
		}
		for t := range p.toolDeps {
			if t.rdeps == nil {
				t.rdeps = make(pkgSet)
			}
			t.rdeps[p] = true
		}
	}

	// for _, p := range loadOrder {
	// 	fmt.Printf(">> %v\n", p.ImportPath)

	// 	var ds []string
	// 	for d := range p.deps {
	// 		if !d.inPkgSpec || !d.pending() {
	// 			continue
	// 		}
	// 		ds = append(ds, d.ImportPath)
	// 	}
	// 	sort.Strings(ds)
	// 	for _, d := range ds {
	// 		fmt.Printf(" d - %v\n", d)
	// 	}
	// 	for t, dirs := range p.toolDeps {
	// 		if !t.pending() {
	// 			continue
	// 		}
	// 		ods := ""
	// 		if len(dirs) != 0 {
	// 			var odss []string
	// 			for od := range dirs {
	// 				odss = append(odss, od.ImportPath)
	// 			}
	// 			sort.Strings(odss)
	// 			ods = fmt.Sprintf(" [%v]", strings.Join(odss, ","))
	// 		}
	// 		fmt.Printf(" t - %v%v\n", t, ods)
	// 	}
	// }

	return loaded
}

type Package struct {
	Dir          string
	Name         string
	ImportPath   string
	Target       string
	ForTest      string
	Deps         []string
	GoFiles      []string
	CgoFiles     []string
	TestGoFiles  []string
	XTestGoFiles []string
}

type pkg struct {
	Dir        string
	Name       string
	ImportPath string
	Target     string

	GoFiles  []string
	CgoFiles []string

	testPkg   *pkg
	isTestPkg bool

	deps        map[*pkg]bool
	toolDeps    map[*pkg]map[*pkg]bool
	nonToolDeps map[string]bool

	rdeps map[*pkg]bool

	inPkgSpec bool

	isTool     bool
	pendingVal map[*pkg]bool

	hashVal []byte
}

func (p *pkg) String() string {
	return p.ImportPath
}

func (p *pkg) pending() bool {
	if p.pendingVal == nil {
		p.pendingVal = make(pkgSet)
		for d := range p.deps {
			if d.pending() {
				p.pendingVal[d] = true
			}
		}
		for t := range p.toolDeps {
			if t.pending() {
				p.pendingVal[t] = true
			}
		}
		if p.isTool || len(p.toolDeps) > 0 || len(p.pendingVal) > 0 || len(p.nonToolDeps) > 0 {
			// the install/generate step
			p.pendingVal[p] = true
		}
	}

	return len(p.pendingVal) > 0
}

func (p *pkg) ready() bool {
	p.pending()
	switch len(p.pendingVal) {
	case 0:
		return true
	case 1:
		if p.pendingVal[p] {
			return true
		}
	}
	return false
}

func (p *pkg) donePending(v *pkg) {
	if _, ok := p.pendingVal[v]; !ok {
		if p == v && !p.isTool && len(p.toolDeps) == 0 && len(p.nonToolDeps) == 0 {
			return
		}
		debugf("we had:\n")
		for d := range p.pendingVal {
			debugf(" => %v\n", d)
		}
		fatalf("tried to complete pending for %v in %v but did not exist", v, p)
	}
	delete(p.pendingVal, v)
}

func (p *pkg) hash() []byte {
	if p.hashVal != nil {
		return p.hashVal
	}
	h := sha256.New()
	// when we enable full loading of deps this distinction will
	// go away
	if p.inPkgSpec {
		var deps []*pkg
		for d := range p.deps {
			if d.inPkgSpec {
				deps = append(deps, d)
			}
		}
		for t := range p.toolDeps {
			deps = append(deps, t)
		}
		sort.Slice(deps, func(i, j int) bool {
			return deps[i].ImportPath < deps[j].ImportPath
		})
		for _, d := range deps {
			if _, err := h.Write(d.hash()); err != nil {
				fatalf("failed to hash: %v", err)
			}
		}
		p.hashFiles(h, p.GoFiles)
		p.hashFiles(h, p.CgoFiles)
	}
	p.hashVal = h.Sum(nil)
	return p.hashVal
}

func (p *pkg) hashFiles(h hash.Hash, files []string) {
	for _, f := range files {
		path := f
		if !filepath.IsAbs(f) {
			path = filepath.Join(p.Dir, f)
		}
		fi, err := os.Open(path)
		if err != nil {
			fatalf("failed to open %v: %v", path, err)
		}
		_, err = io.Copy(h, fi)
		fi.Close()
		if err != nil {
			fatalf("failed to hash %v: %v", path, err)
		}
	}
}

type hashRes map[*pkg]string

func (h hashRes) equals(v hashRes) (bool, error) {
	if len(h) != len(v) {
		return false, fmt.Errorf("hashRes length mismatch")
	}

	for hk, hv := range h {
		vv, ok := v[hk]
		if !ok {
			return false, fmt.Errorf("hashRes key mistmatch")
		}
		if hv != vv {
			return false, nil
		}
	}

	return true, nil
}

func (p *pkg) snap() hashRes {
	res := make(hashRes)
	res[p] = string(p.hash())
	for _, outPkgMap := range p.toolDeps {
		for op := range outPkgMap {
			res[op] = string(op.hash())
		}
	}
	return res
}

func (p *pkg) zeroSnap() hashRes {
	res := make(hashRes)
	res[p] = ""
	for _, outPkgMap := range p.toolDeps {
		for op := range outPkgMap {
			res[op] = ""
		}
	}
	return res
}

type pkgSet map[*pkg]bool

// returns the ordered pkg load list and a list of import paths of tools and
// output packages detected during scanning of the loaded packages' directives
// that were not part of the original package spec. These might need a second
// load
func readPkgs(pkgs []string, dontScan bool, notInPkgSpec bool) ([]*pkg, []string) {
	if len(pkgs) == 0 {
		return nil, nil
	}

	var loadOrder []*pkg

	toolsAndOutPkgs := make(pkgSet)
	res := make(chan *Package)

	go func() {
		args := []string{"go", "list", "-test", "-f", `{{$pkg := .}}{{with (eq .ForTest "")}}{{with $pkg}}{"Dir": "{{.Dir}}", "Name": "{{.Name}}", "ImportPath": "{{.ImportPath}}"{{if eq .Name "main"}}, "Target": "{{.Target}}"{{end}}{{with .Deps}}, "Deps": ["{{join . "\", \""}}"]{{end}}{{with .GoFiles}}, "GoFiles": ["{{join . "\", \""}}"]{{end}}{{with .TestGoFiles}}, "TestGoFiles": ["{{join . "\", \""}}"]{{end}}{{with .CgoFiles}}, "CgoFiles": ["{{join . "\", \""}}"]{{end}}{{with .XTestGoFiles}}, "XTestGoFiles": ["{{join . "\", \""}}"]{{end}}}{{end}}{{end}}`}
		args = append(args, pkgs...)
		cmd := exec.Command(args[0], args[1:]...)

		out, err := cmd.Output()
		if err != nil {
			var stderr []byte
			if err, ok := err.(*exec.ExitError); ok {
				stderr = err.Stderr
			}
			fatalf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr)
		}

		dec := json.NewDecoder(bytes.NewReader(out))
		for {
			var p Package
			if err := dec.Decode(&p); err != nil {
				if err == io.EOF {
					break
				}
				fatalf("failed to decode output from golist: %v %v\n%s", strings.Join(cmd.Args, " "), err, out)
			}
			if p.ForTest == "" {
				res <- &p
			}
		}
		close(res)
	}()

	for pp := range res {
		// we collapse down the test deps into the package deps
		// because from a go generate perspective they are one and
		// the same. We don't care for the files in the test

		p := resolve(pp.ImportPath)
		p.Dir = pp.Dir
		p.Name = pp.Name
		p.Target = pp.Target

		p.GoFiles = pp.GoFiles
		p.CgoFiles = pp.CgoFiles

		// TODO pending https://go-review.googlesource.com/c/go/+/112755
		p.inPkgSpec = !notInPkgSpec

		loadOrder = append(loadOrder, p)

		// invalidate any existing hash
		p.hashVal = nil

		ip := pp.ImportPath
		if strings.HasSuffix(ip, ".test") {
			p.isTestPkg = true
			ip = strings.TrimSuffix(ip, ".test")
			rp := resolve(ip)
			rp.testPkg = p
			continue
		}

		p.deps = make(pkgSet)

		for _, d := range pp.Deps {
			p.deps[resolve(d)] = true
		}

		if dontScan {
			continue
		}

		var gofiles []string
		gofiles = append(gofiles, pp.GoFiles...)
		gofiles = append(gofiles, pp.CgoFiles...)
		gofiles = append(gofiles, pp.TestGoFiles...)
		gofiles = append(gofiles, pp.XTestGoFiles...)

		dirs := make(map[*pkg]map[*pkg]bool)
		nonToolDirs := make(map[string]bool)

		for _, f := range gofiles {
			check := func(line int, args []string) error {
				// TODO add support for go run with package

				cmd := args[0]
				cmdPath, ok := config.baseCmds[cmd]
				if !ok {
					// check if it's a nonTool
					if _, ok := config.nonCmds[cmd]; ok {
						nonToolDirs[cmd] = true
						return nil
					}
					return fmt.Errorf("failed to resolve cmd %v", cmd)
				}
				cmdPkg := resolve(cmdPath)
				pm, ok := dirs[cmdPkg]
				if !ok {
					pm = make(map[*pkg]bool)
					dirs[cmdPkg] = pm
					cmdPkg.isTool = true
					toolsAndOutPkgs[cmdPkg] = true
				}

				for i, a := range args {
					if a == "--" {
						// end of flags
						break
					}
					const prefix = "-" + gogenerate.FlagOutPkgPrefix
					if !strings.HasPrefix(a, prefix) {
						continue
					}

					rem := strings.TrimPrefix(a, prefix)
					if len(rem) == 0 || rem[0] == '=' || rem[0] == '-' {
						return fmt.Errorf("bad arg %v", a)
					}

					var dirOrPkg string
					var pkgPath string

					for j := 1; j < len(rem); j++ {
						if rem[j] == '=' {
							dirOrPkg = rem[j+1:]
							goto ResolveDirOrPkg
						}
					}

					if i+1 == len(args) {
						return fmt.Errorf("bad args %q", strings.Join(args, " "))
					} else {
						dirOrPkg = args[i+1]
					}

				ResolveDirOrPkg:
					// TODO we could improve this logic
					if filepath.IsAbs(dirOrPkg) {
						bpkg, err := build.ImportDir(dirOrPkg, build.FindOnly)
						if err != nil {
							return fmt.Errorf("failed to resolve %v to a package: %v", dirOrPkg, err)
						}
						pkgPath = bpkg.ImportPath
					} else {
						bpkg, err := build.Import(dirOrPkg, p.Dir, build.FindOnly)
						if err != nil {
							return fmt.Errorf("failed to resolve %v in %v to a package: %v", dirOrPkg, p.Dir, err)
						}
						pkgPath = bpkg.ImportPath
					}

					outPkg := resolve(pkgPath)
					pm[outPkg] = true
					toolsAndOutPkgs[outPkg] = true
				}

				return nil
			}
			if err := gogenerate.DirFunc(ip, p.Dir, f, check); err != nil {
				fatalf("error checking %v: %v", filepath.Join(p.Dir, f), err)
			}
		}

		for d, ods := range dirs {
			if p.toolDeps == nil {
				p.toolDeps = make(map[*pkg]map[*pkg]bool)
			}
			p.toolDeps[d] = ods

			// verify that none of the output directories are a Dep
			for op := range ods {
				if p.deps[op] {
					fatalf("package %v has directive %v that outputs to %v, but that is also a dep", p.ImportPath, d, op)
				}
			}
		}

		for cmd := range nonToolDirs {
			if p.nonToolDeps == nil {
				p.nonToolDeps = make(map[string]bool)
			}
			p.nonToolDeps[cmd] = true
		}
	}

	var ips []string
	for t := range toolsAndOutPkgs {
		if !t.inPkgSpec {
			ips = append(ips, t.ImportPath)
		}
	}

	return loadOrder, ips
}
