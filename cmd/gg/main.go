package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
	"unicode"

	"github.com/rogpeppe/go-internal/cache"
	"github.com/rogpeppe/go-internal/imports"
	"myitcv.io/gogenerate"
)

//go:generate gobin -m -run myitcv.io/cmd/gg/internal/genmain

const (
	hashSize  = 32
	debug     = false
	trace     = false
	traceTime = false
	hashDebug = false
)

type tagsFlag []string

func (t *tagsFlag) String() string {
	return strings.Join(*t, " ")
}

func (t *tagsFlag) Set(s string) error {
	parts := strings.Fields(s)
	*t = append(*t, parts...)
	return nil
}

var (
	flagSet           = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fDebug            = flagSet.Bool("debug", debug, "debug mode")
	fTrace            = flagSet.Bool("trace", trace, "trace timings")
	fTraceTime        = flagSet.Bool("traceTime", traceTime, "trace timings")
	fGraph            = flagSet.Bool("graph", false, "dump dependency graph")
	fWorkP            = flagSet.Int("p", runtime.NumCPU(), "the number of bits of work that can be run in parallel")
	fMaxGenIterations = flagSet.Int("r", 10, "maximum number of generation iterations per package")
	fTags             tagsFlag

	// isProgram indicates whether we are running via a testscript test or not. In case
	// we are running via a testscript test we only end up invoking main1; hence
	// we set isProgram = true when main is invoked
	isProgram bool

	nilHash [hashSize]byte

	tabber    = tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', tabwriter.AlignRight)
	startTime = time.Now()
	lastTime  = startTime
)

func init() {
	flagSet.Var(&fTags, "tags", "space-separated list of tags (can appear multiple times)")
}

func main() {
	isProgram = true
	os.Exit(main1())
}

func main1() int {
	logTiming("start main")
	defer tabber.Flush()
	defer logTiming("end main")
	if err := mainerr(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func newDepsMap() depsMap {
	return depsMap{
		deps:      make(map[dep]bool),
		rdeps:     make(map[dep]bool),
		dirtydeps: make(map[dep]bool),
	}
}

func installGoWrapper() (string, error) {
	td, err := ioutil.TempDir("", "gg_gobin_tmp_")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir for gobin install: %v", err)
	}
	godir := filepath.Join(td, "gosrc")
	if err := os.Mkdir(godir, 0777); err != nil {
		return "", fmt.Errorf("failed to create go dir %v: %v", godir, err)
	}
	mod := filepath.Join(godir, "go.mod")
	if err := ioutil.WriteFile(mod, []byte("module mod"), 0666); err != nil {
		return "", fmt.Errorf("failed to write go.mod file %v: %v", mod, err)
	}
	main := filepath.Join(godir, "main.go")
	cts, err := strconv.Unquote(goMain)
	if err != nil {
		return "", fmt.Errorf("failed to unquote main contents: %v", err)
	}
	if err := ioutil.WriteFile(main, []byte(cts), 0666); err != nil {
		return "", fmt.Errorf("failed to write main file %v: %v", main, err)
	}
	cmd := exec.Command("go", "build", "-o="+filepath.Join(td, "go"), "main.go")
	cmd.Dir = godir

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to run %v in %v: %v\n%s", strings.Join(cmd.Args, " "), godir, err, out)
	}
	return td, nil
}

func mainerr() (reterr error) {
	flagSet.Usage = func() {
		mainUsage(os.Stderr)
	}
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return err
	}

	if *fWorkP < 1 {
		return fmt.Errorf("value for -p must be at least 1")
	}

	mm, err := mainMod()
	if err != nil {
		return fmt.Errorf("failed to determine main module: %v", err)
	}

	if *fTraceTime {
		*fTrace = false
	}

	if *fTrace && isProgram {
		td, err := installGoWrapper()
		if err != nil {
			return err
		}
		defer os.RemoveAll(td)
	}

	sort.Slice(fTags, func(i, j int) bool {
		return fTags[i] < fTags[j]
	})

	ucd, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to determine user cache dir: %v", err)
	}

	artefactsCacheDir := filepath.Join(ucd, "gg-artefacts")
	if err := os.MkdirAll(artefactsCacheDir, 0777); err != nil {
		return fmt.Errorf("failed to create build cache dir %v: %v", artefactsCacheDir, err)
	}

	artefactsCache, err := cache.Open(artefactsCacheDir)
	if err != nil {
		return fmt.Errorf("failed to open build cache dir: %v", err)
	}
	defer artefactsCache.Trim()

	td, err := ioutil.TempDir("", "gg-workings")
	if err != nil {
		return fmt.Errorf("failed to create temp dir for workings: %v", err)
	}
	defer os.RemoveAll(td)

	var tags []string
	tagsMap := make(map[string]bool)
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	tagsMap[goos] = true
	tagsMap[goarch] = true
	for _, t := range fTags {
		tagsMap[t] = true
		tags = append(tags, t)
	}
	// TODO add support for parsing GOFLAGS to extract -tags values
	// TODO once we get a resolution on https://github.com/golang/go/issues/26849#issuecomment-460301061
	// we can then set GOFLAGS for in the environment passed to each generator

	gg := &gg{
		pkgLookup:         make(map[string]*pkg),
		dirLookup:         make(map[string]*pkg),
		gobinModLookup:    make(map[string]*gobinModDep),
		gobinGlobalLookup: make(map[string]string),
		gobinGlobalCache:  make(map[string]*gobinGlobalDep),
		commLookup:        make(map[string]*commandDep),
		GOOS:              goos,
		GOARCH:            goarch,
		tagsMap:           tagsMap,
		tags:              tags,
		cache:             artefactsCache,
		tempDir:           td,
	}
	gg.cliPatts = flagSet.Args()
	gg.mainMod = mm

	if !*fDebug {
		defer func() {
			if err := recover(); err != nil {
				if rerr, ok := err.(error); ok {
					reterr = rerr
				} else {
					panic(fmt.Errorf("got something other than an error: %v [%T]", err, err))
				}
			}
		}()
	}

	gg.run()

	return reterr
}

