// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

var fP = flag.Uint64("p", 0, "the PID of the process to walk for bash sub-processes")

const (
	NodeVersion     = "NODEVERSION"
	GoVersion       = "GOVERSION"
	MustChangeToDir = "MUST_CHANGE_TO_DIR"
)

// this is pretty horrendous code...

func main() {
	flag.Parse()

	runtime.LockOSThread()

	// we need to fail gracefully because the output (and exit code) from this
	// program will not be seen

	// if we have any remaining args they will be treated as the input to exec
	if len(flag.Args()) == 0 {
		return
	}

	if *fP != 0 {
		// we need to try and find the first bash child process of the process
		// and get the cwd of the process. If if is an xterm there will be just
		// one bash process. If it's chrome, for example, there will be none.
		// So we literally take the first one if there is one.
		bestPid := uint64(0)

		cmd := exec.Command("pgrep", "-x", "bash", "-P", strconv.FormatUint(*fP, 10))
		output, err := cmd.CombinedOutput()
		if err != nil {
			ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
			// exit status of 1 simply simply means we matched nothing
			if ws.ExitStatus() != 1 {
				log.Fatalf("Could not run pgrep: %v, %v\n", err, output)
			}
		}

		lines := strings.Split(string(output), "\n")

		if len(lines) > 0 {
			cPid, err := strconv.ParseUint(lines[0], 10, 64)
			if err != nil {
				log.Fatalf("Could not parseint: %v\n", err)
			}
			bestPid = cPid
		}

		if bestPid != 0 {
			n, err := os.Readlink(fmt.Sprintf("/proc/%v/cwd", bestPid))
			if err != nil {
				log.Fatalf("Could not read cwd of best pid %v: %v", bestPid, err)
			}

			// we don't care if this fails
			os.Setenv(MustChangeToDir, n)

			gv, err := goVersion(bestPid)
			if err == nil {
				os.Setenv(GoVersion, gv)
			}
			nv, err := nodeVersion(bestPid)
			if err == nil {
				os.Setenv(NodeVersion, nv)
			}
		}
	}

	cmd := flag.Args()
	syscall.Exec(cmd[0], cmd, os.Environ())
}

func goVersion(pid uint64) (string, error) {
	mi, err := os.Open(fmt.Sprintf("/proc/%d/mountinfo", pid))
	if err != nil {
		return "", err
	}
	defer mi.Close()

	root := ""

	sc := bufio.NewScanner(mi)

	for sc.Scan() {
		line := sc.Text()
		parts := strings.Fields(line)

		if parts[4] == "/home/myitcv/gos" {
			root = parts[3]
			break
		}
	}

	if strings.HasPrefix(root, "/home/myitcv/.gos/") {
		return strings.TrimPrefix(root, "/home/myitcv/.gos/"), nil
	}

	if root == "/home/myitcv/dev/go" {
		return "tip", nil
	}

	return "", errors.New("Not mounted or unknown error")
}

func nodeVersion(pid uint64) (string, error) {
	mi, err := os.Open(fmt.Sprintf("/proc/%d/mountinfo", pid))
	if err != nil {
		return "", err
	}
	defer mi.Close()

	root := ""

	sc := bufio.NewScanner(mi)

	for sc.Scan() {
		line := sc.Text()
		parts := strings.Fields(line)

		if parts[4] == "/home/myitcv/nodes" {
			root = parts[3]
			break
		}
	}

	if strings.HasPrefix(root, "/home/myitcv/.nodes/") {
		return strings.TrimPrefix(root, "/home/myitcv/.nodes/"), nil
	}

	return "", errors.New("Not mounted or unknown error")
}
