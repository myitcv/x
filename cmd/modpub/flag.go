package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func setupAndParseFlags() {
	flag.Parse()

	res := new(strings.Builder)

	// could optionally add some useful details here

	// this feels a bit gross...
	flag.CommandLine.SetOutput(res)
	flag.PrintDefaults()

	usage = res.String()

	flag.CommandLine.SetOutput(os.Stderr)
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
}
