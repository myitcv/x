package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/goproxytest"
	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

var (
	proxyURL  string
	gobinPath string

	fCommandTestSleep = flag.Float64("commandTestSleep", 0.0, "length of time to sleep in command test")
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(ggMain{m}, map[string]func() int{
		"gg": main1,
	}))
}

type ggMain struct {
	m *testing.M
}

func (m ggMain) Run() int {
	// Start the Go proxy server running for all tests.
	srv, err := goproxytest.NewServer("testdata/mod", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot start proxy: %v", err)
		return 1
	}
	proxyURL = srv.URL

	td, err := installGoWrapper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to install go wrapper: %v\n", err)
		return 1
	}
	defer os.RemoveAll(td)

	cmd := exec.Command("go", "install", "github.com/myitcv/gobin")
	cmd.Env = append(os.Environ(), "GOBIN="+td)

	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
		return 1
	}

	gobinPath = filepath.Join(td, "gobin")

	return m.m.Run()
}

func TestScripts(t *testing.T) {
	p := testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"rmglob": rmglob,
		},
		Setup: func(e *testscript.Env) error {
			var newEnv []string
			var work string
			var path string
			for _, e := range e.Vars {
				switch {
				case strings.HasPrefix(e, "PATH="):
					path = e
				case strings.HasPrefix(e, "WORK="):
					work = e
					fallthrough
				default:
					newEnv = append(newEnv, e)
				}
			}
			path = strings.TrimPrefix(path, "PATH=")
			work = strings.TrimPrefix(work, "WORK=")
			home := filepath.Join(work, "home")
			gopath := filepath.Join(home, "gopath")
			e.Vars = newEnv
			pathVals := []string{
				filepath.Dir(gobinPath),
				filepath.Join(gopath, "bin"),
				path,
			}
			e.Vars = append(e.Vars,
				"HOME="+home,
				"GOPATH="+gopath,
				"PATH="+strings.Join(pathVals, string(os.PathListSeparator)),
				"GOPROXY="+proxyURL,
				"GOBINPATH="+gobinPath,
				"COMMANDTESTSLEEP="+fmt.Sprintf("%.2f", *fCommandTestSleep),
			)

			if build.Default.ReleaseTags[len(build.Default.ReleaseTags)-1] == "go1.11" {
				e.Vars = append(e.Vars, "GO111=go1.11")
			}

			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}

func rmglob(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("rmglob does not support negation")
	}
	for _, g := range args {
		ag := ts.MkAbs(g)
		matches, err := filepath.Glob(ag)
		if err != nil {
			ts.Fatalf("failed to glob %v: %v", ag, err)
		}
		for _, m := range matches {
			if err := os.Remove(m); err != nil {
				ts.Fatalf("failed to remove: %v: %v", m, err)
			}
		}
	}
}
