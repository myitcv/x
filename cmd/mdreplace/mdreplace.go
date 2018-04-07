// mdreplace is a tool to help you keep your markdown README/documentation current.
package main // import "myitcv.io/cmd/mdreplace"

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

// take input from stdin or files (args)
// -w flag will write back to files (error if stdin)
// if no -w flag, write to stdout
//
// blocks take the form
//
// <!-- __XYZ: command args ...
// template block
// -->
// anything
// <!-- END -->
//
// No nesting supported; fatal if nested; fatal if block not terminated
// The "anything" part may contain code blocks; these code blocks effectively
// escape the end block; so the code block itself must be terminated, otherwise
// we will never find the end.

// ===========================
// Blocks:
//
// __TEMPLATE: cmd arg1 arg2 ...
// Takes the output from the command (string) and passes it to the
// template defined in the template block.
//
// Functions available for such a block include:
//
// * Lines(s string) []string
//
// __JSON: assumes the output from the command will be JSON; that is decoded into
// an interface{} and passed to the template defined in the template block.
// ===========================

var (
	fHelp  = flag.Bool("h", false, "show usage information")
	fWrite = flag.Bool("w", false, "whether to write back to input files (cannot be used when reading from stdin)")

	usage string // gets populated at runtime
)

type stateFn func() stateFn

func main() {
	setupAndParseFlags()

	if *fHelp {
		infof(usage)
		os.Exit(0)
	}

	args := flag.Args()

	if *fWrite && len(args) == 0 {
		fatalf("Cannot use -w flag when reading from stdin\n\n%v", usage)
	}

	if len(args) == 0 {
		if err := run(os.Stdin, os.Stdout); err != nil {
			fatalf("%v\n", err)
		}
	} else {
		// ensure all the files exist first

		var files []*os.File

		for _, f := range args {
			i, err := os.Open(f)
			if err != nil {
				fatalf("failed to open %v: %v\n", f, err)
			}

			files = append(files, i)
		}

		for _, f := range files {
			// we can do this concurrently
			var out io.Writer

			if *fWrite {
				out = new(bytes.Buffer)
			} else {
				out = os.Stdout
			}

			if err := run(f, out); err != nil {
				fatalf("%v\n", err)
			}

			// write back if -w specific
			if *fWrite {
				fn := f.Name()

				err := ioutil.WriteFile(f.Name(), out.(*bytes.Buffer).Bytes(), 0644)
				if err != nil {
					fatalf("failed to write to %v: %v\n", fn, err)
				}

			}

		}
	}

}

func run(r io.Reader, w io.Writer) error {
	_, items := lex(r)

	if err := process(items, w); err != nil {
		return err
	}

	return nil
}

func infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func debugf(format string, args ...interface{}) {
	if debug {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