func mainMod() (string, error) {
	// TODO performance: instead of exec-ing we could work this out ourselves (~16ms)
	var stderr bytes.Buffer
	cmd := exec.Command("go", "env", "GOMOD")
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.Bytes())
	}

	return strings.TrimSpace(string(out)), nil
}

type gg struct {
	GOOS   string
	GOARCH string

	cliPatts []string
	mainMod  string

	// map of import path to pkg. This is the definitive set of all *pkg
	pkgLookup map[string]*pkg

	// map of directory to pkg
	dirLookup map[string]*pkg

	// map of import path to dep
	gobinModLookup map[string]*gobinModDep

	// map of import path to target file
	gobinGlobalLookup map[string]string
	// map of target file to dep
	gobinGlobalCache map[string]*gobinGlobalDep

	// map of command name to dep
	commLookup map[string]*commandDep

	// tagsMap contains build tags + GOOS + GOARCH
	tagsMap map[string]bool
	// tags is just the build tags provided via GOFLAGS or -tags
	tags []string

	cache *cache.Cache

	tempDir string
}

func (g *gg) allDeps() []dep {
	var res []dep
	for _, p := range g.pkgLookup {
		res = append(res, p)
	}
	for _, gb := range g.gobinModLookup {
		res = append(res, gb)
	}
	for _, gb := range g.gobinGlobalCache {
		res = append(res, gb)
	}
	for _, c := range g.commLookup {
		res = append(res, c)
	}
	return res
}

func (g *gg) generate(w *pkg) (moreWork []dep) {
	for {
		if w.genCount == *fMaxGenIterations {
			g.fatalf("hit max number of iterations (%v) for %v", *fMaxGenIterations, w.ImportPath)
		}
		w.genCount++

		deltaDirs := make(map[string]bool)
		canContinue := func() bool {
			importMisses := make(missingDeps)
			var newWork []dep

			for od := range deltaDirs {
				odp, ok := g.dirLookup[od]
				if !ok {
					continue
				}
				g.undo(odp)
				g.refreshImports(odp, importMisses)
				newWork = append(newWork, odp)
			}

			dirMisses := make(missingDeps)
			g.refreshDirectiveDeps(w, dirMisses)

			g.loadMisses(importMisses, dirMisses)

			// if the current package is still ready then we continue
			// for another round of generation
			if w.Ready() {
				return true
			}

			var nw dep
			for len(newWork) > 0 {
				nw, newWork = newWork[0], newWork[1:]
				if nw.Done() {
					continue
				}
				if nw.Ready() {
					moreWork = append(moreWork, nw)
				} else {
					for d := range nw.Deps().deps {
						newWork = append(newWork, d)
					}
				}
			}

			// because at this point the current piece of work cannot
			// be finished and so we need to bail early
			return false
		}

		hw := newHash("## generate " + w.ImportPath)
		fmt.Fprintf(hw, "goos %v goarch %v\n", g.GOOS, g.GOARCH)
		// we add tags to the generate hash because we can't know a generator
		// will use them.
		fmt.Fprintf(hw, "tags: %v\n", fTags)
		fmt.Fprintf(hw, "Deps:\n")
		g.hashDeps(hw, w)
		fmt.Fprintf(hw, "Directives:\n")
		outDirs := make(map[string]bool)
		dirNames := make(map[string]map[generator]bool)
		for _, d := range w.dirs {
			fmt.Fprintf(hw, "%v\n", d.HashString())
			gens := dirNames[d.gen.DirectiveName()]
			if gens == nil {
				gens = make(map[generator]bool)
				dirNames[d.gen.DirectiveName()] = gens
			}
			gens[d.gen] = true
			for _, od := range d.outDirs {
				outDirs[od] = true
			}
		}
		outDirs[w.Dir] = true
		var outDirOrder []string
		for od := range outDirs {
			outDirOrder = append(outDirOrder, od)
		}
		sort.Strings(outDirOrder)
		// TODO performance: in theory we could reuse the "post" from the previous round
		// for pre. Unclear whether there would be any benefit from so doing
		pre := g.hashOutDirs(outDirOrder, w, hw, dirNames)

		if fp, _, err := g.cache.GetFile(hw.Sum()); err == nil {
			r, err := newArchiveReader(fp)
			if err != nil {
				goto CacheMiss
			}
			for {
				fn, err := r.ExtractFile()
				if err != nil {
					if err == io.EOF {
						break
					}
					goto CacheMiss
				}
				deltaDirs[filepath.Dir(fn)] = true
			}
			if err := r.Close(); err != nil {
				goto CacheMiss
			}
			if len(deltaDirs) == 0 {
				// zero delta to apply; we are done
				break
			}

			if canContinue() {
				continue
			}
			return
		}
	CacheMiss:

		// if we get here we had a cache miss so we are going to have to run
		// go generate

		logTrace("generate %v", w)
		for _, d := range w.dirs {
			cmd := exec.Command(d.args[0], d.args[1:]...)
			if *fTrace {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			}
			cmd.Dir = w.Dir
			cmd.Env = append(os.Environ(),
				"GOARCH="+g.GOARCH,
				"GOOS="+g.GOOS,
				"GOFILE="+d.file,
				"GOLINE="+strconv.Itoa(d.line),
				"GOPACKAGE="+d.pkgName,
				"DOLLAR="+"$",
			)

			var traceArgs string
			if *fTrace || *fTraceTime {
				var pargs []string
				for _, a := range d.args {
					if strings.IndexFunc(a, unicode.IsSpace) != -1 {
						a = "'" + a + "'"
					}
					pargs = append(pargs, a)
				}
				traceArgs = strings.Join(pargs, " ")
				line := fmt.Sprintf("run generator: %v", traceArgs)
				logTiming(line)
				logTrace(line)
			}

			var out []byte
			var err error
			if *fTrace {
				err = cmd.Run()
			} else {
				out, err = cmd.CombinedOutput()
			}
			if err != nil {
				g.fatalf("failed to run %v in %v: %v\n%s", strings.Join(cmd.Args, " "), w.Dir, err, out)
			}
			if *fTrace || *fTraceTime {
				line := fmt.Sprintf("ran generator: %v", traceArgs)
				logTiming(line)
				logTrace(line)
			}
		}

		post := g.hashOutDirs(outDirOrder, w, nil, dirNames)
		ar := g.newArchive()

		// TODO work out if/how we handle file removals, i.e. files that got _removed_
		// by any of the generators in any of the output directories... and make them -1
		// entries in the archive, i.e. remove
		var delta []string
		for fn, posth := range post {
			preh, ok := pre[fn]
			if !ok || preh != posth {
				delta = append(delta, fn)
				deltaDirs[filepath.Dir(fn)] = true
			}
		}
		if len(delta) == 0 {
			if err := g.cachePutArchive(hw.Sum(), ar); err != nil {
				g.fatalf("failed to put zero-length archive: %v", err)
			}
			break
		}

		sort.Strings(delta)
		for _, f := range delta {
			if err := ar.PutFile(f); err != nil {
				g.fatalf("failed to put %v into archive: %v", f, err)
			}
		}
		if err := g.cachePutArchive(hw.Sum(), ar); err != nil {
			g.fatalf("failed to write archive to cache: %v", err)
		}

		if canContinue() {
			continue
		}
		return
	}
	return
}

