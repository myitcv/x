// bindmntresolve prints the real directory path on disk of a possibly bind-mounted path
package main

import (
	"flag"
	"fmt"
	"os"

	"myitcv.io/cmd/internal/bindmnt"
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

	p, err := bindmnt.Resolve(p)
	if err != nil {
		return err
	}

	fmt.Println(p)
	return nil
}
