package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type usageErr struct {
	err     string
	flagSet *flag.FlagSet
}

func (u usageErr) Error() string { return u.err }

type flagErr string

func (f flagErr) Error() string { return string(f) }

type usage struct {
	*flag.FlagSet
}

func (u usage) usage() {
	fmt.Fprintf(os.Stderr, `
Usage:

   egrunner [flags] DOCKERFILE SCRIPT

`[1:])
	u.PrintDefaults()
}

var _ flag.Value = (*dockerFlags)(nil)

var (
	fDockerRunFlags   dockerFlags
	fDockerBuildFlags dockerFlags
)

type dockerFlags []string

func (d *dockerFlags) String() string {
	return strings.Join(*d, " ")
}

func (d *dockerFlags) Set(v string) error {
	*d = append(*d, v)
	return nil
}
