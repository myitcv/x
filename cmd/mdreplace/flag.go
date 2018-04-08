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
	res.WriteString(`Usage:

  mdreplace file1 file2 ...
  mdreplace

When called with no file arguments, mdreplace works with stdin

Flags:
`)

	// this feels a bit gross...
	flag.CommandLine.SetOutput(res)
	flag.PrintDefaults()

	usage = res.String()

	flag.CommandLine.SetOutput(os.Stderr)
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
}
