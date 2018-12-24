package main

import (
	"flag"
	"fmt"
)

func gobinParse(args []string) (mainMod bool, patt string, err error) {
	fs := flag.NewFlagSet("gobin", 0)
	fs.BoolVar(&mainMod, "m", false, "resolve dependencies via the main module (as given by go env GOMOD)")
	fs.String("mod", "", "provide additional control over updating and use of go.mod")
	run := fs.Bool("run", false, "run the provided main package")
	fs.Bool("p", false, "print gobin install cache location for main packages")
	fs.Bool("v", false, "print the module path and version for main packages")
	fs.Bool("d", false, "stop after installing main packages to the gobin install cache")
	fs.Bool("u", false, "check for the latest tagged version of main packages")
	fs.Bool("nonet", false, "prevent network access")
	fs.Bool("debug", false, "print debug information")

	if err = fs.Parse(args); err != nil {
		err = fmt.Errorf("failed to parse gobin flags: %v", err)
		return
	}

	if !*run {
		err = fmt.Errorf("gobin args did not specify -run")
		return
	}

	gbargs := fs.Args()
	if len(args) == 0 {
		err = fmt.Errorf("failed to parse main package[@version]")
	}

	patt = gbargs[0]
	return
}
