// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	_log "log"
	"os"
	"path/filepath"
	"runtime"
)

var (
	fVerbose = flag.Bool("v", false, "log interesting messages")
	fFix     = flag.Bool("f", false, "fix the files passed as arguments")
	fIndent  = flag.String("indent", "\t", "the indent string")
	fPrefix  = flag.String("prefix", "", "the prefix string")
	fConc    = flag.Uint("conc", uint(runtime.NumCPU()), "the number of concurrent formatters; defaults to number of cores")
)

type Config struct {
	Prefix string
	Indent string
}

const (
	ConfigFileName = ".jsonlintconfig.json"
)

const (
	logPrefix = ""
	logFlags  = 0
)

var config Config
var log = _log.New(os.Stdout, logPrefix, logFlags)
var elog = _log.New(os.Stderr, logPrefix, logFlags)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	flag.Parse()

	files := flag.Args()

	if len(files) == 0 {
		files = []string{os.Stdin.Name()}
	}

	nf := int(*fConc)

	formatters := make(chan *formatter, nf)

	for i := 0; i < nf; i++ {
		formatters <- &formatter{}
	}

	failed := make(chan bool)

	go func() {
		for _, file := range files {
			f := <-formatters

			go func(fn string, f *formatter) {
				f.file = fn
				err := f.format()
				if err != nil {
					elog.Print(err)
					failed <- true
				}

				formatters <- f
			}(file, f)
		}

		// now we need to count the formatters back in
		for i := 0; i < nf; i++ {
			<-formatters
		}

		// then signal that we're done
		close(failed)
	}()

	errCount := 0

	for {
		if _, ok := <-failed; !ok {
			break
		}

		errCount++
	}

	if errCount > 0 {
		os.Exit(1)
	}
}

type formatter struct {
	file string
}

var procFile = fmt.Errorf("error handling file")

func (f *formatter) failf(format string, args ...interface{}) {
	var fn string

	if f.file == os.Stdin.Name() {
		fn = "<stdin>"
	} else {
		fn = f.file
	}

	panic(fmt.Errorf("%v: %v\n", fn, fmt.Sprintf(format, args...)))
}

func (f *formatter) format() (retErr error) {
	var file *os.File
	var err error

	defer func() {
		if err := recover(); err != nil {
			retErr = err.(error)
		}

		if file != nil {
			file.Close()
		}
	}()

	if f.file == os.Stdin.Name() {
		file = os.Stdin
	} else {
		file, err = os.Open(f.file)
		if err != nil {
			f.failf("unable to open: %v", err)
		}
	}

	var r io.Reader = file
	var orig hash.Hash

	if !*fFix {
		// we have to work out whether the file is formatted or not
		orig = sha1.New()
		r = io.TeeReader(file, orig)
	}

	// for a bit of fun let's use a pipe...
	pr, pw := io.Pipe()

	go func() {
		defer file.Close()

		r := bufio.NewReader(r)
		skip := false

		for {
			var err error

			done := false
			write := true
			line, rerr := r.ReadSlice('\n')

			if rerr == nil {
				if !skip && bytes.HasPrefix(line, []byte("//")) {
					write = false
				}

				skip = false
			} else if rerr == bufio.ErrBufferFull {
				if !skip && bytes.HasPrefix(line, []byte("//")) {
					skip = true
					write = false
				}
			} else if rerr == io.EOF {
				if skip {
					write = false
				}

				done = true
			} else {
				pw.CloseWithError(fmt.Errorf("unable to read input: %v", err))
				return
			}

			if write {
				start := 0

				for start < len(line) {
					start, err = pw.Write(line[start:])

					if err != nil {
						pw.CloseWithError(fmt.Errorf("unable to write to pipe: %v", err))
						return
					}
				}
			}

			if done {
				break
			}
		}

		pw.Close()
	}()

	dec := json.NewDecoder(pr)

	var j interface{}
	if err := dec.Decode(&j); err != nil {
		f.failf("does not contain valid JSON: %v", err)
	}

	// TODO we could optimise this to reuse configs etc?
	// need to handle errors etc
	c := deriveConfig(file)

	if *fVerbose {
		log.Printf("For file %v using config %#v\n", file.Name(), c)
	}

	b, err := json.MarshalIndent(j, c.Prefix, c.Indent)
	if err != nil {
		f.failf("could not be formatted: %v", err)
	}

	b = append(b, '\n')

	if *fFix {
		if file == os.Stdin {
			_, err = os.Stdout.Write(b)
		} else {
			err = ioutil.WriteFile(file.Name(), b, 0644)
		}
		if err != nil {
			f.failf("could not write formatted JSON back to file: %v", err)
		}
	} else {
		hash := sha1.New()
		_, err := hash.Write(b)
		if err != nil {
			f.failf("could not compute hash of formatted content: %v", err)
		}
		if !bytes.Equal(hash.Sum(nil), orig.Sum(nil)) {
			f.failf("is not well-formatted")
		}
	}

	return nil
}

// TODO this needs tidying up
func deriveConfig(file *os.File) (res Config) {
	var err error

	res.Indent = *fIndent
	res.Prefix = *fPrefix

	var dir string

	if file == os.Stdin {
		d, err := os.Getwd()
		if err != nil {
			elog.Fatalf("Could not get working directory: %v", err)
		}

		dir = d
	} else {
		abs, err := filepath.Abs(file.Name())
		if err != nil {
			elog.Fatalf("Could not get absolute path to %v: %v", file.Name(), err)
		}

		dir = filepath.Dir(abs)
	}

	var fi *os.File

	for {
		fp := filepath.Join(dir, ConfigFileName)

		if *fVerbose {
			log.Printf("Checking for config file %v\n", fp)
		}

		fi, err = os.Open(fp)

		if err == nil {
			break
		}

		p := filepath.Dir(dir)

		if p == dir {
			break
		}

		dir = p
	}

	if fi == nil {
		return
	}

	if *fVerbose {
		log.Printf("Found config file at %v\n", fi.Name())
	}

	dec := json.NewDecoder(fi)
	err = dec.Decode(&res)

	clErr := fi.Close()

	if clErr != nil {
		elog.Fatalf("Could not close file %v: %v", fi.Name(), clErr)
	}

	if err != nil {
		elog.Fatalf("Unable to decode config from %v: %v", fi.Name(), err)
	}

	return res
}
