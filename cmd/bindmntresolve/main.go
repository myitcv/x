// +build linux

// bindmntresolve prints the real directory path on disk of a possibly bind-mounted path
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	os.Exit(main1())
}

func main1() int {
	if err := mainerr(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func mainerr() error {
	flag.Usage = func() {
		mainUsage(os.Stderr)
	}
	flag.Parse()

	var p string

	switch len(flag.Args()) {
	case 0:
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %v", err)
		}
		p = cwd
	case 1:
		p = flag.Arg(0)
	default:
		return fmt.Errorf("bindmntresolve takes at most one argument")
	}

	if es, err := filepath.EvalSymlinks(p); err != nil {
		return fmt.Errorf("failed to resolve symlinks in %v: %v", p, err)
	} else {
		p = es
	}

	targetRegexp := regexp.MustCompile(`^.*\[(.*)\]$`)

	var res string

	// findmnt -n --raw -T /home/myitcv/.mountpoints/github_myitcv_neovim/src/github.com/myitcv/neovim
	//
	// results in a space-separated (quoted?) output:
	//
	// TARGET SOURCE FSTYPE OPTIONS
	//
	// e.g.
	//
	// /home/myitcv/gostuff /dev/sda1[/home/myitcv/.gostuff/1.11.1] ext4 rw,relatime,errors=remount-ro,data=ordered
	for {
		cmd := exec.Command("findmnt", "-n", "--raw", "-T", p)
		outb, err := cmd.Output()
		if err != nil {
			var stderr []byte
			if ee, ok := err.(*exec.ExitError); ok {
				stderr = append([]byte("\n"), ee.Stderr...)
			}
			return fmt.Errorf("failed to run %v: %v%s", strings.Join(cmd.Args, " "), err, stderr)
		}

		out := string(outb)
		if out[len(out)-1] == '\n' {
			out = out[:len(out)-1]
		}

		// there should be a single line
		lines := strings.Split(string(out), "\n")
		if len(lines) != 1 {
			return fmt.Errorf("command %v gave multiple lines: %q", strings.Join(cmd.Args, " "), out)
		}

		line := lines[0]

		fs := strings.Fields(line)
		if len(fs) != 4 {
			return fmt.Errorf("line %q did not have 4 fields", line)
		}

		target, disksource := fs[0], fs[1]

		// if target == / we are done (because there will not be a source)
		if target == "/" {
			res = filepath.Join(p, res)
			break
		}

		sms := targetRegexp.FindStringSubmatch(disksource)
		if sms == nil || len(sms) != 2 {
			return fmt.Errorf("source %q did not match as expected: %v", disksource, sms)
		}
		source := sms[1]

		// calculate p relative to target
		rel, err := filepath.Rel(target, p)
		if err != nil {
			return fmt.Errorf("failed to calculate %v relative to %v", p, target)
		}
		if rel != "." {
			res = filepath.Join(rel, res)
		}

		p = source
	}

	// add back the missing /
	fmt.Println(res)

	return nil
}
