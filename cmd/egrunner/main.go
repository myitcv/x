// egrunner runs bash scripts in a Docker container to help with creating reproducible examples.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"mvdan.cc/sh/syntax"
	"myitcv.io/cmd/internal/bindmnt"
)

var (
	debugOut = false
	stdOut   = false
)

const (
	debug = false

	scriptName      = "script.sh"
	blockPrefix     = "block:"
	outputSeparator = "============================================="
	commentStart    = "**START**"

	commentEnvSubstAdj = "egrunner_envsubst:"
	commentRewrite     = "egrunner_rewrite:"

	commgithubcli = "githubcli"

	outJson  = "json"
	outStd   = "std"
	outDebug = "debug"
)

func main() { os.Exit(main1()) }

func main1() int {
	err := mainerr()
	if err == nil {
		return 0
	}
	switch err := err.(type) {
	case usageErr:
		fmt.Fprintln(os.Stderr, err.Error())
		err.flagSet.Usage()
		return 2
	case flagErr:
		return 2
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}

type context struct {
	fDockerRunFlags   dockerFlags
	fDockerBuildFlags dockerFlags

	fDebug      *bool
	fOut        *string
	fGoRoot     *string
	fGoProxy    *string
	fGithubCLI  *string
	fEnvSubVars *string
	fUID        *bool
	fGID        *bool

	dockerfile string
	script     string
}

func mainerr() error {
	fs := flag.NewFlagSet("flags", flag.ContinueOnError)
	fs.Usage = usage{fs}.usage
	c := &context{
		fDebug:      fs.Bool("debug", false, "Print debug information for egrunner"),
		fOut:        fs.String("out", "json", "output format; json(default)|debug|std"),
		fGoRoot:     fs.String("goroot", os.Getenv("EGRUNNER_GOROOT"), "path to GOROOT to use"),
		fGoProxy:    fs.String("goproxy", os.Getenv("EGRUNNER_GOPROXY"), "path to GOPROXY to use"),
		fGithubCLI:  fs.String("githubcli", "", "path to githubcli program"),
		fEnvSubVars: fs.String("envsubst", "HOME,GITHUB_ORG,GITHUB_USERNAME", "comma-separated list of env vars to expand in commands"),
		fUID:        fs.Bool("uid", false, "Set UID as a build arg for docker build"),
		fGID:        fs.Bool("gid", false, "Set GID as a build arg for docker build"),
	}
	fs.Var(&c.fDockerRunFlags, "drf", "flag to pass to docker run")
	fs.Var(&c.fDockerBuildFlags, "dbf", "flag to pass to docker build")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return flagErr(err.Error())
	}

	args := fs.Args()
	if len(args) != 2 {
		return usageErr{"incorrect arguments", fs}
	}

	c.dockerfile = args[0]
	c.script = args[1]

	return c.run()
}

