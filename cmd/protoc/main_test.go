// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
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
		"protoc": main1,
	}))
}

func TestScripts(t *testing.T) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("failed to get UserCacheDir: %v", err)
	}

	cmd := exec.Command("go", "env", "GOPATH", "GOMOD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected output from %v: %q", strings.Join(cmd.Args, " "), out)
	}
	gopath, gomod := lines[0], filepath.Dir(lines[1])

	p := testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Vars = append(env.Vars,
				"PROTOCCACHE="+filepath.Join(ucd, "protoc-test-cache"),
				"MAINMOD="+gomod,
				"GOPATH="+gopath,
			)
			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	t.Run("runscripts", func(t *testing.T) {
		testscript.Run(t, p)
	})
}