func (g *gg) addDep(d dep, nd dep) {
	if *fDebug {
		var count int
		_, depsok := d.Deps().deps[nd]
		if depsok {
			count++
		}
		_, ddepsok := d.Deps().dirtydeps[nd]
		if ddepsok {
			count++
		}
		_, rdepsok := nd.Deps().rdeps[d]
		if rdepsok {
			count++
		}
		if count != 0 && count != 3 {
			g.fatalf("inconsistency in terms of deps %v and %v; %v %v %v", d, nd, depsok, ddepsok, rdepsok)
		}
	}
	d.Deps().deps[nd] = true
	if !nd.Done() {
		d.Deps().dirtydeps[nd] = true
	}
	nd.Deps().rdeps[d] = true
}

func (g *gg) dropDep(d dep, od dep) {
	if _, ok := d.Deps().deps[od]; !ok {
		g.fatalf("inconsistency: %v is not a dep of %v", od, d)
	}
	if _, ok := od.Deps().rdeps[d]; !ok {
		g.fatalf("inconsistency: %v is not an rdep of %v", d, od)
	}
	delete(d.Deps().deps, od)
	delete(d.Deps().dirtydeps, od)
	delete(od.Deps().rdeps, d)
}

func (g *gg) doneDep(dd dep) []dep {
	if !dd.Done() {
		g.fatalf("tried to mark %v as done, but it's not done", dd)
	}
	var work []dep
	for d := range dd.Deps().rdeps {
		if _, ok := d.Deps().deps[dd]; !ok {
			g.fatalf("failed to mark dep %v as done; is not currently a dep", dd)
		}
		if _, ok := d.Deps().dirtydeps[dd]; !ok {
			g.fatalf("failed to mark dep %v as done; is not currently dirty", dd)
		}
		delete(d.Deps().dirtydeps, dd)
		if len(d.Deps().dirtydeps) == 0 {
			work = append(work, d)
		}
	}
	return work
}

func (g *gg) list(patts []string, opts ...string) ([]*Package, error) {

	cmd := exec.Command("go", "list")
	if *fTrace {
		cmd.Stderr = os.Stderr
	}

	hasDeps := false
	for _, o := range opts {
		if o == "-deps" {
			hasDeps = true
		}
	}

	if len(g.tags) > 0 {
		opts = append(opts, "-tags="+strings.Join(g.tags, " "))
	}

	if !hasDeps {
		opts = append(opts, "-find")
	}

	opts = append(opts, "-json")
	opts, err := adjustTagsFlag(opts)
	if err != nil {
		return nil, err
	}

	cmd.Args = append(cmd.Args, opts...)
	cmd.Args = append(cmd.Args, patts...)

	// TODO optimise by using -f here for only the fields we need

	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}

		return nil, fmt.Errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr)
	}

	dec := json.NewDecoder(bytes.NewReader(out))

	var res []*Package

	for {
		var p Package

		if err := dec.Decode(&p); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to decode output from %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
		}

		// For anything go generate related we are simply interested
		// in the package itself. This includes all of the files that
		// we care about (that are not ignored by build constraints).
		if p.ForTest != "" || strings.HasSuffix(p.ImportPath, ".test") {
			continue
		}

		res = append(res, &p)
	}

	return res, nil
}

