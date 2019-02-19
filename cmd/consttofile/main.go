package main

import (
	"flag"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	os.Exit(main1())
}

func main1() int {
	switch err := mainerr(); err {
	case nil:
		return 0
	case flag.ErrHelp:
		return 2
	default:
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
}

// TODO take into account build tags more explicitly

func mainerr() (retErr error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		mainUsage(os.Stderr)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("consttofile takes at least one argument")
	}

	envPkg := os.Getenv("GOPACKAGE")

	config := &packages.Config{
		Mode:  packages.LoadSyntax,
		Fset:  token.NewFileSet(),
		Tests: true,
	}

	pkgs, err := packages.Load(config, ".")
	if err != nil {
		return fmt.Errorf("could not load package from current dir: %v", err)
	}

	forTest := regexp.MustCompile(` \[[^\]]+\]$`)

	testPkgs := make(map[string]*packages.Package)
	var nonTestPkg *packages.Package

	// Becase of https://github.com/golang/go/issues/27910 we have to
	// apply some janky logic to find the "right" package
	for _, p := range pkgs {
		switch {
		case strings.HasSuffix(p.PkgPath, ".test"):
			// we don't ever want this package
			continue
		case forTest.MatchString(p.ID):
			testPkgs[p.Name] = p
		default:
			nonTestPkg = p
		}
	}

	ids := func() []string {
		var ids []string
		for _, p := range pkgs {
			ids = append(ids, p.ID)
		}
		sort.Strings(ids)
		return ids
	}

	if nonTestPkg == nil {
		return fmt.Errorf("always expect to have the actual package. Got %v", ids())
	}

	var pkg *packages.Package

	if strings.HasSuffix(envPkg, "_test") {
		if pkg = testPkgs[envPkg]; pkg == nil {
			return fmt.Errorf("called with package name %v, but go/packages did not give us such a package. Got %v", envPkg, ids())
		}
	} else {
		if pkg = testPkgs[envPkg]; pkg == nil {
			pkg = nonTestPkg
		}
	}

	for _, cn := range fs.Args() {
		co := pkg.Types.Scope().Lookup(cn)
		if co == nil {
			return fmt.Errorf("failed to find const %v\n", cn)
		}

		c, ok := co.(*types.Const)
		if !ok {
			return fmt.Errorf("found %v, but it was not a const, instead it was a %T", cn, co)
		}

		if c.Val().Kind() != constant.String {
			return fmt.Errorf("expected %v to be a string constant; got %v", cn, c.Val().Kind())
		}

		i := strings.LastIndex(cn, "_")
		if i == -1 || i == len(cn)-1 {
			return fmt.Errorf("constant %v does not specifcy an extension", cn)
		}

		fn := "gen_" + cn[:i] + "_consttofile" + "." + cn[i+1:]

		if err := ioutil.WriteFile(fn, []byte(constant.StringVal(c.Val())), 0666); err != nil {
			return fmt.Errorf("failed to write to %v: %v", fn, err)
		}
	}

	return nil
}
