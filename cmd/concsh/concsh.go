// concsh allows you to concurrently run commands from your shell.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// all args after the first -- are then considered as a --- (notice the extra -) separated
// list of commands to be run concurrently
//
// output is interleaved on a line-by-line basis
//
// there is no shell evaluation of arguments
//
// TODO could support shell evalulation of lines (command line version already covered?)?
// TODO improve panics; some situations we might be able to better detect/handle?
// TODO add some mode whereby commands are executed only if all commands are valid (means
// that stdin read commands not executed until stdin is closed)
//
// exit code is 0 if all commands succeed without error; one of the non-zero exit codes otherwise

//go:generate gobin -m -run myitcv.io/cmd/pkgconcat -out gen_cliflag.go myitcv.io/_tmpls/cliflag

type result struct {
	exitCode int
	output   string
}

var (
	fConcurrency = flag.Uint("conc", 0, "define how many commands can be running at any given time; 0 = no limit; default = 0")
	fDebug       = flag.Bool("debug", false, "debug output")

	limiter chan struct{}
)

func main() {
	setupAndParseFlags(`concsh allows you to concurrently run commands from your shell

Usage:
	concsh -- comand1 arg1_1 arg1_2 ... --- command2 arg2_1 arg 2_2 ... --- ...
	concsh

In the case no arguments are provided, concsh will read the commands to execute from stdin, one per line

`)

	defer func() {
		r := recover()

		if err, ok := r.(error); ok {
			fmt.Fprintf(os.Stderr, "concsh error whilst running: %v", err)
			os.Exit(1)
		}

		panic(r)
	}()

	if *fConcurrency > 0 {
		limiter = make(chan struct{}, *fConcurrency)

		for i := uint(0); i < *fConcurrency; i++ {
			go func() {
				limiter <- struct{}{}
			}()
		}
	}

	var argSets [][]string

	exit := make(chan struct{})
	done := make(chan struct{})
	counter := make(chan struct{})
	results := make(chan result)

	go func() {
		exitCode := 0
		nr := 0
		finished := false

	Done:
		for {
			select {
			case <-counter:
				nr++
			case res := <-results:
				// we can't do anything other than put the combined output into
				// standard out.... because if we split the output we race on order
				// which is even worse
				fmt.Fprint(os.Stdout, res.output)

				if res.exitCode != 0 {
					exitCode = res.exitCode
				}

				nr--
				if finished && nr == 0 {
					break Done
				}
			case <-done:
				finished = true
			}
		}

		os.Exit(exitCode)
	}()

	if len(flag.Args()) == 0 {
		// read from stdin
		sc := bufio.NewScanner(os.Stdin)
		line := 1

		for sc.Scan() {
			args, err := split(sc.Text())
			if err != nil {
				infof("could not parse command on line %v: %v", line, err)
			}

			runCmd(args, counter, results)
			line++
		}
		if err := sc.Err(); err != nil {
			fatalf("unable to read from stdin: %v", err)
		}
	} else {
		var args []string

		for _, v := range flag.Args() {
			if v == "---" {
				argSets = append(argSets, args)
				args = nil
			} else {
				args = append(args, v)
			}
		}

		// in case we did not have a final ---
		argSets = append(argSets, args)

		for _, ag := range argSets {
			runCmd(ag, counter, results)
		}
	}

	done <- struct{}{}
	<-exit
}

func runCmd(args []string, counter chan struct{}, results chan result) int {
	res := 0

	if len(args) > 0 {
		if limiter != nil {
			<-limiter
		}
		counter <- struct{}{}
		go runCmdImpl(args, results)
	}

	return res
}

// based on the nice clean, algorithm in go generate
// https://github.com/golang/go/blob/c1730ae424449f38ea4523207a56c23b2536a5de/src/cmd/go/generate.go#L292

func split(line string) ([]string, error) {
	var words []string

Words:
	for {
		line = strings.TrimLeft(line, " \t")
		if len(line) == 0 {
			break
		}
		if line[0] == '"' {
			for i := 1; i < len(line); i++ {
				c := line[i] // Only looking for ASCII so this is OK.
				switch c {
				case '\\':
					if i+1 == len(line) {
						return nil, fmt.Errorf("bad backslash")
					}
					i++ // Absorb next byte (If it's a multibyte we'll get an error in Unquote).

				case '"':
					word, err := strconv.Unquote(line[0 : i+1])
					if err != nil {
						return nil, fmt.Errorf("bad quoted string")
					}
					words = append(words, word)
					line = line[i+1:]

					// Check the next character is space or end of line.
					if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
						return nil, fmt.Errorf("expect space after quoted argument")
					}
					continue Words
				}
			}
			return nil, fmt.Errorf("mismatched quoted string")
		}
		i := strings.IndexAny(line, " \t")
		if i < 0 {
			i = len(line)
		}
		words = append(words, line[0:i])
		line = line[i:]
	}

	return words, nil
}

func runCmdImpl(args []string, results chan result) {

	cmd := exec.Command(args[0], args[1:]...)

	var res result

	out, err := cmd.CombinedOutput()
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			panic(fmt.Errorf("Failed to run %q: %v\n%v", strings.Join(args, " "), err, string(out)))
		}

		exitCode := 0

		switch ws := ee.Sys().(type) {
		case syscall.WaitStatus:
			exitCode = ws.ExitStatus()
		default:
			panic(fmt.Errorf("Unknown exit error for %q: need to add case for %T", strings.Join(args, " "), ws))
		}

		res.exitCode = exitCode
	}

	res.output = string(out)

	debugf("Got output: %q\n", res.output)

	results <- res

	if limiter != nil {
		limiter <- struct{}{}
	}
}

func debugf(format string, args ...interface{}) {
	if *fDebug {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