// load loads the packages resolved by patts and their deps
func (g *gg) run() {
	logTiming("start run")
	// Resolve patterns to pkgs
	pkgs, err := g.list(g.cliPatts, "-deps", "-test")
	if err != nil {
		g.fatalf("failed to load patterns [%v]: %v", strings.Join(g.cliPatts, " "), err)
	}
	logTiming("initial list complete")

	// If we are in module mode, work out whether we are being asked to generate
	// for packages outside the main module. If we are this is an error. In GOPATH
	// mode, there is no such thing as main module, and any other packages will be
	// within the writable GOPATH... so this is allowed.
	var badPkgs []string
	for _, p := range pkgs {
		if len(p.Match) > 0 && p.Module != nil && p.Module.GoMod != g.mainMod {
			badPkgs = append(badPkgs, p.ImportPath)
		}
	}
	if len(badPkgs) > 0 {
		g.fatalf("cannot generate packages outside the main module: %v", strings.Join(badPkgs, ", "))
	}

	// Populate the import path and directory lookup maps, and mark *pkg according
	// to whether they should be generated. Keep a slice of those to be generated
	// for convenience.
	for _, p := range pkgs {
		np := g.newPkg(p)
		if len(p.Match) > 0 {
			np.generate = true
			if np.x != nil {
				np.x.generate = true
			}
		}
	}

	// Calculate dependency and reverse dependency graph. For packages that we are generating,
	// also consider test imports.
	for _, p := range g.pkgLookup {
		g.addDepsFromImports(p)
	}
	logTiming("dep graph complete")

	// There is one class of generator that we might not have
	// seen as part of the previous go list; gobin -m (and indeed
	// go run). Such generators will resolve their dependencies via
	// the main module, but go:generate directives are not seen by
	// the go tool. There is no way to automatically include such
	// tools (go list -tags tools results in multiple packages in the
	// the same directory).
	//
	// Similarly, when we are generating code later, it might be that
	// dependencies get "added" as part of the generation process
	// that we have not seen yet. After each phase (and this discovery
	// of the generators is such a phase) we might therefore need to
	// resolve a number of imports. We do this in a single command to
	// keep things efficient.
	gobinModMisses := make(missingDeps)

	// go:generate directive search amongst *pkg to generate. If we happen
	// to find a generator which itself is a generate target that's fine;
	// we will simply end up recording the dependency on that generator
	// (it can't by definition be an import dependency).
	for _, p := range g.pkgLookup {
		visit := func(p *pkg) {
			if !p.generate {
				return
			}
			g.refreshDirectiveDeps(p, gobinModMisses)

			if p.x != nil {
				g.refreshDirectiveDeps(p.x, gobinModMisses)
			}
		}
		visit(p)
		if p.x != nil {
			visit(p.x)
		}
	}
	logTiming("initial refreshDirectiveDeps complete")

	g.loadMisses(nil, gobinModMisses)

	logTiming("initial loadMisses complete")

	// At this point we should have a complete dependency graph, including the generators.
	// Find roots and start work
	if *fGraph {
		fmt.Printf("digraph {\n")
	}
	var work []dep
	for _, d := range g.allDeps() {
		if *fGraph {
			if p, isPkg := d.(*pkg); !isPkg || !p.Standard {
				for rd := range d.Deps().rdeps {
					lhs := strings.Replace(d.String(), "\"", "\\\"", -1)
					rhs := strings.Replace(rd.String(), "\"", "\\\"", -1)
					fmt.Printf("\"%v\" -> \"%v\";\n", lhs, rhs)
				}
			}
		}
		if d.Ready() {
			work = append(work, d)
		}
	}
	if *fGraph {
		fmt.Printf("}\n")
	}

	logTiming("start work")

	for len(work) > 0 {
		// It may be that since we added a piece of work one of its deps has been
		// marked as not ready, and hence len(w.Deps().dirtyDeps) > 0. Drop any
		// such work
		if *fWorkP == 1 {
			sortDeps(work)
		}
		var i int
		todo := make(map[dep]bool)
		var haveGenerate bool
		for {
			if i == len(work) || haveGenerate || len(todo) == *fWorkP {
				work = work[i:]
				break
			}
			w := work[i]
			if !w.Ready() || w.Done() {
				continue
			}
			if p, ok := w.(*pkg); ok && p.generate {
				haveGenerate = true
			}
			todo[w] = true
			i++
		}
		var todoOrder []dep
		for w := range todo {
			todoOrder = append(todoOrder, w)
		}

		var wg sync.WaitGroup

		// TODO performance: this is not necessary
		sortDeps(todoOrder)

		for _, w := range todoOrder {
			wg.Add(1)
			func(w dep) {
				defer wg.Done()
				switch w := w.(type) {
				case *pkg:
					if w.generate {
						if len(w.dirs) > 0 {
							moreWork := g.generate(w)
							debugf("more work: %v\n", moreWork)
							if len(moreWork) != 0 {
								work = append(work, moreWork...)
								return
							}
						}
						w.generated = true
					}
					if w.isXTest {
						// no need to hash
						return
					}
					if !w.Standard {
						logTrace("hash %v", w)
					}
					hw := newHash("## pkg " + w.ImportPath)
					fmt.Fprintf(hw, "## pkg %v\n", w.ImportPath)
					for _, i := range w.Imports {
						g.hashImport(hw, i)
					}
					files := stringList(
						w.GoFiles,
						w.CgoFiles,
						w.CFiles,
						w.CXXFiles,
						w.FFiles,
						w.MFiles,
						w.HFiles,
						w.SFiles,
						w.SysoFiles,
						w.SwigFiles,
						w.SwigCXXFiles,
					)
					for _, fn := range files {
						g.hashFile(hw, w.Dir, fn)
					}
					w.hash = hw.Sum()
				case *commandDep:
					logTrace("hash commandDep %v", w)
					hw := newHash("## commandDep " + w.name)
					fp, err := exec.LookPath(w.name)
					if err != nil {
						g.fatalf("failed to find %v in PATH: %v", w.name, err)
					}
					g.hashFile(hw, "", fp)
					w.hash = hw.Sum()
				case *gobinGlobalDep:
					logTrace("hash gobinGlobalDep %v", w)
					hw := newHash("## gobinGlobalDep " + w.targetPath)
					g.hashFile(hw, "", w.targetPath)
					w.hash = hw.Sum()
				case *gobinModDep:
					logTrace("hash gobinModDep %v", w)
					w.hash = w.pkg.hash
				}
			}(w)
		}
		wg.Wait()
		logTiming("round complete %v", todoOrder)
		for _, w := range todoOrder {
			if w.Done() {
				work = append(work, g.doneDep(w)...)
			}
		}
	}
}

