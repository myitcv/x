package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kr/fs"
)

var (
	fTarget = flag.String("target", "", "target directory for publishing")
)

func main() {
	// we trust that a go.mod file is accurate
	// i.e. that submodules are correctly nested and the respective go.mod files
	// reflect that

	flag.Parse()

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if *fTarget == "" {
		fmt.Fprintln(os.Stderr, "-target flag is required")
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	target, err := filepath.Abs(*fTarget)
	if err != nil {
		panic(err)
	}

	repoRoot := cwd
	var found bool

	for {
		fi, err := os.Stat(filepath.Join(repoRoot, ".git"))
		if err != nil {
			if os.IsNotExist(err) {
				rr := filepath.Dir(repoRoot)

				if rr == repoRoot {
					break
				}

				repoRoot = rr
			}

			panic(err)
		}

		if !fi.IsDir() {
			break
		}

		found = true

		break
	}

	if !found {
		panic(fmt.Errorf("Failed to find .git repo root from directory %v", cwd))
	}

	walker := fs.Walk(".")

	relRoot, err := filepath.Rel(repoRoot, cwd)
	if err != nil {
		panic(fmt.Errorf("unable to resolve relative path: %v", err))
	}

	type version struct {
		commitish string
		Version   string
		Name      string
		Short     string
		Time      time.Time
	}

	type module struct {
		path       string
		importPath string
		versions   []*version
		submodules []string
	}

	var modules []*module

	for walker.Step() {
		if err := walker.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		p := walker.Path()

		if p == "go.mod" || strings.HasSuffix(p, string(os.PathSeparator)+"go.mod") {

			m := &module{path: filepath.Join(relRoot, strings.TrimSuffix(p, "go.mod"))}

			// TODO using the go.mod parser when it exists
			// for now assume the file is well-formed
			gm, err := os.Open(p)
			if err != nil {
				panic(fmt.Errorf("failed to open %v: %v", p, err))
			}

			scanner := bufio.NewScanner(gm)

			for scanner.Scan() {
				// we expect module to be the first line
				line := scanner.Text()
				parts := strings.Split(line, " ")

				if len(parts) != 2 {
					panic(fmt.Errorf("go.mod file %v had unexpected first line: %v", p, line))
				}

				m.importPath = strings.Trim(parts[1], "\"")

				break
			}

			if err := scanner.Err(); err != nil {
				panic(fmt.Errorf("error reading version details: %v", err))
			}

			// now get the versions

			cmd := exec.Command("git", "log", "origin/master", "--first-parent", "--reverse", "--date=format-local:%Y-%m-%dT%H:%M:%SZ", "--pretty=format:%H %cd", ".")
			cmd.Env = append(os.Environ(), "TZ=UTC")
			cmd.Dir = filepath.Join(cwd, m.path)

			out, err := cmd.CombinedOutput()
			if err != nil {
				panic(fmt.Errorf("cmd failed: %v\n%v", err, string(out)))
			}

			r := bytes.NewReader(out)

			s := bufio.NewScanner(r)

			for s.Scan() {
				// each line will be a commit ref and time
				line := s.Text()

				parts := strings.Split(line, " ")

				if len(parts) != 2 {
					panic(fmt.Errorf("unexpected result line from git log: %v", line))
				}

				t, err := time.Parse(time.RFC3339, parts[1])
				if err != nil {
					panic(fmt.Errorf("failed to parse time %v: %v", parts[1], err))
				}

				ver := fmt.Sprintf("v0.0.0-%v-%v", t.Format("20060102150405"), parts[0])

				m.versions = append(m.versions, &version{
					commitish: parts[0],
					Version:   ver,
					Time:      t,
					Short:     ver,
					Name:      ver,
				})
			}

			if err := s.Err(); err != nil {
				panic(fmt.Errorf("error reading version details: %v", err))
			}

			modules = append(modules, m)
		}
	}

	if len(modules) == 0 {
		return
	}

	// sort modules based on depth

	sort.Slice(modules, func(i, j int) bool {
		lhs := modules[i].path
		rhs := modules[j].path

		if lhs == "." {
			return true
		}

		if rhs == "." {
			return false
		}

		return strings.Count(lhs, "/") < strings.Count(rhs, "/")
	})

	// as the controlling go routine ensure all the target directories exist
	// to avoid racing on those - versions will be mutually exclusive
	for i := range modules {
		m := modules[i]
		if err := os.MkdirAll(filepath.Join(target, m.importPath, "@v"), 0755); err != nil {
			panic(err)
		}

		for _, sm := range modules[i+1:] {
			if i == 0 {
				// cwd
				m.submodules = append(m.submodules, sm.path)
				continue
			}

			p := m.path + string(os.PathSeparator)
			if strings.HasPrefix(sm.path, p) {
				m.submodules = append(m.submodules, strings.TrimPrefix(sm.path, p))
			}
		}
	}

	fmt.Println("Found modules:")

	for _, m := range modules {
		// write the version file
		targetM := filepath.Join(target, m.importPath, "@v")
		vfName := filepath.Join(targetM, "list")
		vf, err := os.Create(vfName)
		if err != nil {
			panic(fmt.Errorf("failed to create version file %v: %v", vfName, err))
		}

		for _, v := range m.versions {
			fmt.Fprintf(vf, "%v %v\n", v.Version, v.Time.Format(time.RFC3339))

			targetV := filepath.Join(targetM, v.Version)

			func() {
				td, err := ioutil.TempDir("", "modpub-")
				if err != nil {
					panic(err)
				}

				defer os.RemoveAll(td)

				fmt.Printf("ImportPath: %v, Path: %v\n", m.importPath, m.path)

				for _, v := range m.versions {
					fmt.Printf("  version %v %v\n", v.Version, v.Time)
				}

				for _, sm := range m.submodules {
					fmt.Printf("  submodule %v\n", sm)
				}

				// git archive into the td; we use the cwd
				run("git archive %v | tar -C %q -x", v.commitish, td)

				// remove all submodules

				cd := filepath.Join(td, relRoot, m.path)

				for _, sm := range m.submodules {
					smpath := filepath.Join(cd, sm)
					if err := os.RemoveAll(smpath); err != nil {
						panic(err)
					}
				}

				// copy the .mod file
				run("cp %v %v", filepath.Join(cd, "go.mod"), targetV+".mod")

				// write the .info file
				infoFname := targetV + ".info"
				infoF, err := os.Create(infoFname)
				if err != nil {
					panic(fmt.Errorf("failed to create info file %v; %v", infoFname, err))
				}

				enc := json.NewEncoder(infoF)

				if err := enc.Encode(v); err != nil {
					panic(fmt.Errorf("failed to write to info file %v: %v", infoFname, err))
				}

				infoF.Close()

				// create the zip file

				zipFname := targetV + ".zip"
				zipF, err := os.Create(zipFname)
				if err != nil {
					panic(fmt.Errorf("failed to create zip file %v: %v", zipFname, err))
				}

				zipper := zip.NewWriter(zipF)

				walker := fs.Walk(cd)

				for walker.Step() {
					if err := walker.Err(); err != nil {
						fmt.Fprintln(os.Stderr, err)
						continue
					}

					if walker.Stat().IsDir() {
						continue
					}

					f := walker.Path()
					p := path.Join(m.importPath+"@"+v.Version, strings.TrimPrefix(f, cd+string(os.PathSeparator)))

					w, err := zipper.Create(p)
					if err != nil {
						panic(fmt.Errorf("failed to write %v to zip file %v: %v", p, zipFname, err))
					}

					if fc, err := ioutil.ReadFile(f); err != nil {
						panic(fmt.Errorf("failed to read %v: %v", f, err))
					} else {
						if _, err := w.Write(fc); err != nil {
							panic(fmt.Errorf("failed to write contents of %v to %v: %v", f, zipFname, err))
						}
					}
				}

				if err := zipper.Close(); err != nil {
					panic(fmt.Errorf("failed to close zipper: %v", err))
				}

				if err := zipF.Close(); err != nil {
					panic(fmt.Errorf("failed to close zip file %v: %v", zipFname, err))
				}
			}()
		}

		vf.Close()
	}
}

func run(c string, vs ...interface{}) {
	sh := fmt.Sprintf(c, vs...)
	cmd := exec.Command("sh", "-c", sh)

	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("failed to run %v: %v\n%v", sh, err, string(out)))
	}

}
