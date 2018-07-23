// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

// gitgodoc allows you to view `godoc` documentation for different branches of a Git repository
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// TODO make cross-platform

// TODO implement pruning of remote branches that no longer exist (we should be able to detect after each
// push)

type branchPorts map[string]uint
type branchLocks map[string]*sync.Mutex

const (
	refCopy               = "@refcopy"
	debug                 = false
	jqueryReplacementFile = `<script src="/__static/jquery-2.0.3.min.js"></script>`
	additionalScriptTags  = `<script src="$1/godocs.js"></script>
 						 	<script src="/__static/bootstrap.min.js"></script>
							<script src="/__static/site.js"></script>`
	additionalStylesheets = `<link type="text/css" rel="stylesheet" href="$1/style.css">
						  	 <link type="text/css" rel="stylesheet" href="/__static/bootstrap.min.css">
							 <link type="text/css" rel="stylesheet" href="/__static/site.css">`
)

var (
	validBranch    = regexp.MustCompile("^[a-zA-Z0-9_-]+$")
	href           = regexp.MustCompile(`(href|src)="/("|[^/][^"]*")`)
	footerTag      = regexp.MustCompile(`<div id="footer">`)
	godocjs        = regexp.MustCompile(`<script type="text\/javascript" src="([\/a-z0-9_-]+)\/godocs\.js"><\/script>`)
	jqueryFilename = regexp.MustCompile(`<script type="text\/javascript" src="([\/a-z0-9_-]+)\/jquery\.js"><\/script>`)
	stylecss       = regexp.MustCompile(`<link type="text\/css" rel="stylesheet" href="([\/a-z0-9_-]+)\/style\.css">`)

	fServeFrom = flag.String("serveFrom", "", "directory to use as a working directory")
	fRepo      = flag.String("repo", "", "git url from which to clone")
	fPkg       = flag.String("pkg", "", "the package the repo represents")
	fGoPath    = flag.String("gopath", "", "a relative GOPATH that will be used when running the godoc server (prepended with the repo dir)")
	fPortStart = flag.Uint("port", 8080, "the port on which to serve; controlled godoc instances will be started on subsequent ports")
)

