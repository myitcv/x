// modpub is a tool to help create a directory of vgo modules from a git respository.
//
// For more information see https://github.com/myitcv/x/blob/master/cmd/modpub/README.md
package main

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

const (
	panicOnError = true
)

var (
	fTarget  = flag.String("target", "", "target directory for publishing")
	fVerbose = flag.Bool("v", false, "give verbose output")

	usage string
)

func main() {
	// Notes
	//
	// 1. We trust that any given go.mod file is accurate i.e. that submodules are
	// correctly nested and the respective go.mod files reflect that
	// 2. We assume that go.mod files have the module line _as the first line_
	setupAndParseFlags()

	cwd, err := os.Getwd()
	if err != nil {
		fatalf("failed to get cwd: %v", err)
	}

	if *fTarget == "" {
		fatalf("-target flag is required\n\n%v", usage)
	}

	target, err := filepath.Abs(*fTarget)
	if err != nil {
		fatalf("failed to make target absolute: %v", err)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		fatalf("target %v must exist")
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

			fatalf("error trying to find .git directory: %v", err)
		}

		if !fi.IsDir() {
			break
		}

		found = true

		break
	}

	if !found {
		fatalf("failed to find .git repo root, walking upwards from %v", target)
	}

	walker := fs.Walk(".")

	relRoot, err := filepath.Rel(repoRoot, cwd)
	if err != nil {
		fatalf("unable to resolve %v relative to %v: %v", cwd, repoRoot, err)
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

		if filepath.Base(p) == "go.mod" {

			m := &module{path: filepath.Join(relRoot, filepath.Dir(p))}

			// TODO using the go.mod parser when it exists
			// for now assume the file is well-formed
			gm, err := os.Open(p)
			if err != nil {
				fatalf("failed to open %v: %v", p, err)
			}

			scanner := bufio.NewScanner(gm)

			for scanner.Scan() {
				// we expect module to be the first line
				line := scanner.Text()

				parts := strings.Split(line, " ")

				if len(parts) != 2 {
					fatalf("go.mod file %v had unexpected first line: %v", p, line)
				}

				m.importPath = strings.Trim(parts[1], "\"")

				// we don't care about the rest
				break
			}

			if err := scanner.Err(); err != nil {
				fatalf("error reading version details from %v: %v", p, err)
			}

			// now get the versions
			// note this is run in the same directory as the go.mod file with a "." argument
			cmd := exec.Command("git", "log", "origin/master", "--first-parent", "--reverse", "--date=format-local:%Y-%m-%dT%H:%M:%SZ", "--pretty=format:%H %cd", ".")
			cmd.Env = append(os.Environ(), "TZ=UTC")
			cmd.Dir = filepath.Join(cwd, m.path)

			out, err := cmd.CombinedOutput()
			if err != nil {
				fatalf("cmd [%v] in directory %v failed: %v\n%v", strings.Join(cmd.Args, " "), cmd.Dir, err, string(out))
			}

			r := bytes.NewReader(out)

			s := bufio.NewScanner(r)

			for s.Scan() {
				// each line will be a commit ref and time
				line := s.Text()

				parts := strings.Split(line, " ")

				if len(parts) != 2 {
					fatalf("unexpected result line from git log (cmd [%v] in directory %v): %v", strings.Join(cmd.Args, " "), cmd.Dir, line)
				}

				t, err := time.Parse(time.RFC3339, parts[1])
				if err != nil {
					fatalf("failed to parse time %v (cmd [%v] in directory %v): %v", strings.Join(cmd.Args, " "), cmd.Dir, parts[1], err)
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
				fatalf("error reading version details (cmd [%v] in directory %v): %v", strings.Join(cmd.Args, " "), cmd.Dir, err)
			}

			modules = append(modules, m)
		}
	}

	if len(modules) == 0 {
		infof("found no modules\n")
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

		return strings.Count(lhs, string(os.PathSeparator)) < strings.Count(rhs, string(os.PathSeparator))
	})

	// as the controlling go routine ensure all the target directories exist
	// to avoid racing on those - versions will be mutually exclusive
	for i := range modules {
		m := modules[i]

		vd := filepath.Join(target, m.importPath, "@v")
		if err := os.MkdirAll(vd, 0755); err != nil {
			fatalf("failed to create directory %v: %v", vd, err)
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

	vinfof("Found modules:\n")

	for _, m := range modules {
		vinfof("ImportPath: %v, Path: %v\n", m.importPath, m.path)

		for _, sm := range m.submodules {
			vinfof("  submodule %v\n", sm)
		}

		if len(m.versions) == 0 {
			vinfof("  ** no versions\n")
			continue
		}

		// write the version file
		targetM := filepath.Join(target, m.importPath, "@v")
		vfName := filepath.Join(targetM, "list")
		vf, err := os.Create(vfName)
		if err != nil {
			fatalf("failed to create version file %v: %v", vfName, err)
		}

		for _, v := range m.versions {
			// write the version
			fmt.Fprintf(vf, "%v %v\n", v.Version, v.Time.Format(time.RFC3339))

			targetV := filepath.Join(targetM, v.Version)

			func() {
				td, err := ioutil.TempDir("", "modpubX-")
				if err != nil {
					fatalf("failed to create a working temp dir: %v", err)
				}

				defer os.RemoveAll(td)

				vinfof("  version %v %v\n", v.Version, v.Time)

				// git archive into the td; we use the cwd
				{
					cmd := exec.Command("git", "archive", v.commitish)
					stdout, err := cmd.StdoutPipe()
					if err != nil {
						fatalf("failed to get stdout pipe for git archive: %v", err)
					}

					tr := tar.NewReader(stdout)

					if err := cmd.Start(); err != nil {
						fatalf("failed to start git archive: %v", err)
					}

					for {
						hdr, err := tr.Next()
						if err == io.EOF {
							break
						}
						if err != nil {
							fatalf("failed to read output from git archive: %v", err)
						}

						fn := filepath.Join(td, hdr.Name)

						switch {
						case hdr.Typeflag == tar.TypeDir:
							if err := os.MkdirAll(fn, 0700); err != nil {
								fatalf("failed to mkdir %v: %v", fn, err)
							}
						case hdr.Typeflag == tar.TypeReg:
							f, err := os.Create(fn)
							if err != nil {
								fatalf("failed to create %v: %v", fn, err)
							}

							if _, err := io.Copy(f, tr); err != nil {
								fatalf("failed to write git archive file %v to %v: %v", hdr.Name, fn, err)
							}

							if err := f.Close(); err != nil {
								fatalf("failed to close %v: %v", fn, err)
							}
						case hdr.Typeflag == tar.TypeXGlobalHeader:
							// noop
						default:
							fatalf("skipping %v; type unknown: %v\n", hdr.Name, hdr.Typeflag)
						}

					}
				}

				// remove all submodules

				cd := filepath.Join(td, relRoot, m.path)

				for _, sm := range m.submodules {
					smpath := filepath.Join(cd, sm)
					if err := os.RemoveAll(smpath); err != nil {
						fatalf("failed to remove submodule %v in path %v: %v", sm, cd, err)
					}
				}

				// copy the .mod file
				{
					im := filepath.Join(cd, "go.mod")
					in, err := os.Open(im)
					if err != nil {
						fatalf("failed to open %v: %v", im, err)
					}

					om := targetV + ".mod"
					ot, err := os.Create(om)
					if err != nil {
						fatalf("failed to create %v: %v", om, err)
					}

					if _, err := io.Copy(ot, in); err != nil {
						fatalf("failed to copy %v to %v: %v", im, om, err)
					}
				}

				// write the .info file
				infoFname := targetV + ".info"
				infoF, err := os.Create(infoFname)
				if err != nil {
					fatalf("failed to create info file %v; %v", infoFname, err)
				}

				enc := json.NewEncoder(infoF)

				if err := enc.Encode(v); err != nil {
					fatalf("failed to write to info file %v: %v", infoFname, err)
				}

				infoF.Close()

				// create the zip file

				zipFname := targetV + ".zip"
				zipF, err := os.Create(zipFname)
				if err != nil {
					fatalf("failed to create zip file %v: %v", zipFname, err)
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
						fatalf("failed to write %v to zip file %v: %v", p, zipFname, err)
					}

					if fc, err := ioutil.ReadFile(f); err != nil {
						fatalf("failed to read %v: %v", f, err)
					} else {
						if _, err := w.Write(fc); err != nil {
							fatalf("failed to write contents of %v to %v: %v", f, zipFname, err)
						}
					}
				}

				if err := zipper.Close(); err != nil {
					fatalf("failed to close zipper for %v: %v", zipFname, err)
				}

				if err := zipF.Close(); err != nil {
					fatalf("failed to close zip file %v: %v", zipFname, err)
				}
			}()
		}

		vf.Close()
	}
}

func fatalf(format string, vs ...interface{}) {
	s := fmt.Sprintf(format, vs...)

	if panicOnError {
		panic(fmt.Errorf(s))
	}

	fmt.Fprintf(os.Stderr, s)
	os.Exit(1)
}

func infof(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func vinfof(format string, args ...interface{}) {
	if *fVerbose {
		infof(format, args...)
	}
}
