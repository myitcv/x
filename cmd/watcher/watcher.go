// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

// watcher is a Linux-based directory watcher for triggering commands
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	fsnotify "gopkg.in/fsnotify/fsnotify.v1"

	"github.com/kr/fs"
)

// TODO
// Warning: this code is pretty messy; some of my first Go code
//
// * implement timeout for killing long-running process

var (
	fIgnorePaths ignorePaths

	fDebug           = flag.Bool("debug", false, "give debug output")
	fQuiet           = flag.Duration("q", 100*time.Millisecond, "the duration of the 'quiet' window; format is 1s, 10us etc. Min 1 millisecond")
	fPath            = flag.String("p", "", "the path to watch; default is CWD [*]")
	fFollow          = flag.Bool("f", false, "whether to follow symlinks or not (recursively) [*]")
	fDie             = flag.Bool("d", false, "die on first notification; only consider -p and -f flags")
	fDontClearScreen = flag.Bool("c", false, "do not clear the screen before running the command")
	fNotInitial      = flag.Bool("i", false, "don't run command at time zero; only applies when -d not supplied")
	fTimeout         = flag.Duration("t", 0, "the timeout after which a process is killed; not valid with -k")
	fDontKill        = flag.Bool("k", false, "don't kill the running command on a new notification")

	hashCache = make(map[string]string)
)

const (
	GitDir = ".git"
)

var GloballyIgnoredDirs = []string{GitDir}

func init() {
	flag.Var(&fIgnorePaths, "I", "Paths to ignore. Absolute paths are absolute to the path; relative paths can match anywhere in the tree")
}

type ignorePaths []string

func (i *ignorePaths) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ignorePaths) String() string {
	return fmt.Sprint(*i)
}

func showUsage() {
	fmt.Fprintf(os.Stderr, "Command mode:\n\t%v [-q duration] [-p /path/to/watch] [-i] [-f] [-c] [-k] CMD ARG1 ARG2...\n\nDie mode:\n\t%v -d [-p /path/to/watch] [-f]\n\n", os.Args[0], os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nOnly options marked with [*] are valid in die mode\n")
}

//go:generate pkgconcat -out gen_cliflag.go myitcv.io/_tmpls/cliflag

func main() {
	setupAndParseFlags("")

	if *fDebug {
		*fDontClearScreen = true
	}

	path := *fPath
	if path == "" {
		path = "."
	}
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(path)
	if err != nil {
		fatalf("Could not stat -p supplied path [%v]: %v\n", path, err)
	}

	if *fDie {
		if *fQuiet < 0 {
			fatalf("Quiet window duration [%v] must be positive\n", *fQuiet)
		}
		if *fTimeout < 0 {
			fatalf("Command timeout duration [%v] must be positive\n", *fTimeout)
		}
	}

	if *fDie && *fQuiet < time.Millisecond {
		log.Fatalln("Quiet time period must be at least 1 millisecond")
	}

	w, err := newWatcher()
	if err != nil {
		fatalf("Could not create a watcher: %v\n", err)
	}
	defer w.close()

	w.kill = !*fDontKill
	w.timeout = *fTimeout
	w.quiet = *fQuiet
	w.initial = !*fNotInitial
	w.command = flag.Args()
	w.clearScreen = !*fDontClearScreen
	w.ignorePaths = append(fIgnorePaths, GloballyIgnoredDirs...)
	w.absPath = path

	if *fDie {
		w.watchOnce(path)
	} else {
		w.watchLoop(path)
	}
}

type watcher struct {
	iwatcher    *fsnotify.Watcher
	kill        bool
	clearScreen bool
	command     []string
	ignorePaths []string
	absPath     string
	initial     bool
	timeout     time.Duration
	quiet       time.Duration
}

func newWatcher() (*watcher, error) {
	res := &watcher{}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("could not create a watcher: %v", err)
	}
	res.iwatcher = w
	return res, nil
}

func (w *watcher) close() error {
	err := w.iwatcher.Close()
	if err != nil {
		return fmt.Errorf("could not close watcher: %v", err)
	}
	return nil
}

func (w *watcher) recursiveWatchAdd(p string) {
	// p is a path; may or may not be a directory

	fi, err := os.Stat(p)
	if err != nil {
		debugf("** recursiveWatchAdd: os.Stat(%v): %v\n", p, err)
		return
	}
	if !fi.IsDir() {
		hashCache[p] = hash(p)
		if err := w.iwatcher.Add(p); err != nil {
			debugf("** recursiveWatchAdd: watcher add to %v: %v\n", p, err)
		}
		return
	}

	walker := fs.Walk(p)
WalkLoop:
	for walker.Step() {
		if err := walker.Err(); err != nil {
			debugf("** recursiveWatchAdd: walker.Err: %v\n", err)
			continue
		}
		s := walker.Stat()

		hashCache[walker.Path()] = hash(walker.Path())

		if s.IsDir() {

			for _, s := range w.ignorePaths {
				rel, _ := filepath.Rel(w.absPath, walker.Path())

				if filepath.IsAbs(s) {
					nonAbs := strings.TrimPrefix(s, "/")

					if nonAbs == rel {
						walker.SkipDir()
						continue WalkLoop
					}

				} else {
					if strings.HasSuffix(rel, s) {
						walker.SkipDir()
						continue WalkLoop
					}
				}
			}
			if err := w.iwatcher.Add(walker.Path()); err != nil {
				debugf("** recursiveWatchAdd: walker add watch to dir member %v: %v\n", walker.Path(), err)
			}
		} else {
			if err := w.iwatcher.Add(walker.Path()); err != nil {
				debugf("** recursiveWatchAdd: walker add watch to %v: %v\n", walker.Path(), err)
			}
		}
	}
}

