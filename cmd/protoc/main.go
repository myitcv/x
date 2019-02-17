// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"myitcv.io/gogenerate"
)

const (
	// this constant needs to be kept in sync with the go:generate directive below
	protobufVersion = "v3.6.1"
)

//go:generate gobin -m -run myitcv.io/cmd/helpflagtopkgdoc

// This directive needs to be kept in sync with the constant above
//go:generate gobin -m -run myitcv.io/cmd/protoc/internal/getprotoc v3.6.1

//go:generate gobin -m -run github.com/jteeuwen/go-bindata/go-bindata -o gen_protoczip_go-bindata_${GOOS}_${GOARCH}.go downloads/$GOOS/$GOARCH

//go:generate gofmt -w gen_protoczip_go-bindata_${GOOS}_${GOARCH}.go

type valsFlag struct {
	vals []string
}

func (e *valsFlag) String() string {
	return fmt.Sprintf("%v", e.vals)
}

func (e *valsFlag) Set(v string) error {
	e.vals = append(e.vals, v)
	return nil
}

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
		if ee, ok := err.(*exec.ExitError); ok {
			return ExitCode(ee.ProcessState)
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
}

func mainerr() (retErr error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		mainUsage(os.Stderr)
	}
	var ipkgs valsFlag
	var idirsVals valsFlag
	var infiles valsFlag
	fGoOut := fs.String("go_out", "", "C++ protoc define --go_out flag")
	fs.Var(&infiles, gogenerate.FlagInFilesPrefix+"input", "flag for input files")
	fs.Var(&ipkgs, "Ipkg", "Go package path equilvane to C++ protoc -I flag")
	fs.Var(&idirsVals, "I", "Directories to pass through as -I flag values")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	var files []string
	files = append(files, fs.Args()...)
	files = append(files, infiles.vals...)

	if len(files) == 0 {
		return fmt.Errorf("no .proto files to compile")
	}

	for _, f := range files {
		if !strings.HasSuffix(f, ".proto") {
			return fmt.Errorf("don't know how to handle file named %v; expected .proto file", f)
		}
		if _, err := os.Stat(f); err != nil {
			return err
		}
	}

	idirs := idirsVals.vals
	if len(ipkgs.vals) > 0 {
		cmd := exec.Command("go", "list", "-f={{.Dir}}")
		for _, p := range ipkgs.vals {
			cmd.Args = append(cmd.Args, p)
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to resolve package paths via %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.Bytes())
		}
		idirs = append(idirs, strings.Split(strings.TrimSpace(stdout.String()), "\n")...)
	}

	td, err := ioutil.TempDir("", "protoc-temp-path")
	if err != nil {
		return fmt.Errorf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(td)

	zipfn := path.Join("downloads", runtime.GOOS, runtime.GOARCH, protobufVersion+".zip")

	zipc, err := Asset(zipfn)
	if err != nil {
		return fmt.Errorf("failed to find %v: %v", zipfn, err)
	}

	r, err := zip.NewReader(bytes.NewReader(zipc), int64(len(zipc)))
	if err != nil {
		return fmt.Errorf("failed to unzip %v: %v", zipfn, err)
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		fn := filepath.FromSlash(f.Name)
		if filepath.IsAbs(fn) {
			return fmt.Errorf("protoc zip %v has absolute file path %v", zipfn, f.Name)
		}
		fn = filepath.Join(td, fn)

		if err := os.MkdirAll(filepath.Dir(fn), 0777); err != nil {
			return fmt.Errorf("failed to create directory for %v: %v", fn, err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open f.Name: %v", err)
		}
		f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			return fmt.Errorf("failed to create %v: %v", fn, err)
		}
		if _, err = io.Copy(f, rc); err != nil {
			return fmt.Errorf("failed to write to %v: %v", fn, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close %v: %v", fn, err)
		}
		rc.Close()
	}

	tdbin := filepath.Join(td, "bin")
	if err := installProtoGenGo(tdbin); err != nil {
		return fmt.Errorf("failed to install protoc-gen-go to %v: %v", tdbin, err)
	}

	cmd := exec.Command(filepath.Join(tdbin, "protoc"))
	cmd.Args[0] = "protoc"
	cmd.Env = append(os.Environ(),
		"PATH="+tdbin+string(filepath.ListSeparator)+os.Getenv("PATH"),
	)
	if *fGoOut != "" {
		cmd.Args = append(cmd.Args, "--go_out="+*fGoOut)
	}
	for _, d := range idirs {
		cmd.Args = append(cmd.Args, "-I="+d)
	}
	cmd.Args = append(cmd.Args, files...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee
		}
		return fmt.Errorf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
	}

	gooutParts := strings.Split(*fGoOut, ":")
	outDir := gooutParts[len(gooutParts)-1]
	if outDir == "" {
		outDir = "."
	}

	// rename the output files
	// TODO we are assuming writing to the current directory here
	for _, f := range files {
		if !strings.HasSuffix(f, ".proto") {
			return fmt.Errorf("don't know how to handle output from file %v", f)
		}
		f = strings.TrimSuffix(filepath.Base(f), ".proto")
		of := filepath.Join(outDir, f+".pb.go")
		f = strings.TrimPrefix(f, "gen_")
		nf := filepath.Join(outDir, "gen_"+f+"_protoc.go")
		if err := os.Rename(of, nf); err != nil {
			return fmt.Errorf("failed to rename %v to %v: %v", of, nf, err)
		}
		cmd := exec.Command("gofmt", "-w", nf)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
		}
	}

	return nil
}

func installProtoGenGo(dir string) error {
	cmd := exec.Command("gobin", "-m", "-p", "github.com/golang/protobuf/protoc-gen-go")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.Bytes())
	}

	path := strings.TrimSpace(stdout.String())

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %v: %v", path, err)
	}
	defer f.Close()

	target := filepath.Join(dir, "protoc-gen-go")

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %v: %v", path, err)
	}

	tf, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		return fmt.Errorf("failed to create %v: %v", target, err)
	}

	if _, err := io.Copy(tf, f); err != nil {
		return fmt.Errorf("failed to copy from %v to %v: %v", path, target, err)
	}

	if err := tf.Close(); err != nil {
		return fmt.Errorf("failed to close %v: %v", target, err)
	}

	return nil
}
