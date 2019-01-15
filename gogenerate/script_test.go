package gogenerate

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

var gobinPath string

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(gogenMain{m}, map[string]func() int{
		"cmd": func() int {
			flag.Parse()

			tagsM := make(map[string]bool, 0)
			if flag.Arg(2) != "" {
				tags := strings.Split(flag.Arg(2), ",")
				for _, t := range tags {
					tagsM[t] = true
				}
			}

			out, err := FilesContainingCmd(flag.Arg(0), flag.Arg(1), tagsM)
			if err != nil {
				fmt.Fprintf(os.Stderr, "unable to run FilesContainingCmd: %v", err)
				return 1
			}

			names := []string{}
			for name, _ := range out {
				names = append(names, name)
			}

			sort.SliceStable(names, func(i, j int) bool { return names[i] < names[j] })
			for _, n := range names {
				fmt.Fprintf(os.Stdout, "%v: %v\n", n, out[n])
			}

			return 0
		},
	}))
}

type gogenMain struct {
	m *testing.M
}

func (m gogenMain) Run() int {
	// create a temp dir for installing gobin
	tmp, err := ioutil.TempDir("", "gogen_gobin_tmp_")
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create temp dir for gobin install: %v", err)
		return 1
	}

	defer func() {
		os.RemoveAll(tmp)
	}()

	cmd := exec.Command("go", "install", "github.com/myitcv/gobin")
	cmd.Env = append(os.Environ(), "GOBIN="+tmp)

	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
		return 1
	}

	gobinPath = filepath.Join(tmp, "gobin")

	return m.m.Run()
}

func TestScripts(t *testing.T) {
	params := testscript.Params{
		Dir: "testdata",
		Setup: func(e *testscript.Env) error {
			var newEnv []string
			var path string

			for _, e := range e.Vars {
				if strings.HasPrefix(e, "PATH=") {
					path = e
				} else {
					newEnv = append(newEnv, e)
				}
			}
			e.Vars = newEnv
			path = strings.TrimPrefix(path, "PATH=")
			e.Vars = append(e.Vars,
				"PATH="+filepath.Dir(gobinPath)+string(os.PathListSeparator)+path,
				"GOBINPATH="+gobinPath,
			)

			return nil
		},
	}

	if err := gotooltest.Setup(&params); err != nil {
		t.Fatal(err)
	}

	testscript.Run(t, params)
}