func (g *gg) addDepsFromImports(p *pkg) {
	seen := make(map[string]bool)
	var imports []string
	if p.isXTest && p.generate {
		imports = append(imports, p.XTestImports...)
	} else {
		imports = append(imports, p.Imports...)
		if p.generate {
			imports = append(imports, p.TestImports...)
		}
	}
	for _, ip := range imports {
		if seen[ip] || g.isSpecialImport(ip) {
			continue
		}
		dep, ok := g.pkgLookup[ip]
		if !ok {
			g.fatalf("failed to resolve import %v", ip)
		}
		g.addDep(p, dep)
		seen[ip] = true
	}
	if p.x != nil {
		g.addDepsFromImports(p.x)
	}
}

func (g *gg) newArchive() *archiveWriter {
	ar, err := newArchiveWriter(g.tempDir, "")
	if err != nil {
		g.fatalf("failed to create archive: %v", err)
	}
	return ar
}

func (g *gg) cachePutArchive(id cache.ActionID, ar *archiveWriter) error {
	if err := ar.Close(); err != nil {
		return fmt.Errorf("failed to close archive: %v", ar)
	}
	f, err := os.Open(ar.file.Name())
	if err != nil {
		return fmt.Errorf("failed to open archive %v for reading: %v", ar.file.Name(), err)
	}
	if _, _, err := g.cache.Put(id, f); err != nil {
		return fmt.Errorf("failed to write archive to cache: %v", err)
	}
	return nil
}

func (g *gg) isSpecialImport(path string) bool {
	return path == "C"
}

func (g *gg) loadMisses(pkgMisses, dirMisses missingDeps) {
	if len(pkgMisses) == 0 && len(dirMisses) == 0 {
		return
	}
	var paths []string
	patts := make(map[string]bool)
	for ip := range pkgMisses {
		patts[ip] = true
		paths = append(paths, ip)
	}
	for ip := range dirMisses {
		patts[ip] = true
		paths = append(paths, ip)
	}
	pkgs, err := g.list(paths, "-deps")
	if err != nil {
		g.fatalf("failed to load patterns [%v]: %v", strings.Join(paths, " "), err)
	}

	// We might already have loaded a pkg that is a dep of one of the misses
	// Skip those (after a quick dir check); should be a no-op

	// We also do a check to ensure that the number of packages returned with a
	// Match field is equal to the number of misses, and that furthermore the
	// misses' patterns match exactly the import path of the package returned
	// i.e. we don't support relative gobin -m -run $pkg specs.

	var newPkgs []*pkg
	for _, p := range pkgs {
		if len(p.Match) > 0 {
			if len(p.Match) != 1 {
				g.fatalf("multiple patterns resolved to %v: %v", p.ImportPath, p.Match)
			}
			if p.Match[0] != p.ImportPath {
				g.fatalf("pattern %v was not identical to import path %v", p.Match[0], p.ImportPath)
			}
			delete(patts, p.ImportPath)
		}
		if _, ok := g.pkgLookup[p.ImportPath]; ok {
			continue
		}
		newPkgs = append(newPkgs, g.newPkgImpl(p))
	}

	if len(patts) > 0 {
		var ps []string
		for p := range patts {
			ps = append(ps, p)
		}
		g.fatalf("failed to resolve patterns %v to package(s)", ps)
	}

	for _, p := range newPkgs {
		seen := make(map[string]bool)
		var imports []string
		imports = append(imports, p.Imports...)
		for _, ip := range imports {
			if seen[ip] || g.isSpecialImport(ip) {
				continue
			}
			dep, ok := g.pkgLookup[ip]
			if !ok {
				g.fatalf("failed to resolve import %v", ip)
			}
			g.addDep(p, dep)
			seen[ip] = true
		}
	}

	for ip, deps := range pkgMisses {
		np := g.pkgLookup[ip]
		if np == nil {
			g.fatalf("inconsistency in state of g.pkgLookup: could not find %v", ip)
		}
		for d := range deps {
			g.addDep(d, np)
		}
	}

	for ip, deps := range dirMisses {
		mod := g.gobinModLookup[ip]
		if mod == nil {
			g.fatalf("inconsistency in state of g.gobinModLookup: could not find %v", ip)
		}
		p := g.pkgLookup[ip]
		if p == nil {
			g.fatalf("inconsistency in state of g.pkgLookup: could not find %v", ip)
		}
		mod.pkg = p
		g.addDep(mod, p)
		for d := range deps {
			g.addDep(d, mod)
		}
	}
}