func (c *context) run() error {
	type rewrite struct {
		p *regexp.Regexp
		r string
	}

	var rewrites []rewrite
	envsubvars := strings.Split(*c.fEnvSubVars, ",")

	applyRewrite := func(s string) string {
		for _, r := range rewrites {
			s = r.p.ReplaceAllString(s, r.r)
		}

		return s
	}

	switch *c.fOut {
	case outJson, outStd, outDebug:
	default:
		return errorf("unknown option to -out: %v", *c.fOut)
	}

	debugOut = *c.fOut == outDebug || debug || *c.fDebug
	if !debugOut {
		stdOut = *c.fOut == outStd
	}

	toRun := new(bytes.Buffer)
	toRun.WriteString(`#!/usr/bin/env bash
set -u
set -o pipefail

assert()
{
  E_PARAM_ERR=98
  E_ASSERT_FAILED=99

  if [ -z "$2" ]
  then
    exit $E_PARAM_ERR
  fi

  lineno=$2

  if [ ! $1 ]
  then
    echo "Assertion failed:  \"$1\""
    echo "File \"$0\", line $lineno"
    exit $E_ASSERT_FAILED
  fi
}

catfile()
{
	echo "\$ cat $1"
	cat "$1"
}

comment()
{
	if [ "$#" -eq 0 ] || [ "$1" == "" ]
	then
		echo ""
	else
		echo "$1" | fold -w 100 -s | sed -e 's/^$/#/' | sed -e 's/^\([^#]\)/# \1/'
	fi
}

`)

	var ghcli string
	if *c.fGithubCLI != "" {
		if abs, err := filepath.Abs(*c.fGithubCLI); err == nil {
			ghcli = abs
		}
	} else {
		// this is a fallback in case any lookups via gobin fail
		ghcli, _ = exec.LookPath(commgithubcli)

		gobin, err := exec.LookPath("gobin")
		if err != nil {
			goto FinishedLookupGithubCLI
		}

		mbin := exec.Command(gobin, "-mod=readonly", "-p", "myitcv.io/cmd/githubcli")
		if mout, err := mbin.Output(); err == nil {
			ghcli = string(mout)
			goto FinishedLookupGithubCLI
		}

		gbin := exec.Command(gobin, "-nonet", "-p", "myitcv.io/cmd/githubcli")
		if gout, err := gbin.Output(); err == nil {
			ghcli = string(gout)
		}
	}

FinishedLookupGithubCLI:

	ghcli = strings.TrimSpace(ghcli)

	if ghcli != "" {
		// ghcli could still be empty at this point. We do nothing
		// because it's not guaranteed that it is required in the script.
		// Hence we let that error happen if and when it does and the user
		// will be able to work it out (hopefully)
	}

	fn := c.script

	fi, err := os.Open(fn)
	if err != nil {
		return errorf("failed to open %v: %v", fn, err)
	}

	f, err := syntax.NewParser(syntax.KeepComments).Parse(fi, fn)
	if err != nil {
		return errorf("failed to parse %v: %v", fn, err)
	}

	var last *syntax.Pos
	var b *block

	// blocks is a mapping from statement index to *block this allows us to take
	// the output from each statement and not only assign it to the
	// corresponding index but also add to the block slice too (if the block is
	// defined)
	seenBlocks := make(map[block]bool)

	p := syntax.NewPrinter()

	stmtString := func(s *syntax.Stmt) string {
		// temporarily "blank" the comments associated with the stmt
		cs := s.Comments
		s.Comments = nil
		var b bytes.Buffer
		p.Print(&b, s)
		s.Comments = cs
		return b.String()
	}

	type cmdOutput struct {
		Cmd string
		Out string
	}

	var stmts []*cmdOutput
	blocks := make(map[block][]*cmdOutput)

	pendingSep := false

	// find the # START comment
	var start *syntax.Comment

	// TODO it would be significantly cleaner if we grouped, tidied etc all the statements
	// and comments into a custom data structure in one phase, then processed it in another.
	// The mixing of logic below is hard to read. Not to mention much more efficient.

	// process handles comment blocks and any special instructions within them
	process := func(cb []syntax.Comment) error {
		for _, c := range cb {
			l := strings.TrimSpace(c.Text)
			switch {
			case strings.HasPrefix(l, commentEnvSubstAdj):
				l := strings.TrimPrefix(l, commentEnvSubstAdj)
				for _, d := range strings.Fields(l) {
					a, d := d[0], d[1:]
					if len(d) == 0 {
						return errorf("envsubst adjustment invalid: %q", l)
					}

					switch a {
					case '+':
						envsubvars = append(envsubvars, d)
					case '-':
						nv := envsubvars[:0]
						for _, v := range envsubvars {
							if v != d {
								nv = append(nv, v)
							}
						}
						envsubvars = nv
					default:
						return errorf("envsubst adjustment invalid: %q", l)
					}
				}
			case strings.HasPrefix(l, commentRewrite):
				l := strings.TrimPrefix(l, commentRewrite)
				fs, err := splitQuotedFields(l)
				if err != nil {
					return errorf("failed to handle arguments for rewrite %q: %v", l, err)
				}
				if len(fs) != 2 {
					return errorf("rewrite expects exactly 2 (quoted) arguments; got %v from %q", len(fs), l)
				}
				p, err := regexp.Compile(fs[0])
				if err != nil {
					return errorf("failed to compile rewrite regexp %q: %v", fs[0], err)
				}
				rewrites = append(rewrites, rewrite{p, fs[1]})
			}
		}

		return nil
	}

	for si, s := range f.Stmts {

		lastNonBlank := uint(0)
		if last != nil {
			lastNonBlank = last.Line()
		}
		var commBlock []syntax.Comment
		for _, c := range s.Comments {
			if start == nil {
				if s.Pos().After(c.End()) {
					if strings.TrimSpace(c.Text) == commentStart {
						start = &c
					}
				}
			}

			// commBlock != nil indicates we have started adding comments to a block
			// The end of the block is makred by a blank line.

			// Work out whether we have passed a blank line.
			if c.Pos().Line() > lastNonBlank+1 {
				if err := process(commBlock); err != nil {
					return err
				}
				commBlock = make([]syntax.Comment, 0)
			}

			if commBlock != nil {
				// this comment is contiguous with last in existing comment
				commBlock = append(commBlock, c)
			}
			lastNonBlank = c.End().Line()
		}
		if s.Pos().Line() > lastNonBlank+1 {
			if err := process(commBlock); err != nil {
				return err
			}
		}

		if start == nil || start.Pos().After(s.Pos()) {
			continue
		}
		setBlock := false
		for _, c := range s.Comments {
			if s.Pos().After(c.End()) && s.Pos().Line()-1 == c.End().Line() {
				l := strings.TrimSpace(c.Text)
				if strings.HasPrefix(l, blockPrefix) {
					v := block(strings.TrimSpace(strings.TrimPrefix(l, blockPrefix)))
					if seenBlocks[v] {
						return errorf("block %q used multiple times", v)
					}
					seenBlocks[v] = true
					b = &v
					setBlock = true
				}
			}
		}
		if !setBlock {
			if last != nil && last.Line()+1 != s.Pos().Line() {
				// end of contiguous block
				b = nil
			}
		}
		isAssert := false
		if ce, ok := s.Cmd.(*syntax.CallExpr); ok {
			isAssert = len(ce.Args) > 0 && ce.Args[0].Lit() == "assert"
		}
		nextIsAssert := false
		if si < len(f.Stmts)-1 {
			s := f.Stmts[si+1]
			if ce, ok := s.Cmd.(*syntax.CallExpr); ok {
				nextIsAssert = len(ce.Args) > 0 && ce.Args[0].Lit() == "assert"
			}
		}

		if isAssert {
			// TODO improve this by actually inspecting the second argument
			// to assert
			l := stmtString(s)
			l = strings.Replace(l, "$LINENO", fmt.Sprintf("%v", s.Pos().Line()), -1)
			fmt.Fprintf(toRun, "%v\n", l)
		} else {
			co := &cmdOutput{
				Cmd: stmtString(s),
			}

			if pendingSep && !stdOut {
				fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
			}
			var envsubvarsstr string
			if len(envsubvars) > 0 {
				envsubvarsstr = "$" + strings.Join(envsubvars, ",$")
			}
			if !stdOut {
				fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH' | envsubst '%v' \n%v\nTHISWILLNEVERMATCH\n", envsubvarsstr, stmtString(s))
				fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
			}
			stmts = append(stmts, co)
			if debugOut || (stdOut && b != nil) {
				fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH' | envsubst '%v' \n$ %v\nTHISWILLNEVERMATCH\n", envsubvarsstr, stmtString(s))
			}
			fmt.Fprintf(toRun, "%v\n", stmtString(s))

			// if this statement is not an assert, and the next statement is
			// not an assert, then we need to inject an assert that corresponds
			// to asserting a zero exit code
			if !nextIsAssert {
				fmt.Fprintf(toRun, "assert \"$? -eq 0\" %v\n", s.Pos().Line())
			}

			pendingSep = true

			if b != nil {
				blocks[*b] = append(blocks[*b], co)
			}
		}

		// now calculate the last line of this statement, including heredocs etc

		// TODO this might need to be better
		end := s.End()
		for _, r := range s.Redirs {
			if r.End().After(end) {
				end = r.End()
			}
			if r.Hdoc != nil {
				if r.Hdoc.End().After(end) {
					end = r.Hdoc.End()
				}
			}
		}
		last = &end
	}

	if pendingSep {
		fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
	}

	debugf("finished compiling script: \ns%v\n", toRun.String())

	// docker requires the file/directory we are mapping to be within our
	// home directory because of "security"
	tf, err := ioutil.TempFile("", ".go_modules_by_example")
	if err != nil {
		return errorf("failed to create temp file: %v", err)
	}

	tfn := tf.Name()

	defer func() {
		debugf("Removing temp script %v\n", tf.Name())
		os.Remove(tf.Name())
	}()

	if err := ioutil.WriteFile(tfn, toRun.Bytes(), 0644); err != nil {
		return errorf("failed to write to temp file %v: %v", tfn, err)
	}

	debugf("wrote script to %v\n", tfn)

	if etfn, err := bindmnt.Resolve(tfn); err == nil {
		tfn = etfn
	}

	debugf("script will map from %v to %v\n", tfn, scriptName)

	args := []string{"docker", "run", "--rm", "-w", "/home/gopher", "-e", "GITHUB_PAT", "-e", "GITHUB_USERNAME", "-e", "GO_VERSION", "-e", "GITHUB_ORG", "-e", "GITHUB_ORG_ARCHIVE", "--entrypoint", "bash", "-v", fmt.Sprintf("%v:/%v", tfn, scriptName)}

	if ghcli != "" {
		if eghcli, err := bindmnt.Resolve(ghcli); err == nil {
			args = append(args, "-v", fmt.Sprintf("%v:/go/bin/%v", eghcli, commgithubcli))
		}
	}

	for _, df := range fDockerRunFlags {
		parts := strings.SplitN(df, "=", 2)
		switch len(parts) {
		case 1:
			args = append(args, parts[0])
		case 2:
			flag, value := parts[0], parts[1]
			if flag == "-v" {
				vparts := strings.Split(value, ":")
				if len(vparts) != 2 {
					return errorf("-v flag had unexpected format: %q", value)
				}
				src := vparts[0]
				if esrc, err := bindmnt.Resolve(src); err == nil {
					value = esrc + ":" + vparts[1]
				}
			}
			args = append(args, flag, value)
		default:
			panic("invariant fail")
		}
	}

	if *c.fGoRoot != "" {
		if egr, err := bindmnt.Resolve(*c.fGoRoot); err == nil {
			args = append(args, "-v", fmt.Sprintf("%v:/go", egr))
		}
	}

	if filepath.IsAbs(*c.fGoProxy) {
		egp, err := bindmnt.Resolve(*c.fGoProxy)
		if err != nil {
			return fmt.Errorf("failed to resolve bindmnt resolve %v: %v", *c.fGoProxy, err)
		}
		args = append(args, "-v", fmt.Sprintf("%v:/goproxy", egp), "-e", "GOPROXY=file:///goproxy")
	} else {
		args = append(args, "-e", "GOPROXY="+*c.fGoProxy)
	}

	// build docker image
	{
		td, err := ioutil.TempDir("", "egrunner-docker-build")
		if err != nil {
			return errorf("failed to create temp dir for docker build: %v", err)
		}
		defer func() {
			debugf("Removing temp dir %v\n", td)
			os.RemoveAll(td)
		}()
		idf, err := os.Open(c.dockerfile)
		if err != nil {
			return errorf("failed to open Docker file %v: %v", c.dockerfile, err)
		}
		odfn := filepath.Join(td, "Dockerfile")
		odf, err := os.Create(odfn)
		if err != nil {
			return errorf("failed to create temp Dockerfile %v: %v", odfn, err)
		}
		if _, err := io.Copy(odf, idf); err != nil {
			return errorf("failed to copy %v to %v: %v", c.dockerfile, odfn, err)
		}
		if err := odf.Close(); err != nil {
			return errorf("failed to close %v: %v", odfn, err)
		}

		buildArgs := []string{"docker", "build", "-q"}
		if *c.fUID {
			buildArgs = append(buildArgs, "--build-arg=UID="+strconv.Itoa(os.Getuid()))
		}
		if *c.fGID {
			buildArgs = append(buildArgs, "--build-arg=GID="+strconv.Itoa(os.Getgid()))
		}
		buildArgs = append(buildArgs, td)

		var stdout, stderr bytes.Buffer
		dbcmd := exec.Command(buildArgs[0], buildArgs[1:]...)
		dbcmd.Stdout = &stdout
		dbcmd.Stderr = &stderr
		debugf("building docker image with %v\n", strings.Join(dbcmd.Args, " "))
		if err := dbcmd.Run(); err != nil {
			return errorf("failed to run %v: %v\n%s", strings.Join(dbcmd.Args, " "), err, stderr.String())
		}

		iid := strings.TrimSpace(stdout.String())

		args = append(args, iid)
	}

	args = append(args, fmt.Sprintf("/%v", scriptName))

	cmd := exec.Command(args[0], args[1:]...)
	debugf("now running %v via %v\n", tfn, strings.Join(cmd.Args, " "))

	if debugOut || stdOut {
		cmdout, err := cmd.StdoutPipe()
		if err != nil {
			return errorf("failed to create cmd stdout pipe: %v", err)
		}

		var scanerr error
		done := make(chan bool)

		scanner := bufio.NewScanner(cmdout)
		go func() {
			for scanner.Scan() {
				fmt.Println(applyRewrite(scanner.Text()))
			}
			if err := scanner.Err(); err != nil {
				scanerr = err
			}
			close(done)
		}()

		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return errorf("failed to start %v: %v", strings.Join(cmd.Args, " "), err)
		}
		<-done
		if err := cmd.Wait(); err != nil {
			return errorf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
		}
		if scanerr != nil {
			return errorf("failed to rewrite output: %v", scanerr)
		}
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	cur := new(strings.Builder)
	for scanner.Scan() {
		l := scanner.Text()
		if l == outputSeparator {
			lines = append(lines, cur.String())
			cur = new(strings.Builder)
			continue
		}
		cur.WriteString(applyRewrite(l))
		cur.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return errorf("error scanning cmd output: %v", err)
	}

	if len(lines) != 2*len(stmts) {
		return errorf("had %v statements but %v lines of output", len(stmts), len(lines))
	}

	j := 0
	for i := 0; i < len(lines); {
		// strip the last \n off the cmd
		stmts[j].Cmd = lines[i][:len(lines[i])-1]
		i += 1
		stmts[j].Out = lines[i]
		i += 1
		j += 1
	}

	tmpl := struct {
		Stmts  []*cmdOutput
		Blocks map[block][]*cmdOutput
	}{
		Stmts:  stmts,
		Blocks: blocks,
	}

	byts, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return errorf("error marshaling JSON: %v", err)
	}

	fmt.Printf("%s\n", byts)

	return nil
}

func splitQuotedFields(s string) ([]string, error) {
	// Split fields allowing '' or "" around elements.
	// Quotes further inside the string do not count.
	var f []string
	for len(s) > 0 {
		for len(s) > 0 && isSpaceByte(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		// Accepted quoted string. No unescaping inside.
		if s[0] == '"' || s[0] == '\'' {
			quote := s[0]
			s = s[1:]
			i := 0
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, errorf("unterminated %c string", quote)
			}
			f = append(f, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isSpaceByte(s[i]) {
			i++
		}
		f = append(f, s[:i])
		s = s[i:]
	}
	return f, nil
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func errorf(format string, args ...interface{}) error {
	if debugOut {
		panic(fmt.Errorf(format, args...))
	}
	return fmt.Errorf(format, args...)
}

func debugf(format string, args ...interface{}) {
	if debugOut {
		fmt.Fprintf(os.Stderr, "+ "+format, args...)
	}
}
