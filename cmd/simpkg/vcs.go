// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

// borrowed from https://github.com/9466/godep/blob/master/vcs.go

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/tools/go/vcs"
)

type VCS struct {
	vcs *vcs.Cmd

	FetchCmd         string
	IdentifyCmd      string
	CommitishSyncCmd string
}

var vcsBzr = &VCS{
	vcs: vcs.ByCmd("bzr"),

	IdentifyCmd: "version-info --custom --template {revision_id}",
}

var vcsGit = &VCS{
	vcs: vcs.ByCmd("git"),

	FetchCmd:         "fetch --all",
	IdentifyCmd:      "rev-parse HEAD",
	CommitishSyncCmd: "checkout {commitish}",
}

var vcsHg = &VCS{
	vcs: vcs.ByCmd("hg"),

	IdentifyCmd: "identify --id --debug",
}

var cmd = map[*vcs.Cmd]*VCS{
	vcsBzr.vcs: vcsBzr,
	vcsGit.vcs: vcsGit,
	vcsHg.vcs:  vcsHg,
}

func VCSFromDir(dir, srcRoot string) (*VCS, string, error) {
	vcscmd, reporoot, err := vcs.FromDir(dir, srcRoot)
	if err != nil {
		return nil, "", err
	}
	vcsext := cmd[vcscmd]
	if vcsext == nil {
		return nil, "", fmt.Errorf("%s is unsupported: %s", vcscmd.Name, dir)
	}
	return vcsext, reporoot, nil
}

func VCSForImportPath(importPath string) (*VCS, error) {
	rr, err := vcs.RepoRootForImportPath(importPath, false)
	if err != nil {
		return nil, err
	}
	vcs := cmd[rr.VCS]
	if vcs == nil {
		return nil, fmt.Errorf("%s is unsupported: %s", rr.VCS.Name, importPath)
	}
	return vcs, nil
}

func (v *VCS) fetch(dir string) (string, error) {
	out, err := v.runOutput(dir, v.FetchCmd)
	return string(bytes.TrimSpace(out)), err
}

func (v *VCS) identify(dir string) (string, error) {
	out, err := v.runOutput(dir, v.IdentifyCmd)
	return string(bytes.TrimSpace(out)), err
}

func (v *VCS) syncCommitish(dir, commitish string) (string, error) {
	out, err := v.runOutput(dir, v.CommitishSyncCmd, "commitish", commitish)
	return string(bytes.TrimSpace(out)), err
}

func (v *VCS) runOutput(dir string, cmdline string, kv ...string) ([]byte, error) {
	return v.run1(dir, cmdline, kv, true)
}

// runOutputVerboseOnly is like runOutput but only generates error output to standard error in verbose mode.
func (v *VCS) runOutputVerboseOnly(dir string, cmdline string, kv ...string) ([]byte, error) {
	return v.run1(dir, cmdline, kv, false)
}

// run1 is the generalized implementation of run and runOutput.
func (v *VCS) run1(dir string, cmdline string, kv []string, verbose bool) ([]byte, error) {
	m := make(map[string]string)
	for i := 0; i < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	args := strings.Fields(cmdline)
	for i, arg := range args {
		args[i] = expand(m, arg)
	}

	_, err := exec.LookPath(v.vcs.Cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "godep: missing %s command.\n", v.vcs.Name)
		return nil, err
	}

	cmd := exec.Command(v.vcs.Cmd, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	out := buf.Bytes()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "# cd %s; %s %s\n", dir, v.vcs.Cmd, strings.Join(args, " "))
			os.Stderr.Write(out)
		}
		return nil, err
	}
	return out, nil
}

func expand(m map[string]string, s string) string {
	for k, v := range m {
		s = strings.Replace(s, "{"+k+"}", v, -1)
	}
	return s
}
