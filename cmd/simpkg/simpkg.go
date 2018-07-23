// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	_log "log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

var log = _log.New(os.Stdout, "", 0)
var elog = _log.New(os.Stderr, "", _log.Lshortfile)

const VENDOR_FILE = "vendor.json"

func usage() {
	log.Printf("%v: \n", os.Args[0])
	log.Println()
	log.Println("  init:    ensure that vendor.json exists within first element of $GOPATH")
	log.Println("  get:     call `go get` with any following args")
	log.Println("  list:    call `go list ./...` with any following args within first element of $GOPATH")
	log.Println("  reset:   reset the contents of $GOPATH[0]/src using the contents of $GOPATH[0]/vendor.json as the pinning reference")
	log.Println("  update:  update $GOPATH[0]/vendor.json with the packages it contains at their commit id")
	log.Println()
	os.Exit(1)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		usage()
	}

	switch args[0] {
	case "get":
		cmd := exec.Command("go", args...)
		setGOPATH(cmd)
		outB, err := cmd.CombinedOutput()
		out := string(outB)

		if err != nil {
			elog.Fatalf(out)
		}

		log.Print(out)
	case "init":
		gopath := getGOPATH()
		target := filepath.Join(gopath, VENDOR_FILE)
		_, err := vendorFile()
		if err != nil {
			gpfi, gperr := os.Stat(gopath)

			if gperr != nil || !gpfi.IsDir() {
				elog.Fatalf("GOPATH %v does not exist as a directory\n", gopath)
			} else if err == os.ErrNotExist {
				log.Printf("Creating %v\n", target)
				_, err := os.Create(target)
				if err != nil {
					elog.Fatal(err)
				}
			} else if err != nil {
				elog.Fatal(err)
			}

			_, err = os.Open(target)
		} else {
			log.Printf("No action required: %v already exists\n", target)
		}
	case "list":
		out := list(args[1:])
		log.Print(out)

	case "update":
		// TODO make more efficient
		var pkgs []string
		gopath := getGOPATH()
		target := filepath.Join(gopath, VENDOR_FILE)

		_, err := vendorFile()
		if err != nil {
			elog.Fatalf("vendor file %v does not exist\n", target)
		}

		buf := bytes.NewBuffer([]byte(list([]string{"./..."})))
		sc := bufio.NewScanner(buf)
		for sc.Scan() {
			pkgs = append(pkgs, sc.Text())
		}
		pkgMap := make(map[string]string)
		for _, p := range pkgs {
			vcs, err := VCSForImportPath(p)
			if err != nil {
				elog.Fatalf("We got an error: %v\n", err)
			}
			id, err := vcs.identify(filepath.Join(gopath, "src", p))
			if err != nil {
				elog.Fatalf("Got an error in identity: %v\n", err)
			}
			pkgMap[p] = id
		}
		byts, err := json.MarshalIndent(pkgMap, "", "  ")
		if err != nil {
			elog.Fatal(err)
		}
		err = ioutil.WriteFile(target, byts, 0644)
		if err != nil {
			elog.Fatal(err)
		}

	case "reset":
		gopath := getGOPATH()
		target := filepath.Join(gopath, VENDOR_FILE)
		fi, err := vendorFile()
		if err != nil {
			elog.Fatalf("vendor file %v does not exist\n", target)
		}
		byts, err := ioutil.ReadAll(fi)
		if err != nil {
			elog.Fatalf("Unable to read from %v: %v\n", target, err)
		}
		pkgMap := make(map[string]string)
		err = json.Unmarshal(byts, &pkgMap)
		if err != nil {
			elog.Fatalf("Unable to read package map from %v: %v\n", target, err)
		}
		for p, id := range pkgMap {
			vcs, err := VCSForImportPath(p)
			if err != nil {
				elog.Fatalf("Could not get VCS: %v\n", err)
			}

			// TODO optimise the gets... we need only get -u
			// once on a given repo

			// actId, err := vcs.identify(filepath.Join(gopath, "src", p))
			_, err = vcs.identify(filepath.Join(gopath, "src", p))
			if err != nil {
				elog.Fatalf("Package did not exist even after update fetch: %v\n", err)
			}

			log.Printf("Dir: %v\n", filepath.Join(gopath, "src", p))
			_, err = vcs.syncCommitish(filepath.Join(gopath, "src", p), id)
			if err != nil {
				out, err := vcs.fetch(filepath.Join(gopath, "src", p))
				if err != nil {
					elog.Fatalf("Could not fetch %v: %v\n", p, out)
				}
				out, err = vcs.syncCommitish(filepath.Join(gopath, "src", p), id)
				if err != nil {
					elog.Fatalf("Could not sync %v to %v: %v\n", p, id, out)
				}
			}

			// log.Printf("Reset %v => %v (was %v)\n", p, id, actId)
		}

	default:
		usage()
	}
}

func vendorFile() (*os.File, error) {
	gopath := getGOPATH()
	target := filepath.Join(gopath, VENDOR_FILE)

	return os.Open(target)
}

func list(args []string) string {
	gopath := getGOPATH()
	err := os.Chdir(gopath)
	if err != nil {
		elog.Fatal(err)
	}

	args = append([]string{"list"}, args...)

	cmd := exec.Command("go", args...)
	setGOPATH(cmd)
	outB, err := cmd.CombinedOutput()
	out := string(outB)

	if err != nil {
		elog.Fatal(out)
	}

	return out
}

func getGOPATH() string {
	gopath := os.Getenv("GOPATH")
	gopath = strings.Split(gopath, ":")[0]
	if !path.IsAbs(gopath) {
		elog.Fatalf("GOPATH %v is not absolute\n", gopath)
	}
	return gopath
}

func setGOPATH(c *exec.Cmd) {
	// sets the GOPATH for c to be the first entry
	// in the current value of GOPATH
	gopath := getGOPATH()

	env := os.Environ()
	for i, v := range env {
		if strings.HasPrefix(v, "GOPATH=") {
			env[i] = "GOPATH=" + gopath
		}
	}
}
