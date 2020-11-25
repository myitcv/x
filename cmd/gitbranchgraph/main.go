package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync"
)

const (
	splitter             = "::::"
	archivedBranchPrefix = "zzz_"
)

type edge struct {
	from string
	to   string
}

var (
	fRemote = flag.String("remote", "origin", "which remote to treat as the reference point")
)

func main() {
	flag.Parse()

	currCommitCmd := exec.Command("git", "rev-parse", "HEAD")
	currCommitLines := mustlines(currCommitCmd)
	if len(currCommitLines) != 1 {
		panic(fmt.Errorf("expected a single line from %v: got %v", strings.Join(currCommitCmd.Args, " "), len(currCommitLines)))
	}
	currCommit := currCommitLines[0]

	refsCmd := exec.Command("git", "for-each-ref", "--format=%(objectname)"+splitter+"%(refname:short)")
	refsCmd.Args = append(refsCmd.Args, flag.Args()...)
	refs := mustlines(refsCmd)

	var wg sync.WaitGroup
	var dataLock sync.Mutex
	var edges []edge
	commitsToNames := make(map[string]map[string]bool)
	namesToCommits := make(map[string]string)
	for _, refline := range refs {
		parts := strings.Split(refline, splitter)
		if len(parts) != 2 {
			panic(fmt.Errorf("unexpected output line from %v: %q", strings.Join(refsCmd.Args, " "), refline))
		}
		_, refname := path.Split(parts[1])
		if strings.HasPrefix(refname, archivedBranchPrefix) {
			continue
		}
		commit := parts[0]

		wg.Add(1)
		go func() {
			// get the merge base
			gmbCmd := exec.Command("git", "merge-base", "--fork-point", *fRemote, commit)

			// default the merge-base to itself, because it we can't find a merge base
			// we just want to see whether we have any ref names for the commit
			gmb := commit
			gmbs, err := lines(gmbCmd)
			if err == nil {
				if len(gmbs) != 1 {
					panic(fmt.Errorf("got unexpected number of merge-base's from %v: %v; wanted 1", strings.Join(gmbCmd.Args, " "), len(gmbs)))
				}
				gmb = gmbs[0]
			}

			// get the chain from the merge base to the ref's commit
			logCmd := exec.Command("git", "log", "--pretty=%H"+splitter+"%D", "--first-parent", fmt.Sprintf("%v~1..%v", gmb, commit))
			var prevCommits []string
			loglines := mustlines(logCmd)
			dataLock.Lock()
			for i, log := range loglines {
				parts := strings.Split(log, splitter)
				if len(parts) != 1 && len(parts) != 2 {
					panic(fmt.Errorf("unexpected log format from %v: %q", strings.Join(logCmd.Args, " "), log))
				}
				c := parts[0]
				if _, ok := commitsToNames[c]; !ok {
					commitsToNames[c] = make(map[string]bool)
				}
				if len(parts) == 2 {
					refnameslist := parts[1]
					refnameslist = strings.Replace(refnameslist, "HEAD ->", "", -1)
					refnames := strings.Split(refnameslist, ",")
					for _, refname := range refnames {
						refname = strings.TrimSpace(refname)
						if refname == "" {
							continue
						}
						if strings.HasPrefix(refname, "tag: ") {
							continue
						}
						parts := strings.Split(refname, "/")
						if len(parts) > 1 && parts[len(parts)-2] != *fRemote {
							continue
						}
						commitsToNames[c][refname] = true
						namesToCommits[refname] = c
					}
				}
				prevCommits = append(prevCommits, c)
				if i > 0 {
					edges = append(edges, edge{
						from: prevCommits[i-1],
						to:   c,
					})
				}
			}
			dataLock.Unlock()
			wg.Done()
		}()
	}

	wg.Wait()

	// write the graph
	f := func(format string, args ...interface{}) {
		fmt.Printf(format, args...)
	}

	f("strict digraph G {\n")
	f("rankdir=LR;\n")
	f("node [ shape=box, fontname=Go, fontsize=10];\n")
	var commitList []string
	for commit := range commitsToNames {
		commitList = append(commitList, commit)
	}
	sort.Strings(commitList)
	for _, commit := range commitList {
		labels := commitsToNames[commit]
		label := commit[:12]
		fontColor := "fontcolor=grey"
		if len(labels) > 0 {
			var ll []string
			for l := range labels {
				if _, ok := namesToCommits[path.Base(l)]; ok {
					fontColor = "fontcolor=black"
				}
				ll = append(ll, l)
			}
			sort.Slice(ll, func(i, j int) bool {
				lhs, rhs := ll[i], ll[j]
				ldir, lname := path.Split(lhs)
				rdir, rname := path.Split(rhs)
				if ldir != "" && rdir == "" {
					return false
				}
				if ldir == "" && rdir != "" {
					return true
				}
				cmp := strings.Compare(ldir, rdir)
				if cmp == 0 {
					cmp = strings.Compare(lname, rname)
				}
				return cmp < 0
			})
			label = strings.Join(ll, "\n")
		}
		options := []string{fmt.Sprintf("label=%q", label), fontColor}
		if commit == currCommit {
			options = append(options, "fillcolor=red", "style=filled")
		}
		f("%q [%v]\n", commit, strings.Join(options, ","))
	}
	sort.Slice(edges, func(i, j int) bool {
		lhs, rhs := edges[i], edges[j]
		cmp := strings.Compare(lhs.from, rhs.from)
		if cmp == 0 {
			cmp = strings.Compare(lhs.to, rhs.to)
		}
		return cmp < 0
	})
	for _, edge := range edges {
		f("%q -> %q\n", edge.to, edge.from)
	}
	// draw edges where there is a difference between the remote
	// and local references
	for refname, commit := range namesToCommits {
		dir, name := path.Split(refname)
		if dir != "" {
			continue
		}
		// does the remote even exist?
		rcommit, ok := namesToCommits[path.Join(*fRemote, name)]
		if !ok {
			continue
		}
		if rcommit == commit {
			continue
		}
		f("%q -> %q [style=dashed, color=grey, arrowhead=none]\n", rcommit, commit)
	}
	f("}\n")
}

func lines(c *exec.Cmd) ([]string, error) {
	out, err := c.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run %v: %v\n%s", strings.Join(c.Args, " "), err, out)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

func mustlines(c *exec.Cmd) []string {
	lines, err := lines(c)
	if err != nil {
		panic(err)
	}
	return lines
}
