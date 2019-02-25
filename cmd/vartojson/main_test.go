package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"vartojson": main1,
	}))
}

func TestScripts(t *testing.T) {
	tmp, err := ioutil.TempDir("", "vartojson_gobin_tmp")
	if err != nil {
		t.Fatalf("unable to create temp dir for vartojson install: %v", err)
	}
	defer os.RemoveAll(tmp)

	cmd := exec.Command("go", "install")
	cmd.Env = append(os.Environ(), "GOBIN="+tmp)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("unable to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}

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
				"PATH="+tmp+string(os.PathListSeparator)+path,
				"GENPATH="+filepath.Join(tmp, "vartojson"),
			)
			return nil
		},
	}
	if err := gotooltest.Setup(&params); err != nil {
		t.Fatal(err)
	}
	// run scripts in a subtest so the call to t.Parallel() within a t.Run in
	// testscript does not make the testscript t.Run's subtest a parallel
	// subtest of TestScripts (which would cause TestScripts to return and hence
	// defer before its parallel subtests have completed)
	t.Run("testscript", func(t *testing.T) {
		testscript.Run(t, params)
	})
}
