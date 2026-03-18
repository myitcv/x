// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

var fP = flag.Uint64("p", 0, "the PID of the process to walk for bash sub-processes")

const (
	NodeVersion     = "NODEVERSION"
	GoVersion       = "GOVERSION"
	CueVersion      = "CUEVERSION"
	RubyVersion     = "RUBYVERSION"
	HugoVersion     = "HUGOVERSION"
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
		// Try to use the given PID directly first (e.g. when passed a
		// foreground process PID from kitty remote control). Fall back to
		// walking the process tree to find a bash descendant (e.g. when
		// passed a window manager PID).
		bestPid := *fP

		// Check if we can read the CWD of the given PID directly
		if _, err := os.Readlink(fmt.Sprintf("/proc/%v/cwd", bestPid)); err != nil {
			bestPid = 0
		}

		// If the direct PID didn't work, walk the process tree to find
		// a bash descendant.
		if bestPid == 0 {
			frontier := []string{strconv.FormatUint(*fP, 10)}
			for len(frontier) > 0 && bestPid == 0 {
				parent := frontier[0]
				frontier = frontier[1:]

				cmd := exec.Command("pgrep", "-x", "bash", "-P", parent)
				output, err := cmd.CombinedOutput()
				if err == nil {
					lines := strings.Split(strings.TrimSpace(string(output)), "\n")
					if len(lines) > 0 && lines[0] != "" {
						cPid, err := strconv.ParseUint(lines[0], 10, 64)
						if err == nil {
							bestPid = cPid
							break
						}
					}
				}

				cmd = exec.Command("pgrep", "-P", parent)
				output, err = cmd.CombinedOutput()
				if err == nil {
					for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
						if line != "" {
							frontier = append(frontier, line)
						}
					}
				}
			}
		}

		if bestPid != 0 {
			n, err := os.Readlink(fmt.Sprintf("/proc/%v/cwd", bestPid))
			if err != nil {
				log.Fatalf("Could not read cwd of best pid %v: %v", bestPid, err)
			}

			os.Setenv(MustChangeToDir, n)

			if v, err := findVersion(bestPid, "/home/myitcv/gos"); err == nil {
				os.Setenv(GoVersion, v)
			}
			if v, err := findVersion(bestPid, "/home/myitcv/nodes"); err == nil {
				os.Setenv(NodeVersion, v)
			}
			if v, err := findVersion(bestPid, "/home/myitcv/cues"); err == nil {
				os.Setenv(CueVersion, v)
			}
			if v, err := findVersion(bestPid, "/home/myitcv/hugos"); err == nil {
				os.Setenv(HugoVersion, v)
			}
			if v, err := findVersion(bestPid, "/home/myitcv/rubys"); err == nil {
				os.Setenv(RubyVersion, v)
			}
		}
	}

	cmd := flag.Args()
	bin, err := exec.LookPath(cmd[0])
	if err != nil {
		log.Fatalf("Could not find %v in PATH: %v", cmd[0], err)
	}
	syscall.Exec(bin, cmd, os.Environ())
}

func findVersion(pid uint64, target string) (string, error) {
	out, err := exec.Command("findmnt", "-l", target, "-N", fmt.Sprintf("%d", pid)).CombinedOutput()
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		return "", fmt.Errorf("failed to find %s mount", target)
	}
	line := lines[1]

	parts := strings.Fields(line)

	if len(parts) != 4 {
		return "", fmt.Errorf("unexpected format of findmnt output: %q", out)
	}

	if parts[0] != target {
		return "", fmt.Errorf("failed to find %s mount in output: %q", target, out)
	}

	// parts[1] should now be like /dev/sda1[/myitcv/.gos/1.20.4]
	targetRegexp := regexp.MustCompile(`^(.+)\[(.+)\]$`)
	m := targetRegexp.FindStringSubmatch(parts[1])
	if m == nil {
		return "", fmt.Errorf("failed to match target in %q", parts[1])
	}
	// If we have a /myitcv/.$LANG prefix, then use the base,
	// otherwise assume tip
	lang := filepath.Base(target)
	fmt.Println("======", m[2], filepath.Join("/myitcv", "."+lang)+string(os.PathSeparator))
	if strings.HasPrefix(m[2], filepath.Join("/myitcv", "."+lang)+string(os.PathSeparator)) {
		return filepath.Base(m[2]), nil
	}
	return "tip", nil
}