func (g *gg) refreshImports(p *pkg, misses missingDeps) {
	var xp *pkg
	xtest := p.Name + "_test"
	fset := token.NewFileSet()
	deps := make(map[string]bool)
	testdeps := make(map[string]bool)
	xtestdeps := make(map[string]bool)
	matchFile := func(fi os.FileInfo) bool {
		return imports.MatchFile(fi.Name(), g.tagsMap)
	}
	pkgs, err := parser.ParseDir(fset, p.Dir, matchFile, parser.ParseComments|parser.ImportsOnly)
	if err != nil {
		g.fatalf("failed to parse %v: %v", p.Dir, err)
	}
	keys := func() (keys []string) {
		for pn := range pkgs {
			keys = append(keys, pn)
		}
		sort.Strings(keys)
		return
	}

	// sanity check on those packages we found
	if _, ok := pkgs[p.Name]; !ok {
		g.fatalf("expected to find %v amongst %v in %v", p.Name, keys(), p.Dir)
	}
	if len(pkgs) > 2 {
		g.fatalf("found multiple packages %v in %v", keys(), p.Dir)
	} else if len(pkgs) == 2 {
		if _, ok := pkgs[xtest]; !ok {
			g.fatalf("expected to find external test %v in %v", xtest, keys())
		}
	}

	p.GoFiles = nil
	p.CgoFiles = nil
	p.TestGoFiles = nil
	p.XTestGoFiles = nil

	if _, ok := pkgs[xtest]; ok {
		// we grew an xtest
		if p.x == nil {
			p.x = &pkg{
				Package: deriveXTest(p.Package),
				depsMap: newDepsMap(),
				isXTest: true,
			}
		}
	}
	xp = p.x

	for ppn, pp := range pkgs {
		isXTest := ppn == xtest
		for _, f := range pp.Files {
			fn := filepath.Base(fset.Position(f.Pos()).Filename)
			isTest := strings.HasSuffix(fn, "_test.go")

			var b bytes.Buffer
			for _, cg := range f.Comments {
				if cg == f.Doc || cg.Pos() > f.Package {
					break
				}
				for _, c := range cg.List {
					b.WriteString(c.Text + "\n")
				}
				b.WriteString("\n")
			}
			if !imports.ShouldBuild(b.Bytes(), g.tagsMap) {
				continue
			}
			var cImport bool
			for _, i := range f.Imports {
				ip := strings.Trim(i.Path.Value, "\"")
				if ip == "C" {
					cImport = true
				}
				if isXTest {
					xtestdeps[ip] = true
				} else if isTest {
					testdeps[ip] = true
				} else {
					deps[ip] = true
				}
			}

			if isXTest {
				p.XTestGoFiles = append(p.XTestGoFiles, fn)
			} else if isTest {
				p.TestGoFiles = append(p.TestGoFiles, fn)
			} else if cImport {
				p.CgoFiles = append(p.CgoFiles, fn)
			} else {
				p.GoFiles = append(p.GoFiles, fn)
			}
		}
	}
	sort.Strings(p.GoFiles)
	sort.Strings(p.CgoFiles)
	sort.Strings(p.TestGoFiles)
	sort.Strings(p.XTestGoFiles)

	pdeps := make(map[*pkg]bool)
	xpdeps := make(map[*pkg]bool)

	p.Imports = nil
	p.TestImports = nil
	p.XTestImports = nil

	for ip := range deps {
		p.Imports = append(p.Imports, ip)
		if rp, ok := g.pkgLookup[ip]; !ok {
			if !g.isSpecialImport(ip) {
				misses.add(ip, p)
			}
		} else {
			pdeps[rp] = true
		}
	}
	for ip := range testdeps {
		p.TestImports = append(p.TestImports, ip)
		if rp, ok := g.pkgLookup[ip]; !ok {
			if !g.isSpecialImport(ip) {
				misses.add(ip, p)
			}
		} else {
			pdeps[rp] = true
		}
	}
	for ip := range xtestdeps {
		p.XTestImports = append(p.XTestImports, ip)
		if rp, ok := g.pkgLookup[ip]; !ok {
			if !g.isSpecialImport(ip) {
				misses.add(ip, xp)
			}
		} else {
			xpdeps[rp] = true
		}
	}

	sort.Strings(p.Imports)
	sort.Strings(p.TestImports)
	sort.Strings(p.XTestImports)

	for cd := range p.Deps().deps {
		if cd, ok := cd.(*pkg); ok {
			if _, isNd := pdeps[cd]; !isNd {
				g.dropDep(p, cd)
			}
		}
	}
	if xp != nil {
		for cd := range xp.Deps().deps {
			if cd, ok := cd.(*pkg); ok {
				if _, isNd := xpdeps[cd]; !isNd {
					g.dropDep(xp, cd)
				}
			}
		}
	}
	for nd := range pdeps {
		if _, isCd := p.Deps().deps[nd]; !isCd {
			g.addDep(p, nd)
		}
	}
	for nd := range xpdeps {
		if _, isCd := xp.Deps().deps[nd]; !isCd {
			g.addDep(xp, nd)
		}
	}
	// add the non-.go files (these will already be sorted by ReadDir)
	p.CFiles = nil
	p.CXXFiles = nil
	p.MFiles = nil
	p.HFiles = nil
	p.FFiles = nil
	p.SFiles = nil
	p.SwigFiles = nil
	p.SwigCXXFiles = nil
	p.SysoFiles = nil
	ls, err := ioutil.ReadDir(p.Dir)
	if err != nil {
		g.fatalf("failed to read dir %v: %v", p.Dir, err)
	}
	for _, fi := range ls {
		if fi.IsDir() {
			continue
		}
		switch filepath.Ext(fi.Name()) {
		case ".go":
			continue
		case ".c":
			p.CFiles = append(p.CFiles, fi.Name())
		case ".cc", ".cxx", ".cpp":
			p.CXXFiles = append(p.CXXFiles, fi.Name())
		case ".m":
			p.MFiles = append(p.MFiles, fi.Name())
		case ".h", ".hh", ".hpp", ".hxx":
			p.HFiles = append(p.HFiles, fi.Name())
		case ".f", ".F", ".for", ".f90":
			p.FFiles = append(p.FFiles, fi.Name())
		case ".s":
			p.SFiles = append(p.SFiles, fi.Name())
		case ".swig":
			p.SwigFiles = append(p.SwigFiles, fi.Name())
		case ".swigcxx":
			p.SwigCXXFiles = append(p.SwigCXXFiles, fi.Name())
		case ".syso":
			p.SysoFiles = append(p.SysoFiles, fi.Name())
		}
	}

	if xp != nil {
		xp.XTestGoFiles = p.XTestGoFiles
		xp.XTestImports = p.XTestImports
	}
}

