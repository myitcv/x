// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

var umount = flag.Bool("u", false, "whether to unmount")

func main() {
	runtime.LockOSThread()

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: mount_wrap OLD_DIR NEW_DIR")
		fmt.Fprintln(os.Stderr, "       mount_wrap -u MOUNT_DIR")
	}
	flag.Parse()

	// restrict the mounts to dirs in the current user's home dir
	u, _ := user.Current()

	if *umount {
		if len(flag.Args()) != 1 {
			flag.Usage()
			os.Exit(1)
		}

		mount_dir := filepath.Clean(flag.Arg(0))

		if !strings.HasPrefix(mount_dir, u.HomeDir) {
			fmt.Printf("Restricted to (un)mounting dirs within your home dir %v\n", u.HomeDir)
			os.Exit(1)
		}

		// now become root
		err := syscall.Setreuid(0, 0)
		if err != nil {
			fmt.Printf("Error performing setuid: %v\n", err)
			os.Exit(1)
		}
		out, err := exec.Command("/bin/umount", mount_dir).Output()
		if err != nil {
			fmt.Printf("Error performing umount: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(out))
	} else {
		if len(flag.Args()) != 2 {
			flag.Usage()
			os.Exit(1)
		}
		old_dir := filepath.Clean(flag.Arg(0))
		new_dir := filepath.Clean(flag.Arg(1))

		if !strings.HasPrefix(old_dir, u.HomeDir) || !strings.HasPrefix(new_dir, u.HomeDir) {
			fmt.Printf("Restricted to (un)mounting dirs within your home dir %v\n", u.HomeDir)
			os.Exit(1)
		}

		// now we become root
		err := syscall.Setreuid(0, 0)
		if err != nil {
			fmt.Printf("Error performing setuid: %v\n", err)
			os.Exit(1)
		}
		out, err := exec.Command("/bin/mount", "--make-private", "/").CombinedOutput()
		if err != nil {
			fmt.Printf("Error performing make-private on /: %v\n", err)
			os.Exit(1)
		}
		out, err = exec.Command("/bin/mount", "-n", "--bind", old_dir, new_dir).CombinedOutput()
		if err != nil {
			fmt.Printf("Error performing mount: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(out))
	}
}
