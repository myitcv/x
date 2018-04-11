package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	usage string
)

func setupAndParseFlags(msg string) {
	flag.Usage = func() {
		res := new(strings.Builder)
		fmt.Fprint(res, msg)

		// this feels a bit gross...
		flag.CommandLine.SetOutput(res)
		flag.PrintDefaults()
		res.WriteString("\n")
		res.WriteString("\n")

		fmt.Fprint(os.Stderr, foldOnSpaces(res.String(), 80))

		os.Exit(0)
	}
	flag.Parse()

	flag.CommandLine.SetOutput(os.Stderr)
}

func foldOnSpaces(input string, width int) string {
	var carry string
	var indent string // the indent (if there is one) when we carry

	sc := bufio.NewScanner(strings.NewReader(input))

	res := new(strings.Builder)
	first := true

Line:
	for {
		carried := carry != ""
		if !carried {
			if !sc.Scan() {
				break
			}

			if first {
				first = false
			} else {
				res.WriteString("\n")
			}

			carry = sc.Text()

			// caclculate the indent
			iBuilder := new(strings.Builder)

			for _, r := range carry {
				if !unicode.IsSpace(r) {
					break
				}
				iBuilder.WriteRune(r)
			}

			indent = iBuilder.String()

			// now strip the space on the line
			carry = strings.TrimSpace(carry)
		}

		if len(carry) == 0 {
			continue
		}

		// we always strip the indent - so write it back
		res.WriteString(indent)

		// fast path where number of bytes is less than width
		// nothing to calculate in terms of width
		// TODO is this safe?
		if len(indent)+len(carry) < width {
			res.WriteString(carry)
			carry = ""
			continue
		}

		lastSpace := -1

		var ia norm.Iter
		ia.InitString(norm.NFD, carry)
		nc := len(indent)

		// TODO handle this better
		if nc >= width {
			fatalf("cannot foldOnSpaces where indent is greater than width")
		}

		var postSpace string

	Space:
		for !ia.Done() {
			prevPos := ia.Pos()
			nbs := ia.Next()
			r, rw := utf8.DecodeRune(nbs)
			if rw != len(nbs) {
				fatalf("didn't expect a multi-rune normalisation response: %v", string(nbs))
			}

			nc++

			// do we have a space? If so there should only be a single rune
			spaceCount := 0

			if isSplitter(r) {
				spaceCount++
			}

			switch spaceCount {
			case 0:
				// we can't split - keep going
				if lastSpace == -1 {
					res.WriteRune(r)
					continue Space
				}

				// so at this point we know we have previously seen
				// a space so nc cannot have previously have been == w
				if nc == width {
					// we are about to exceed the limit; write a new line
					// to our output then put postSpace + prevPos: into carry
					// remembering that postSpace will have a space on the
					// left so we need to trim it
					res.WriteString("\n")
					carry = strings.TrimLeftFunc(postSpace+carry[prevPos:], unicode.IsSpace)
					continue Line
				}

				// so the only thing left to do is add to postSpace
				postSpace += string(r)
				continue Space
			case 1:
				// we have hit a space; if we are already
				// over the limit we want to drop the space
				// and set carry to be the left-space-trimmed
				// remainder

				res.WriteString(postSpace)

				switch {
				case nc == width:
					res.WriteRune(r)
					fallthrough
				case nc > width:
					res.WriteString("\n")
					carry = strings.TrimLeftFunc(carry[ia.Pos():], unicode.IsSpace)
					// indent remains as it was
					continue Line
				}

				// we still have capacity
				res.WriteRune(r)

				// otherwise we are still ok... move our last space
				// pointer up, print anything we had from the previous
				// last space and continue
				lastSpace = nc
				postSpace = ""
				continue Space
			default:
				fatalf("is this even possible?")
			}
		}

		// we exhausted the line
		carry = ""
	}

	if err := sc.Err(); err != nil {
		fatalf("failed to scan in foldOnSpaces: %v", err)
	}

	return res.String()
}

func isSplitter(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}

	switch r {
	case '/':
		return true
	}

	return false
}

func fatalf(format string, args ...interface{}) {
	if format[len(format)-1] != '\n' {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
