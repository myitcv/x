package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	expDirName = ".exp"
)

func TestMain(m *testing.M) {
	tf, err := ioutil.TempFile("", "gg_test_bin*")
	if err != nil {
		panic(fmt.Errorf("failed to create temp file: %v", err))
	}

	cmd := exec.Command("go", "build", "-o", tf.Name(), "myitcv.io/cmd/gg")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out))
	}

	defer func() {
		os.Remove(tf.Name())
	}()

	if err := os.Setenv("GG_TEST_BINARY", tf.Name()); err != nil {
		panic(fmt.Errorf("failed to set GG_TEST_BINARY: %v", err))
	}

	m.Run()
}

type testggData struct {
	t      *testing.T
	binary string

	td  string
	env []string

	dir string
}

func testgg(t *testing.T, dir string) *testggData {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failed to get cwd: %v", err))
	}

	td := filepath.Join(wd, "testdata", dir)

	res := &testggData{
		t:      t,
		binary: os.Getenv("GG_TEST_BINARY"),
		env:    os.Environ(),

		td:  td,
		dir: td,
	}

	res.setenv("GOPATH", td)
	res.updateenv("PATH", func(v string) string {
		return filepath.Join(td, "bin") + string(filepath.ListSeparator) + v
	})

	return res
}

func (tg *testggData) unsetenv(key string) {
	tg.t.Helper()
	for i := range tg.env {
		if strings.HasPrefix(tg.env[i], key+"=") {
			tg.env = append(tg.env[:i], tg.env[i+1:]...)
			break
		}
	}
}

func (tg *testggData) setenv(key, val string) {
	tg.t.Helper()
	tg.unsetenv(key)
	tg.env = append([]string{key + "=" + val}, tg.env...)
}

func (tg *testggData) updateenv(key string, upd func(v string) string) {
	tg.t.Helper()
	for i := range tg.env {
		if strings.HasPrefix(tg.env[i], key+"=") {
			newVal := upd(strings.TrimPrefix(tg.env[i], key+"="))
			newEnv := append(tg.env[:i], key+"="+newVal)
			newEnv = append(newEnv, tg.env[i+1:]...)
			tg.env = newEnv
			break
		}
	}
}

func (tg *testggData) clean() {
	tg.t.Helper()
	// remove all generated files
	filepath.Walk(tg.td, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if info.Name() == expDirName {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(info.Name(), "gen_") && strings.HasSuffix(info.Name(), ".go") {
			return os.Remove(path)
		}

		return nil
	})
}

func (tg *testggData) pd() string {
	tg.t.Helper()
	return filepath.Join(tg.td, "src", "p.com")
}

func (tg *testggData) setdir(dir string) {
	tg.t.Helper()
	tg.dir = dir
}

func (tg *testggData) run(args ...string) {
	tg.t.Helper()
	cmd := exec.Command(tg.binary, args...)
	cmd.Env = tg.env
	cmd.Dir = tg.dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		tg.t.Fatalf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}
	fmt.Printf("%s\n", out)
}

func (tg *testggData) ensure(dir string) {
	tg.t.Helper()
	read := func(dir string) []string {
		var res []string
		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			tg.t.Fatalf("failed to read dir %v: %v", dir, err)
		}
		for _, fi := range fis {
			if strings.HasSuffix(fi.Name(), ".go") {
				res = append(res, filepath.Join(dir, fi.Name()))
			}
		}
		return res
	}

	got := read(dir)
	want := read(filepath.Join(dir, expDirName))

	if len(want) != len(got) {
		tg.t.Fatalf("want %v, got %v", want, got)
	}

	for i := range want {
		bwant := filepath.Base(want[i])
		bgot := filepath.Base(got[i])
		if bwant != bgot {
			tg.t.Fatalf("want %v, got %v", bwant, bgot)
		}

		cmd := exec.Command("diff", "-wu", want[i], got[i])
		out, err := cmd.CombinedOutput()
		if err != nil {
			tg.t.Fatal(string(out))
		}
	}
}

func (tg *testggData) teardown() {
}
