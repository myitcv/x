// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

// +build linux

package main // import "myitcv.io/cmd/unsharemounts"

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"syscall"
)

const CLONE_NEWNS = 0x00020000

// This is designed to mimic unshare(1), the command-line interface for unshare(2)
// but in a much more constrained fashion. Only mounts will be unshared and then
// the calling user's shell will be exec'ed

// This will need to be:
//
// chown root:root
// chmod u+s
//
// to run.

func main() {
	runtime.LockOSThread()

	u, _ := user.Current()
	curr_uid := syscall.Getuid()
	curr_gid := syscall.Getgid()
	shell, err := getShell(u.Username)
	env := syscall.Environ()

	if err != nil {
		fmt.Printf("Failed to get shell for user %v: %v\n", u.Username, err)
		os.Exit(1)
	}

	if err := syscall.Unshare(CLONE_NEWNS); err != nil {
		fmt.Printf("Could not unshare: %v\n", err)
		os.Exit(1)
	}

	// drop root euid/egid - to have got this far we were called setuid
	// or call as root
	if err := syscall.Setreuid(curr_uid, curr_uid); err != nil {
		fmt.Printf("Failed to setuid: %v\n", err)
		os.Exit(1)
	}
	if err := syscall.Setregid(curr_gid, curr_gid); err != nil {
		fmt.Printf("Failed to setgid: %v\n", err)
		os.Exit(1)
	}

	// now exec the user's shell
	if err := syscall.Exec(shell, nil, env); err != nil {
		fmt.Printf("Failed to exec shell %v: %v\n", shell, err)
		os.Exit(1)
	}
}

func getShell(user string) (string, error) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return "", err
	}
	lines := bufio.NewReader(file)
	for {
		line, _, err := lines.ReadLine()
		if err != nil {
			break
		}
		split := strings.Split(string(line), ":")
		if len(split) != 7 {
			return "", errors.New("Unable to parse /etc/passwd")
		}
		if split[0] == user {
			// get the shell
			return split[6], nil
		}
	}

	// we did not find the user
	return "", errors.New("Could not find user")
}
