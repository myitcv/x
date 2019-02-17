package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rogpeppe/go-internal/semver"
)

var (
	goosGoarchMap = map[string]string{
		"linux/amd64":  "linux-x86_64",
		"darwin/amd64": "osx-x86_64",
	}
)

//go:generate gobin -m -run myitcv.io/cmd/helpflagtopkgdoc

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

func mainerr() (retErr error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		mainUsage(os.Stderr)
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if len(fs.Args()) != 1 || !semver.IsValid(fs.Arg(0)) {
		return fmt.Errorf("expected a single semver version argument")
	}

	version := fs.Arg(0)
	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	osarch, ok := goosGoarchMap[goos+"/"+goarch]
	if !ok {
		return fmt.Errorf("no defined target for GOOS %v and GOARCH %v", goos, goarch)
	}
	url := fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/%v/protoc-%v-%v.zip", version, version[1:], osarch)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get %v: %v", url, err)
	}
	defer resp.Body.Close()

	targetdir := filepath.Join("downloads", goos, goarch)
	if err := os.MkdirAll(targetdir, 0777); err != nil {
		return fmt.Errorf("failed to mkdir %v: %v", targetdir, err)
	}
	targetfilename := filepath.Join(targetdir, version+".zip")
	file, err := os.Create(targetfilename)
	if err != nil {
		return fmt.Errorf("failed to create %v: %v", targetfilename, err)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to save %v to %v: %v", url, targetfilename, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close %v: %v", targetfilename, err)
	}

	return nil
}