func (g *gg) refreshDirectiveDeps(p *pkg, misses missingDeps) {
	if !p.generate {
		g.fatalf("inconsistency: %v is not marked for generation", p)
	}

	p.dirs = nil

	var files []string
	if p.isXTest {
		files = append(files, p.XTestGoFiles...)
	} else {
		files = append(files, p.GoFiles...)
		files = append(files, p.CgoFiles...)
		files = append(files, p.TestGoFiles...)
	}

	for _, file := range files {
		err := gogenerate.DirFunc(p.Name, p.Dir, file, func(line int, dirArgs []string) error {
			gen, err := g.resolveDir(p.Dir, dirArgs)
			if err != nil {
				return fmt.Errorf("failed to resolve directive: %v", err)
			}
			outDirs, err := parseOutDirs(p.Dir, dirArgs)
			if err != nil {
				return fmt.Errorf("failed to parse out dirs in %v:%v: %v", filepath.Join(p.Dir, file), line, err)
			}
			for i, v := range outDirs {
				if v == p.Dir {
					outDirs = append(outDirs[:i], outDirs[i+1:]...)
					break
				}
			}
			p.dirs = append(p.dirs, directive{
				pkgName: p.Name,
				file:    file,
				line:    line,
				args:    dirArgs,
				gen:     gen,
				outDirs: outDirs,
			})
			switch gen := gen.(type) {
			case *gobinModDep:
				// If this gobinModDep does not have an underlying pkg, then we haven't previously
				// seen the package's dependencies. We need to resolve the package first (next phase)
				// and then add the dependency. In this next phase we will also ensure that the
				// directive specified import path is valid, i.e. resolves to a single package
				if gen.pkg == nil {
					if _, ok := g.pkgLookup[gen.importPath]; !ok {
						misses.add(gen.importPath, p)
					}
				} else {
					if _, ok := p.Deps().deps[gen]; !ok {
						g.addDep(p, gen)
					}
				}
			default:
				if _, ok := p.Deps().deps[gen]; !ok {
					g.addDep(p, gen)
				}
			}
			return nil
		})

		if err != nil {
			g.fatalf("failed to walk %v%v%v for go:generate directives: %v", p.Dir, string(os.PathSeparator), file, err)
		}
	}

	if p.x != nil {
		g.refreshDirectiveDeps(p.x, misses)
	}
}

func (g *gg) undo(d dep) {
	var w dep
	work := []dep{d}
	for len(work) > 0 {
		w, work = work[0], work[1:]
		addWork := w.Done()
		w.Undo()
		for rd := range w.Deps().rdeps {
			rd.Deps().dirtydeps[w] = true
			if addWork {
				work = append(work, rd)
			}
		}
	}
}

func (g *gg) hashFile(hw io.Writer, dir, file string) {
	fp := filepath.Join(dir, file)
	fmt.Fprintf(hw, "file: %v\n", fp)
	f, err := os.Open(fp)
	if err != nil {
		g.fatalf("failed to open %v: %v", fp, err)
	}
	defer f.Close()
	if _, err := io.Copy(hw, f); err != nil {
		g.fatalf("failed to hash %v: %v", fp, err)
	}
}

func stringList(vs ...[]string) (res []string) {
	for _, v := range vs {
		res = append(res, v...)
	}
	return res
}

func (g *gg) hashOutDirs(outDirs []string, p *pkg, outHash io.Writer, dirNames map[string]map[generator]bool) map[string][hashSize]byte {
	fileHash := make(map[string][hashSize]byte)
	var seen map[string]bool
	if outHash != nil {
		fmt.Fprintf(outHash, "Files:\n")
		// this captures all go list known input files, including such files which
		// are generated.
		seen = make(map[string]bool)
		inputFiles := stringList(
			p.GoFiles,
			p.CgoFiles,
			p.CFiles,
			p.CXXFiles,
			p.FFiles,
			p.MFiles,
			p.HFiles,
			p.SFiles,
			p.SysoFiles,
			p.SwigFiles,
			p.SwigCXXFiles,
		)
		for _, f := range inputFiles {
			seen[f] = true
			var fhw *hash
			ws := []io.Writer{outHash}
			for dn := range dirNames {
				if gogenerate.AnyFileGeneratedBy(f, dn) {
					fhw = newHash("## file " + f)
					ws = append(ws, fhw)
				}
			}
			fw := io.MultiWriter(ws...)
			g.hashFile(fw, p.Dir, f)
			if fhw != nil {
				fileHash[filepath.Join(p.Dir, f)] = fhw.Sum()
			}
		}
		fmt.Fprintf(outHash, "Generated Files:\n")
	}
	// what's left is all generated files not in p.Dir and non-go list files
	// in p.Dir.
	for _, dir := range outDirs {
		isPkgDir := dir == p.Dir
		ls, err := ioutil.ReadDir(dir)
		if err != nil {
			g.fatalf("failed to list contents of %v: %v", dir, err)
		}
		for _, fi := range ls {
			if fi.IsDir() {
				continue
			}
			fn := fi.Name()
			if isPkgDir && seen != nil && seen[fn] {
				continue
			}
			var ws []io.Writer
			var fhw *hash
			for dn := range dirNames {
				if gogenerate.AnyFileGeneratedBy(fn, dn) {
					fhw = newHash("## file " + fn)
					ws = append(ws, fhw)
				}
			}
			if fhw == nil {
				// not generated
				continue
			}
			if outHash != nil {
				ws = append(ws, outHash)
			}
			fw := io.MultiWriter(ws...)
			g.hashFile(fw, dir, fn)
			if fhw != nil {
				fileHash[filepath.Join(dir, fn)] = fhw.Sum()
			}
		}
	}
	return fileHash
}

