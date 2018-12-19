// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

// gg is a dependency-aware wrapper for go generate.
package main

import (
	"flag"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	debug = false
)

var (
	fDebug     = flag.Bool("debug", false, "debug logging")
	fLoopLimit = flag.Int("loopLimit", 10, "limit on the number of go generate iterations per package")
)

// TODO: when we move gg to be fully-cached based, we will need to load all
// deps via go list so that we can derive a hash of their go files etc. At this
// point we will need https://go-review.googlesource.com/c/go/+/112755 or
// similar to have landed.

func main() {
	setupAndParseFlags("")
	loadConfig()

	pre := time.Now()
	defer func() {
		verbosef("total time: %.2fms\n", time.Now().Sub(pre).Seconds()*1000)
	}()

	ps := loadPkgs(flag.Args())

	possRoots := make(pkgSet)

	for p := range ps {
		if p.isTool && p.ready() {
			possRoots[p] = true
		}
	}

	for pr := range possRoots {
		debugf("Poss root: %v\n", pr)
	}

	var work []*pkg
	for pr := range possRoots {
		work = append(work, pr)
	}

	for len(work) > 0 {
		outPkgs := make(map[*pkg]bool)
		var is, gs []*pkg
		var rem []*pkg

	WorkScan:
		for _, w := range work {
			if w.isTool {
				is = append(is, w)
				continue WorkScan
			} else {
				// we are searching for clashes _between_ packages not intra
				// package (because that clash is just fine - no race condition)
				if outPkgs[w] {
					// clash
					goto NoWork
				}
				for _, ods := range w.toolDeps {
					for od := range ods {
						if outPkgs[od] {
							// clash
							goto NoWork
						}
					}
				}
				gs = append(gs, w)
				// no clashes
				outPkgs[w] = true
				for _, ods := range w.toolDeps {
					for od := range ods {
						outPkgs[od] = true
					}
				}
				continue WorkScan
			}

		NoWork:
			rem = append(rem, w)
		}

		work = rem

		var iwg sync.WaitGroup

		// the is (installs) can proceed concurrently, as can the gs (generates),
		// because we know in the case of the latter that their output packages
		// are mutually exclusive
		if len(is) > 0 {
			var ips []string
			for _, i := range is {
				ips = append(ips, i.ImportPath)
			}
			pre := time.Now()
			cmd := exec.Command("go", "install")
			cmd.Args = append(cmd.Args, ips...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				fatalf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
			}
			verbosef("%v (%.2fms)\n", strings.Join(cmd.Args, " "), time.Now().Sub(pre).Seconds()*1000)
		}
		if len(gs) > 0 {
			// when we are done with this block of work we need to reload
			// rdeps of the output packages to ensure they are still current
			rdeps := make(pkgSet)

			done := make(chan *pkg)

			type pkgState struct {
				pre     hashRes
				post    hashRes
				count   int
				pending bool
			}

			state := make(map[*pkg]*pkgState)

			for _, g := range gs {
				for rd := range g.rdeps {
					rdeps[rd] = true
				}

				// We initialise the pre snap to the zero snap which looks wrong
				// but is in fact right. Later we determine whether we have work to
				// do based on pre != post. If there is a diff, we set pre = post.
				// Hence we need to start with post being where we start.

				state[g] = &pkgState{
					pre:  g.zeroSnap(),
					post: g.snap(),
				}
			}

			gpre := time.Now()

			for {
				checkCount := 0
				for g, gs := range state {
					g := g
					gs := gs

					// TODO
					// we need to check that we can still proceed, i.e. we haven't "grown"
					// a new dependency that isn't ready

					if gs.pending {
						continue
					}

					if hashEquals, err := gs.pre.equals(gs.post); err != nil {
						fatalf("failed to compare hashes for %v: %v", g, err)
					} else if !hashEquals {
						gs.pre = gs.post
						gs.pending = true
						gs.count++

						if gs.count > *fLoopLimit {
							fatalf("%v exceeded loop limit", g)
						}

						// fire off work
						go func() {
							pre := time.Now()
							cmd := exec.Command("go", "generate", g.ImportPath)
							verbosef("go generate %v\n", g.ImportPath)
							out, err := cmd.CombinedOutput()
							if err != nil {
								fatalf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
							}

							verbosef("%v iteration %v (%.2fms)\n%s", strings.Join(cmd.Args, " "), gs.count, time.Now().Sub(pre).Seconds()*1000, out)
							done <- g
						}()
					} else {
						checkCount++
					}
				}

				if checkCount == len(state) {
					verbosef("go generate loop completed (%.2fms)\n", time.Now().Sub(gpre).Seconds()*1000)
					break
				}

				select {
				case g := <-done:
					state[g].pending = false

					// reload packages
					toReload := []string{g.ImportPath}
					for _, ods := range g.toolDeps {
						for od := range ods {
							toReload = append(toReload, od.ImportPath)
						}
					}

					debugf("Will reload %v\n", toReload)
					loadPkgs(toReload)

					state[g].post = g.snap()
				}
			}

			// now reload the rdeps
			var toReload []string
			for rd := range rdeps {
				toReload = append(toReload, rd.ImportPath)
			}
			debugf("Will reload %v\n", toReload)
			loadPkgs(toReload)
		}

		iwg.Wait()

		var possWork []*pkg
		var installs []string

		for _, p := range append(is, gs...) {
			p.donePending(p)
			if !p.isTool {
				installs = append(installs, p.ImportPath)
			}
			if !p.ready() {
				for pp := range p.pendingVal {
					debugf(" + %v\n", pp)
				}
				fatalf("%v is still pending on:\n", p)
			}

			debugf("%v marked as complete\n", p)

			for rd := range p.rdeps {
				rd.donePending(p)
				if rd.ready() {
					possWork = append(possWork, rd)
				}
			}
		}

		var pw *pkg
		for len(possWork) > 0 {
			pw, possWork = possWork[0], possWork[1:]
			if pw.isTool || len(pw.toolDeps) > 0 {
				debugf("adding work %v\n", pw)
				work = append(work, pw)
			} else {
				// this is a package which exists as a transitive dep
				pw.donePending(pw)
				if !pw.isTool {
					installs = append(installs, pw.ImportPath)
				}
				for rd := range pw.rdeps {
					rd.donePending(pw)
					if rd.ready() {
						possWork = append(possWork, rd)
					}
				}
			}
		}

		// we don't care if this install works... it's just a temporary fix to speed
		// up subsequent type checks
		args := []string{"go", "install"}
		args = append(args, installs...)
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				fatalf("failed to try %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
			}
		}
	}

	for _, p := range pkgs {
		if !p.ready() {
			debugf("%v is not ready\n", p)
		}
	}
}