// TODO this should be split into separate types and the correct
// type used based on the header sent from Gitlab
type gitLabWebhook struct {
	ObjectKind       string `json:"object_kind"`
	Ref              string `json:"ref"`
	ObjectAttributes struct {
		State        string `json:"state"`
		MergeStatus  string `json:"merge_status"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
	} `json:"object_attributes"`
}

type server struct {
	repo, serveFrom, pkg, refCopyDir string
	gopath                           []string

	nextPort uint

	ports     atomic.Value
	portsLock sync.Mutex

	repos     atomic.Value
	reposLock sync.Mutex

	setupPhase bool
}

func main() {
	log.SetPrefix("")

	flag.Parse()

	s := newServer()

	s.setupRefCopy()
	s.setupPhase = false

	s.serve()
}

func (s *server) setupRefCopy() {
	infof("setting up reference copy of repo in %v", s.refCopyDir)

	s.reposLock.Lock()
	defer s.reposLock.Unlock()

	if _, err := os.Stat(filepath.Join(s.refCopyDir, ".git")); err != nil {
		// the copy does not exist
		// no need to do a fetch; instead we need to clone

		git(s.serveFrom, "clone", s.repo, s.refCopyDir)
		git(s.refCopyDir, "checkout", "-f", "origin/master")
	} else {
		git(s.refCopyDir, "fetch", "-p")
		git(s.refCopyDir, "checkout", "-f", "origin/master")
	}

	// TODO make this configurable/better

	remotes := git(s.refCopyDir, "branch", "-r")

	sc := bufio.NewScanner(bytes.NewBuffer(remotes))

	for sc.Scan() {
		line := strings.Fields(sc.Text())
		branch := filepath.Base(line[0])

		// special case
		if branch == "HEAD" {
			continue
		}

		s.setup(branch, true)
	}

	if err := sc.Err(); err != nil {
		fatalf("error scanning output of remote branch check: %v\n%v", err, string(remotes))
	}
}

// TODO break this apart
func (s *server) serve() {
	tr := &http.Transport{
		DisableCompression:  true,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 20,
	}
	client := &http.Client{Transport: tr}

	http.HandleFunc("/__static/", handleStaticFileRequests)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		debugf("%v\n", r.URL)

		if !path.IsAbs(r.URL.Path) {
			fatalf("expected absolute URL path, got %v", r.URL.Path)
		}

		if r.URL.Path == "/" {
			handleRootRequest(w, r, s)
			return
		}

		parts := strings.Split(r.URL.Path, "/")
		branch := parts[1]
		actUrl := path.Join(parts[2:]...)

		if !validBranch.MatchString(branch) {
			branch = "master"

			// now we need to redirect to the same URL but with a branch
			newUrl := *r.URL
			newUrl.Path = "/" + path.Join(branch, actUrl)
			http.Redirect(w, r, newUrl.String(), http.StatusFound)

			return
		}

		debugf("got request to %v; branch %v ", r.URL, branch)

		if be := s.branchExists(branch); !be {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Branch %v not found\n", branch)

			return
		}

		port := s.getPort(branch)

		scheme := r.URL.Scheme
		if scheme == "" {
			scheme = "http"
		}

		var host string
		if r.URL.Host != "" {
			hostParts := strings.Split(r.URL.Host, ":")
			host = hostParts[0]
		}
		ourl := *r.URL
		ourl.Scheme = scheme
		ourl.Host = fmt.Sprintf("%v:%v", host, port)
		ourl.Path = actUrl
		// ourl.RawQuery = "m=all"

		url := ourl.String()

		debugf("making onward request to %v", url)

		req, err := http.NewRequest(r.Method, url, r.Body)
		if err != nil {
			fatalf("could not create proxy request to %v: %v", url, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			fatalf("could not do proxy req to %v: %v", url, err)
		}

		isHtml := false

		wh := w.Header()
		for k, v := range resp.Header {
			if k == "Content-Type" {
				for _, vv := range v {
					ct := strings.Split(vv, ";")
					for _, p := range ct {
						p = strings.TrimSpace(p)
						if p == "text/html" {
							isHtml = true
						}
					}
				}

			}
			wh[k] = v
		}

		w.WriteHeader(resp.StatusCode)

		if !isHtml {
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				fatalf("could not relay body in response to proxy req to %v: %v", url, err)
			}
		} else {
			// TODO shouldn't have to rewrite if we can get the godoc server to
			// use a unique Path root
			sc := bufio.NewScanner(resp.Body)

			repl := "$1=\"/" + branch + "/$2"

			for sc.Scan() {
				text := sc.Text()
				line := href.ReplaceAllString(text, repl)

				onlyText := strings.TrimSpace(line) // trimming space since they may interfere with hasPrefix or hasSuffix function

				switch {
				case matchesBodyStart(line):
					line = `<body>
					<div style="position:fixed; top:0;right:0;"><a href="?m=all">Show All</a></div>
					`

				case matchesStylesheets(onlyText):
					// append additional stylesheets
					line = stylecss.ReplaceAllString(line, additionalStylesheets)

				case matchesScriptTags(onlyText):
					// append script tags after the last script tag
					line = godocjs.ReplaceAllString(line, additionalScriptTags)

				case matchesFooterTag(onlyText):
					// append the modal divs after the footer
					modalHtmlString := staticFileMap["modalHTMLFragment"]
					line = footerTag.ReplaceAllString(line, modalHtmlString)

				case matchesJqueryFile(onlyText):
					// replace jquery with newer version
					line = jqueryFilename.ReplaceAllString(line, jqueryReplacementFile)
				}

				fmt.Fprintln(w, line)
			}

			if err := sc.Err(); err != nil {
				fatalf("error scanning response body: %v", err)
			}
		}
	})

	url := fmt.Sprintf(":%v", s.nextPort)

	infof("starting server %v", url)

	err := http.ListenAndServe(url, nil)
	if err != nil {
		fatalf("failed to start main server on %v: %v", url, err)
	}
}

func newServer() *server {
	if *fServeFrom == "" || *fRepo == "" || *fPkg == "" {
		flag.Usage()
		os.Exit(1)
	}

	// TODO validate *fPkg

	s := &server{
		nextPort: *fPortStart,
		pkg:      *fPkg,
		repo:     *fRepo,

		setupPhase: true,
	}

	serveFrom, err := filepath.Abs(*fServeFrom)
	if err != nil {
		fatalf("could not make absolute filepath from %v: %v", *fServeFrom, err)
	}

	if sd, err := os.Stat(serveFrom); err == nil {
		if !sd.IsDir() {
			fatalf("%v exists but is not a directory", serveFrom)
		}
	} else {
		err := os.Mkdir(serveFrom, 0700)
		if err != nil {
			fatalf("could not mkdir %v: %v", serveFrom, err)
		}
	}
	s.serveFrom = serveFrom

	if *fGoPath != "" {
		parts := filepath.SplitList(*fGoPath)

		for _, p := range parts {
			if filepath.IsAbs(p) {
				fatalf("gopath flag contains non-relative part %v", p)
			}

			s.gopath = append(s.gopath, p)
		}
	} else {
		s.gopath = []string{""}
	}

	s.refCopyDir = filepath.Join(s.serveFrom, refCopy)

	s.ports.Store(make(branchPorts))
	s.repos.Store(make(branchLocks))

	return s
}

func (s *server) getPort(branch string) uint {
	m := s.ports.Load().(branchPorts)
	port, gdRunning := m[branch]

	if gdRunning {
		return port
	}

	s.portsLock.Lock()
	defer s.portsLock.Unlock()

	// check again
	m = s.ports.Load().(branchPorts)
	port, gdRunning = m[branch]

	if gdRunning {
		return port
	}

	s.nextPort++
	port = s.nextPort

	mp := make(branchPorts)
	for k, v := range m {
		mp[k] = v
	}
	mp[branch] = port
	defer s.ports.Store(mp)

	s.runGoDoc(branch, port)

	// TODO this is pretty gross
	backoff := 50 * time.Millisecond
	attempt := 1
	for {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%v", port))

		if err == nil {
			if resp.StatusCode == 200 {
				break
			}
		}

		attempt++

		if attempt == 10 {
			fatalf("could not connect to godoc instance on branch %v (port %v) after 10 attempts", branch, port)
		}

		infof("could not connect to godoc instance on branch %v (port %v); retrying attempt %v; will sleep for %v", branch, port, attempt, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	// now "signal" we are done setting up this server

	return port
}

func (s *server) branchExists(branch string) bool {
	m := s.repos.Load().(branchLocks)
	_, ok := m[branch]

	return ok
}

func (s *server) setup(branch string, cloneIfMissing bool) bool {
	m := s.repos.Load().(branchLocks)
	_, ok := m[branch]

	if ok {
		return true
	}

	if !s.setupPhase {
		s.reposLock.Lock()
		defer s.reposLock.Unlock()
	}

	//check again now we are critical
	m = s.repos.Load().(branchLocks)
	_, ok = m[branch]

	if ok {
		return true
	}

	ml := make(branchLocks)
	for k, v := range m {
		ml[k] = v
	}
	ml[branch] = new(sync.Mutex)
	defer s.repos.Store(ml)

	ct := s.gitDir(branch)

	if _, err := os.Stat(filepath.Join(ct, ".git")); err == nil {
		return true
	}

	if !cloneIfMissing {
		infof("branch %v does not exist; told not to copy", branch)
		return false
	}

	// keep the refCopy fresh
	if !s.setupPhase {
		git(s.refCopyDir, "fetch")
	}

	// ensure the path to the target exists
	p := filepath.Dir(ct)
	err := os.MkdirAll(p, 0700)
	if err != nil {
		fatalf("could not create branch directory structure %v: %v", p, err)
	}

	// TODO - this is not cross platform
	infof("copying refcopy cp -pr %v %v", s.refCopyDir, ct)
	cmd := exec.Command("cp", "-rp", s.refCopyDir, ct)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fatalf("could not copy refcopy %v to branch: %v\n%v", ct, err, string(out))
	}

	// now ensure we're on that branch in the copy
	git(ct, "checkout", "-f", "origin/"+branch)

	// now "signal" that we are done setting up this branch

	return false
}

func (s *server) fetch(branch string) {
	exists := s.setup(branch, true)
	if exists {
		m := s.repos.Load().(branchLocks)
		pl, ok := m[branch]

		if !ok {
			fatalf("no lock exists for branch?")
		}

		pl.Lock()
		defer pl.Unlock()

		gd := s.gitDir(branch)

		git(gd, "fetch")
		git(gd, "checkout", "-f", "origin/"+branch)
	}
}

func (s *server) branchDir(branch string) string {
	return filepath.Join(s.serveFrom, branch)
}

func (s *server) gitDir(branch string) string {
	return filepath.Join(s.branchDir(branch), "src", s.pkg)
}

func (s *server) buildGoPath(branch string) string {
	var res string
	sep := ""
	for _, p := range s.gopath {
		res = res + sep + filepath.Join(s.serveFrom, branch, p)
		sep = string(filepath.ListSeparator)
	}

	return res
}

func git(cwd string, args ...string) []byte {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd

	infof("running git %v (CWD %v)", strings.Join(args, " "), cwd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		fatalf("failed to run command: git %v: %v\n%v", strings.Join(args, " "), err, string(out))
	}

	infof("done")

	return out
}

func (s *server) runGoDoc(branch string, port uint) {
	gp := s.buildGoPath(branch)

	env := []string{
		"GOROOT=" + runtime.GOROOT(),
		"GOPATH=" + gp,
	}
	cmd := exec.Command("go", "install", "golang.org/x/tools/cmd/godoc")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		fatalf("could not go install golang.org/x/tools/cmd/godoc: %v\n%v", err, string(out))
	}

	ctxt := build.Default
	ctxt.GOPATH = gp

	pkg, err := ctxt.Import("golang.org/x/tools/cmd/godoc", "", 0)
	if err != nil {
		fatalf("could not go build details fo golang.org/x/tools/cmd/godoc: %v", err)
	}

	gdp := filepath.Join(pkg.BinDir, "godoc")

	// run the godoc instance
	go func() {
		attrs := &syscall.SysProcAttr{
			Pdeathsig: syscall.SIGTERM,
		}

		portStr := fmt.Sprintf(":%v", port)
		cmd := exec.Command(gdp, "-http", portStr)
		cmd.Env = env
		cmd.SysProcAttr = attrs

		infof("starting %v server on port %v with env %v", gdp, portStr, env)

		out, err = cmd.CombinedOutput()
		if err != nil {
			fatalf("%v -http %v failed: %v\n%v", gdp, portStr, err, string(out))
		}
	}()
}

func infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func debugf(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}

func fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func handleStaticFileRequests(res http.ResponseWriter, req *http.Request) {
	debugf("serving static content for: %v", req.URL.Path)

	switch req.URL.String() {
	case "/__static/jquery-2.0.3.min.js":
		content := staticFileMap["jquery-2.0.3.min.js"]
		res.Header().Add("Content-Type", "application/javascript")
		res.Write([]byte(content))
	case "/__static/bootstrap.min.js":
		content := staticFileMap["bootstrap.min.js"]
		res.Header().Add("Content-Type", "application/javascript")
		res.Write([]byte(content))
	case "/__static/bootstrap.min.css":
		content := staticFileMap["bootstrap.min.css"]
		res.Header().Add("Content-Type", "text/css; charset=utf-8")
		res.Write([]byte(content))
	case "/__static/site.js":
		content := staticFileMap["site.js"]
		res.Header().Add("Content-Type", "application/javascript")
		res.Write([]byte(content))
	case "/__static/site.css":
		content := staticFileMap["site.css"]
		res.Header().Add("Content-Type", "text/css; charset=utf-8")
		res.Write([]byte(content))
	}
}

func handleRootRequest(res http.ResponseWriter, req *http.Request, s *server) {
	if req.Method == http.MethodPost {
		if vs, ok := req.URL.Query()["refresh"]; ok {

			// TODO support more than just gitlab
			if len(vs) == 1 && vs[0] == "gitlab" {
				// we need to parse the branch from the request

				body, err := ioutil.ReadAll(req.Body)
				if err != nil {
					fatalf("failed to read body of POST request")
				}
				var pHook gitLabWebhook
				err = json.Unmarshal(body, &pHook)

				if err != nil {
					fatalf("could not decode Gitlab web")
				}

				switch pHook.ObjectKind {
				case "merge_request":
					s.fetch(pHook.ObjectAttributes.SourceBranch)

					infof("got a request to refresh branch %v in response to a merge request hook with target %v and state %v", pHook.ObjectAttributes.SourceBranch, pHook.ObjectAttributes.TargetBranch, pHook.ObjectAttributes.State)

				case "push":
					ref := strings.Split(pHook.Ref, "/")
					if len(ref) != 3 {
						fatalf("did not understand format of branch: %v", pHook.Ref)
					}

					branch := ref[2]

					infof("got a request to refresh branch %v in response to a push hook", branch)

					s.fetch(branch)
				default:
					res.WriteHeader(http.StatusInternalServerError)

					msg := fmt.Sprintf("Did not understand Gitlab refresh request; unknown object_kind: %v", pHook.ObjectKind)
					infof(msg)
					fmt.Fprintln(res, msg)

					return
				}

				res.WriteHeader(http.StatusOK)
				res.Write([]byte("OK\n"))

				return
			} else if len(vs) == 1 && vs[0] == "gitlab_merge_request" {

			} else {
				res.WriteHeader(http.StatusInternalServerError)

				msg := fmt.Sprintf("Did not understand refresh request: %v", req.URL)
				infof(msg)
				fmt.Fprintln(res, msg)

				return
			}
		}
	}

	// we should serve a simple page of links to existing branches
	res.WriteHeader(http.StatusOK)
	var bs []string
	m := s.repos.Load().(branchLocks)
	for k := range m {
		bs = append(bs, k)
	}

	sort.Strings(bs)

	tmpl := struct {
		Branches []string
		Pkg      string
	}{
		Branches: bs,
		Pkg:      s.pkg,
	}

	t, err := template.New("webpage").Parse(homePageTemplate)
	if err != nil {
		fatalf("could not parse branch browser template: %v", err)
	}

	t.Execute(res, tmpl)
}

func matchesBodyStart(text string) bool {
	return strings.ToLower(text) == "<body>"
}

func matchesStylesheets(text string) bool {
	return strings.HasPrefix(text, `<link type="text/css" rel="stylesheet" href="`) && strings.HasSuffix(text, `style.css">`)
}

func matchesScriptTags(text string) bool {
	return strings.HasPrefix(text, `<script type="text/javascript" src="`) && strings.HasSuffix(text, `/godocs.js"></script>`)
}

func matchesFooterTag(text string) bool {
	return strings.Contains(text, `<div id="footer">`)
}

func matchesJqueryFile(text string) bool {
	return strings.HasPrefix(text, `<script type="text/javascript" src="`) && strings.HasSuffix(text, `/jquery.js"></script>`)
}