func (g *gg) shouldBuildFile(path string) bool {
	mf := imports.MatchFile(path, g.tagsMap)
	if !mf {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		g.fatalf("failed to open %v\n", f)
	}
	defer f.Close()
	cmts, err := imports.ReadComments(f)
	if err != nil {
		g.fatalf("failed to read comments from %v: %v", path, err)
	}
	return imports.ShouldBuild(cmts, g.tagsMap)
}

func (g *gg) hashImport(hw io.Writer, path string) {
	if g.isSpecialImport(path) {
		return
	}
	d, ok := g.pkgLookup[path]
	if !ok {
		g.fatalf("failed to resolve import %v", path)
	}
	if !d.Done() {
		g.fatalf("inconsistent state: dependency %v is not done", path)
	}
	fmt.Fprintf(hw, "import %v: %x\n", d.ImportPath, d.hash)
}

func (g *gg) hashDeps(hw io.Writer, d dep) {
	var deps []dep
	for dd := range d.Deps().deps {
		if !dd.Done() {
			g.fatalf("consistency error: dep %v is not done", dd)
		}
		deps = append(deps, dd)
	}
	sortDeps(deps)
	for _, d := range deps {
		fmt.Fprintf(hw, "%v\n", d.HashString())
	}
}

func (g *gg) fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}

func (g *gg) newPkg(p *Package) *pkg {
	res := g.newPkgImpl(p)
	return res
}

func (g *gg) newPkgImpl(p *Package) *pkg {
	if _, ok := g.pkgLookup[p.ImportPath]; ok {
		g.fatalf("tried to add pre-existing pkg %v", p.ImportPath)
	}
	if p2, ok := g.dirLookup[p.Dir]; ok {
		g.fatalf("directory overlap between packages %v and %v", p.ImportPath, p2.ImportPath)
	}
	res := &pkg{
		Package: p,
		depsMap: newDepsMap(),
	}
	if len(p.XTestGoFiles) > 0 {
		res.x = &pkg{
			Package: deriveXTest(p),
			depsMap: newDepsMap(),
			isXTest: true,
		}
	}
	g.pkgLookup[p.ImportPath] = res
	g.dirLookup[p.Dir] = res
	if p.ImportPath == "example.com/rename" {
		fmt.Printf("adding pkg dep example.com/rename\n")
	}
	return res
}

func (g *gg) resolveGobinModDep(patt string) *gobinModDep {
	if strings.Index(patt, "@") != -1 {
		g.fatalf("gobin -m directive cannot specify version: %v", patt)
	}
	// We still don't yet know that patt is a valid import path.
	// We assume it is and then check later when if we need to
	// resolve a missing package path.
	ip := patt
	if mod, ok := g.gobinModLookup[ip]; ok {
		return mod
	}
	d := &gobinModDep{
		importPath: ip,
		pkg:        g.pkgLookup[patt],
		depsMap:    newDepsMap(),
	}
	g.gobinModLookup[ip] = d
	return d
}

func (g *gg) resolveGobinGlobalDep(patt string) *gobinGlobalDep {
	if glob, ok := g.gobinGlobalCache[g.gobinGlobalLookup[patt]]; ok {
		return glob
	}
	// we need to use gobin -p to resolve patt
	cmd := exec.Command("gobin", "-p", patt)
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if err, ok := err.(*exec.ExitError); ok {
			stderr = err.Stderr
		}
		g.fatalf("failed to resolve gobin global pattern via %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr)
	}

	target := strings.TrimSpace(string(out))

	if glob, ok := g.gobinGlobalCache[target]; ok {
		g.gobinGlobalLookup[patt] = target
		return glob
	}

	d := &gobinGlobalDep{
		targetPath: target,
		commandDep: &commandDep{
			name:    path.Base(target),
			depsMap: newDepsMap(),
		},
	}
	g.gobinGlobalLookup[patt] = target
	g.gobinGlobalCache[target] = d
	return d
}

func (g *gg) resolveCommandDep(name string) *commandDep {
	if comm, ok := g.commLookup[name]; ok {
		return comm
	}
	d := &commandDep{
		name:    name,
		depsMap: newDepsMap(),
	}
	g.commLookup[name] = d
	return d
}

var _ dep = (*pkg)(nil)

func (g *gg) resolveDir(dir string, dirArgs []string) (generator, error) {
	switch dirArgs[0] {
	case "gobin":
		args := dirArgs[1:]
		mainMod, patt, err := gobinParse(args)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gobin args %v: %v", args, err)
		}
		if mainMod {
			return g.resolveGobinModDep(patt), nil
		} else {
			return g.resolveGobinGlobalDep(patt), nil
		}
	case "go":
		return nil, fmt.Errorf("do not yet know how to handle go command-based directives")
	default:
		return g.resolveCommandDep(dirArgs[0]), nil
	}
}
