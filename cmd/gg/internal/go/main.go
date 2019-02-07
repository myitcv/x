package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	runtime.LockOSThread()

	fmt.Fprintf(os.Stderr, "go %v\n", strings.Join(os.Args[1:], " "))
	sep := string(filepath.ListSeparator)
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to derive path to self: %v", err)
		os.Exit(1)
	}
	selfDir := filepath.Dir(self)
	var path []string
	for _, p := range strings.Split(os.Getenv("PATH"), sep) {
		if p != selfDir {
			path = append(path, p)
		}
	}

	if err := os.Setenv("PATH", strings.Join(path, sep)); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set PATH: %v", err)
		os.Exit(1)
	}

	cmd := exec.Command("go", os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
