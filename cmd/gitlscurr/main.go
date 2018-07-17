// gitlscurr is an extended version of git ls-files that aims to reflect the files
// currently on disk as opposed to the current index (deleted files are in the index
// but not on disk)
//
package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
)

func main() {
	// In set terms we need
	//
	// {git ls-files -o --exclude-standard --cached} - {git ls-files --deleted}

	delLines := make(map[string]struct{})

	run(func(line string) {
		delLines[line] = struct{}{}
	}, "git", "ls-files", "--deleted")

	run(func(line string) {
		if _, ok := delLines[line]; !ok {
			fmt.Print(line)
		}
	}, "git", "ls-files", "-o", "--exclude-standard", "--cached")
}

func run(f func(line string), command string, args ...string) {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	readDone := make(chan struct{})

	go read(stdout, readDone, f)

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	<-readDone

	// now the read is done we can wait on the process
	// without causing a race
	err = cmd.Wait()
	if err != nil {
		panic(err)
	}
}

func read(in io.ReadCloser, done chan struct{}, f func(line string)) {
	b := bufio.NewReader(in)

	for {
		line, err := b.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				if line != "" {
					f(line + "\n")
				}
			}

			// notice we are ignoring io errors... because any other
			// errors will be handled by the Wait on the cmd with a non-zero
			// exit code

			break
		}

		f(line)
	}

	done <- struct{}{}
}
