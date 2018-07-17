// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package debug contains facilities for programs to debug themselves while
// they are running.
package debug

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
)

// PrintStack prints to standard error the stack trace returned by Stack.
func PrintStack() {
	os.Stderr.Write(stack())
}

// Stack returns a formatted stack trace of the goroutine that calls it.
// For each routine, it includes the source line information and PC value,
// then attempts to discover, for Go functions, the calling function or
// method and the text of the line containing the invocation.
func Stack() []byte {
	return stack()
}

// stack implements Stack, skipping 2 frames
func stack() []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := 2; ; i++ { // Caller we care about is the user, 2 frames up
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		line-- // in stack trace, lines are 1-indexed but our array is 0-indexed
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.Trim(lines[n], " \t")
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

// Callers returns the stack trace of the goroutine that called it,
// starting n entries above the caller of Callers, as a space-separated list
// of filename:line-number pairs with no new lines.
func Callers(n, max int) []byte {
	var b bytes.Buffer
	prev := false
	for i := 0; i < max; i++ {
		_, file, line, ok := runtime.Caller(n + 1)
		if !ok {
			return b.Bytes()
		}
		if prev {
			fmt.Fprintf(&b, " ")
		}
		fmt.Fprintf(&b, "%s:%d", file, line)
		n++
		prev = true
	}
	return b.Bytes()
}