func (w *watcher) recursiveWatchRemove(p string) error {
	// TODO make this recursive if needs be?
	err := w.iwatcher.Remove(p)
	if err != nil {
		// TODO anything better to do that just swallow it?
	}
	return nil
}

func (w *watcher) watchOnce(p string) {
	w.recursiveWatchAdd(p)

	retVal := 0

	select {
	case _ = <-w.iwatcher.Events:
		// TODO handle the queue overflow? probably not needed
	case _ = <-w.iwatcher.Errors:
		// TODO handle the queue overflow
		retVal = 1
	}
	os.Exit(retVal)
}

// in case of any errors simply return "" because we're probably
// racing with another process
func hash(fn string) (res string) {
	h := sha256.New()

	fi, err := os.Stat(fn)
	if err != nil {
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		return
	}

	defer f.Close()

	if fi.IsDir() {
		ns, err := f.Readdirnames(0)
		if err != nil {
			return
		}

		for _, e := range ns {
			h.Write([]byte(e))
		}
	} else {
		if _, err := io.Copy(h, f); err != nil {
			return
		}
	}

	return string(h.Sum(nil))
}

func (w *watcher) watchLoop(p string) {
	w.recursiveWatchAdd(p)

	workBus := make(chan struct{})
	eventBus := make(chan fsnotify.Event)

	go w.commandLoop(workBus)

	// buffer
	go func() {
		var buffer []fsnotify.Event
		var backlog []fsnotify.Event
		var timers []<-chan time.Time

		debugf("buffer loop> initial: %v\n", w.initial)

		if w.initial {
			// dummy event
			buffer = append(buffer, fsnotify.Event{})
			timers = append(timers, time.After(0))
		}

	Buffer:
		for {
			debugln("buffer loop> start")
			var timeout <-chan time.Time
			if len(timers) > 0 {
				debugln("buffer loop> can timeout")
				timeout = timers[0]
			}

			var doWork chan struct{}
			if len(backlog) > 0 {
				debugln("buffer loop> have backlog")
				doWork = workBus
			}

			select {
			case e := <-eventBus:
				for _, b := range buffer {
					if b == e {
						continue Buffer
					}
				}
				for _, b := range backlog {
					if b == e {
						continue Buffer
					}
				}
				buffer = append(buffer, e)
				timers = append(timers, time.After(w.quiet))
			case <-timeout:
				e := buffer[0]
				buffer = buffer[1:]

				timers = timers[1:]
				backlog = append(backlog, e)
			case doWork <- struct{}{}:
				debugln("buffer loop> sent work")
				backlog = backlog[1:]
			}
		}

	}()

	for {
		select {
		case e := <-w.iwatcher.Events:
			// TODO handle the queue overflow... this could happen
			// if we do get queue overflow, might need to look at putting
			// subscriptions on another goroutine, buffering the adds/
			// removes somehow
			switch e.Op {
			case fsnotify.Create:
				w.recursiveWatchAdd(e.Name)
			case fsnotify.Remove, fsnotify.Rename:
				w.recursiveWatchRemove(e.Name)
			}

			debugf("event loop> %v for %v\n", e.Op, e.Name)

			hs := hash(e.Name)
			ce := hashCache[e.Name]

			if ce != hs {
				debugln("event loop> cache miss")
				hashCache[e.Name] = hs
				eventBus <- e
			}

		case _ = <-w.iwatcher.Errors:
			// TODO handle the queue overflow
		}
	}
}

func (w *watcher) commandLoop(workBus chan struct{}) {
	outWorkBus := workBus
	args := []string{"-O", "globstar", "-c", "--", strings.Join(w.command, " ")}
	var command *exec.Cmd
	cmdDone := make(chan struct{})

	if w.clearScreen {
		fmt.Printf("\033[2J")
	}

	runCmd := func() {
		if command != nil {
			_ = command.Process.Kill()
		}
		if w.clearScreen {
			fmt.Printf("\033[2J")
		}
		command = exec.Command("bash", args...)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if *fDontKill {
			outWorkBus = nil
		}
		debugf("work loop> starting %q\n", strings.Join(args, " "))
		err := command.Start()
		if err != nil {
			fatalf("We could not run the command provided: %v\n", err)
		}
		go func(c *exec.Cmd) {
			_ = c.Wait()
			debugln("work loop> work done")
			cmdDone <- struct{}{}
		}(command)
	}

	for {
		select {
		case <-outWorkBus:
			debugln("work loop> got work")
			runCmd()
		case <-cmdDone:
			outWorkBus = workBus
			command = nil
		}
	}
}

func debugf(format string, args ...interface{}) {
	if *fDebug {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

func debugln(args ...interface{}) {
	args = append(args, "\n")
	if *fDebug {
		fmt.Fprint(os.Stderr, args...)
	}
}
