package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	eof = -1

	commStart = "<!--"
	commEnd   = "-->"

	codeFence = "```"

	tagTmpl = "__TEMPLATE"
	tagJson = "__JSON"

	end = "END"

	blockEnd = commStart + " " + end + " " + commEnd

	tmplBlock = commStart + " " + tagTmpl + ":"
	jsonBlock = commStart + " " + tagJson + ":"
)

type lexer struct {
	items chan item

	input string
	start int
	pos   int
	width *int
}

func lex(in io.Reader) (*lexer, chan item) {
	h := sha1.New()

	t := io.TeeReader(in, h)

	c, err := ioutil.ReadAll(t)
	if err != nil {
		fatalf("could not read from input: %v", err)
	}

	res := &lexer{
		items: make(chan item),

		input: string(c),
	}

	go res.run()

	return res, res.items
}

func (l *lexer) run() {
	for state := l.lexText; state != nil; {
		state = state()
	}
	close(l.items)
}

func (l *lexer) emit(t itemType) {
	i := item{
		typ: t,
		val: l.input[l.start:l.pos],
	}

	debugf("emit: %v\n", i)

	l.items <- i

	l.start = l.pos
}

func (l *lexer) emitNonEmpty(t itemType) {
	if l.pos <= l.start {
		return
	}

	l.emit(t)
}

func (l *lexer) lexCode() stateFn {
	l.pos += len(codeFence)
	l.emit(itemCodeFence)

	startOfLine := false

loop:
	for {
		if startOfLine && strings.HasPrefix(l.input[l.pos:], codeFence) {
			l.emit(itemCode)
			l.pos += len(codeFence)
			l.emit(itemCodeFence)
			return l.lexText
		}

		switch l.next() {
		case eof:
			break loop
		case '\n':
			startOfLine = true
		default:
			startOfLine = false
		}
	}

	l.errorf("reached end of file before seeing end of code block")
	return nil
}

func (l *lexer) lexText() stateFn {
	// text is the regular part of a markdown file. We are looking for the start
	// of blocks
	// for _, b := range blocks {
	// 	if b.isHeader(line) {
	// 		r.b = b
	// 		return r.readBlockStart
	// 	}
	// }

	startOfLine := true

loop:
	for {
		if startOfLine {
			switch {
			case strings.HasPrefix(l.input[l.pos:], codeFence):
				l.emitNonEmpty(itemText)
				return l.lexCode

			case strings.HasPrefix(l.input[l.pos:], commEnd):
				// we lext text when we are parsing the block arg
				l.emitNonEmpty(itemText)
				return l.lexCommEnd

			case strings.HasPrefix(l.input[l.pos:], blockEnd):
				// we lex text for anything that exists between
				// the start of a block and the end (it will be
				// discarded but hey)
				l.emitNonEmpty(itemText)
				return l.lexBlockEnd

			case strings.HasPrefix(l.input[l.pos:], tmplBlock):
				l.emitNonEmpty(itemText)
				return l.lexTmplBlock

			case strings.HasPrefix(l.input[l.pos:], jsonBlock):
				l.emitNonEmpty(itemText)
				return l.lexJsonBlock
			}
		}

		switch l.next() {
		case eof:
			break loop
		case '\n':
			startOfLine = true
		default:
			startOfLine = false
		}
	}

	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemText)
	}

	l.emit(itemEOF)
	return nil
}

func (l *lexer) lexTmplBlock() stateFn {
	l.pos += len(tmplBlock)
	l.emit(itemTmplBlockStart)

	return l.lexCmdAndArgs
}

func (l *lexer) lexJsonBlock() stateFn {
	l.pos += len(jsonBlock)
	l.emit(itemJsonBlockStart)

	return l.lexCmdAndArgs
}

func (l *lexer) lexBlockEnd() stateFn {
	l.pos += len(blockEnd)
	l.emit(itemBlockEnd)

	// accept spaces
	l.acceptRun(" \t")

	if l.next() != '\n' {
		return l.errorf("expected to see newline after block end")
	}
	l.ignore()

	return l.lexText
}

func (l *lexer) lexCommEnd() stateFn {
	l.pos += len(commEnd)
	l.emit(itemCommEnd)

	// accept spaces
	l.acceptRun(" \t")

	if l.next() != '\n' {
		return l.errorf("expected to see newline after block header")
	}
	l.ignore()

	return l.lexText
}

func intVal(i int) *int {
	return &i
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = intVal(0)
		return eof
	}

	ru, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = intVal(w)
	l.pos += w

	return ru
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	i := item{
		typ: itemError,
		val: fmt.Sprintf(format, args...),
	}

	if panicErrors {
		panic(fmt.Errorf(i.val))
	}

	l.items <- i

	return nil
}

func (l *lexer) backup() {
	if l.width == nil {
		panic(fmt.Errorf("tried to backup twice"))
	}

	l.pos -= *l.width
	l.width = nil
}

// lexCmdAndArgs borrows a lot in style from the go generate command
// parsing of arguments in go:generate directives
func (l *lexer) lexCmdAndArgs() stateFn {
Words:
	for {
		l.acceptRun(" \t")
		l.ignore()

		v := l.next()
		switch v {
		case eof:
			return l.errorf("missing end of line in arg list")
		case '"':
			for {
				switch l.next() {
				case eof:
					return l.errorf("saw end of file before end of quoted arg")
				case '\n':
					return l.errorf("saw end of line before end of quoted arg")
				case '\\':
					l.next()
				case '"':
					_, err := strconv.Unquote(l.input[l.start:l.pos])
					if err != nil {
						return l.errorf("bad quoted string")
					}
					l.emit(itemQuoteArg)

					// if we have more to do ensure the next character is space or end of line.
					switch p := l.peek(); {
					case p != '\n' && p != ' ' && p != '\t':
						return l.errorf("expect space after quoted argument")
					case p == '\n':
						break Words
					}

					continue Words
				}
			}
		default:
			// we have an unquoted arg
			for {
				switch l.input[l.pos] {
				case ' ', '\t':
					l.emit(itemArg)
					continue Words
				case '\n':
					l.emit(itemArg)
					break Words
				}

				if l.next() == eof {
					return l.errorf("missing end of line in arg list")
				}
			}
		}
	}

	// discard the newline
	if n := l.next(); n != '\n' {
		return l.errorf("expected to see new line at this point; saw %v", string(n))
	}

	l.ignore()

	return l.lexText
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (r *lexer) ignore() {
	r.start = r.pos
}
