// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"myitcv.io/gogenerate"
)

const (
	immutableGenCmd           = "immutableGen"
	immutableGenCmdImportPath = "myitcv.io/immutable/cmd/immutableGen"
)

var (
	fGoGenCmds   gogenCmds
	fLicenseFile = gogenerate.LicenseFileFlag(flag.CommandLine)
	fGoGenLog    = gogenerate.LogFlag(flag.CommandLine)
	fDebug       = flag.Bool("debug", false, "print debug messages")
)

const (
	debug = false
)

func init() {
	flag.Var(&fGoGenCmds, "G", "Path to search for imports (flag can be used multiple times)")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix(immutableGenCmd + ": ")

	gogenerate.DefaultLogLevel(fGoGenLog, gogenerate.LogFatal)

	envFile, ok := os.LookupEnv(gogenerate.GOFILE)
	if !ok {
		fatalf("env not correct; missing %v", gogenerate.GOFILE)
	}

	envPkgName, ok := os.LookupEnv(gogenerate.GOPACKAGE)
	if !ok {
		fatalf("env not correct; missing %v", gogenerate.GOPACKAGE)
	}

	wd, err := os.Getwd()
	if err != nil {
		fatalf("unable to get working directory: %v", err)
	}

	tags := make(map[string]bool)

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	tags[goos] = true

	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	tags[goarch] = true

	pathDirFiles, err := gogenerate.FilesContainingCmd(wd, immutableGenCmdImportPath, tags)
	if err != nil {
		fatalf("could not determine if we are the first file: %v", err)
	}

	if pathDirFiles == nil {
		fatalf("cannot find any files containing the %v or %v directive", immutableGenCmdImportPath, immutableGenCmd)
	}

	if pathDirFiles[envFile] > 1 {
		fatalf("expected a single occurrence of %v directive in %v. Got: %v", immutableGenCmdImportPath, envFile, pathDirFiles)
	}

	licenseHeader, err := gogenerate.CommentLicenseHeader(fLicenseFile)
	if err != nil {
		fatalf("could not comment license file: %v", err)
	}

	execute(wd, envPkgName, licenseHeader, fGoGenCmds)
}

func debugf(format string, args ...interface{}) {
	if debug || *fDebug {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

func fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}

func infoln(args ...interface{}) {
	if *fGoGenLog == string(gogenerate.LogInfo) {
		log.Println(args...)
	}
}

func infof(format string, args ...interface{}) {
	if *fGoGenLog == string(gogenerate.LogInfo) {
		log.Printf(format, args...)
	}
}
