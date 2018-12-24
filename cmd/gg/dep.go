package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"myitcv.io/gogenerate"
)

type dep interface {
	fmt.Stringer
	Deps() depsMap
	Ready() bool
	Done() bool
	Undo()
	HashString() string
}

type generator interface {
	dep
	DirectiveName() string
}

type depsMap struct {
	deps      map[dep]bool
	rdeps     map[dep]bool
	dirtydeps map[dep]bool
}

func sortDeps(deps []dep) {
	sort.Slice(deps, func(i, j int) bool {
		lhs, rhs := deps[i], deps[j]
		switch lhs := lhs.(type) {
		case *pkg:
			switch rhs := rhs.(type) {
			case *pkg:
				return lhs.ImportPath < rhs.ImportPath
			case *gobinModDep, *gobinGlobalDep, *commandDep:
				return true
			default:
				panic(fmt.Errorf("could not compare %T with %T", lhs, rhs))
			}
		case *gobinModDep:
			switch rhs := rhs.(type) {
			case *pkg:
				return false
			case *gobinModDep:
				return lhs.importPath < rhs.importPath
			case *gobinGlobalDep, *commandDep:
				return true
			default:
				panic(fmt.Errorf("could not compare %T with %T", lhs, rhs))
			}
		case *gobinGlobalDep:
			switch rhs := rhs.(type) {
			case *pkg, *gobinModDep:
				return false
			case *gobinGlobalDep:
				return lhs.targetPath < rhs.targetPath
			case *commandDep:
				return true
			default:
				panic(fmt.Errorf("could not compare %T with %T", lhs, rhs))
			}
		case *commandDep:
			switch rhs := rhs.(type) {
			case *pkg, *gobinModDep, *gobinGlobalDep:
				return false
			case *commandDep:
				return lhs.name < rhs.name
			default:
				panic(fmt.Errorf("could not compare %T with %T", lhs, rhs))
			}
		}
		panic("should not get here")
	})
}

type missingDeps map[string]map[*pkg]bool

func (m missingDeps) add(ip string, p *pkg) {
	deps, ok := m[ip]
	if !ok {
		deps = make(map[*pkg]bool)
		m[ip] = deps
	}
	deps[p] = true
}

func parseOutDirs(dir string, args []string) ([]string, error) {
	outDirs := make(map[string]bool)
	for i := 0; i < len(args); i++ {
		v := args[i]
		if v == "--" {
			break
		}
		if !strings.HasPrefix(v, "-"+gogenerate.FlagOutDirPrefix) {
			continue
		}
		v = strings.TrimPrefix(v, "-"+gogenerate.FlagOutDirPrefix)
		j := strings.Index(v, "=")
		var d string
		if j == -1 {
			if i+1 == len(args) || args[i+1] == "--" {
				return nil, fmt.Errorf("invalid output dir flag amongst: %v", args)
			}
			d = args[i+1]
		} else {
			d = v[j+1:]
		}
		if !filepath.IsAbs(d) {
			d = filepath.Join(dir, d)
		}
		ed, err := filepath.EvalSymlinks(d)
		if err != nil {
			return nil, fmt.Errorf("failed to eval symlinks for dir %v: %v", d, err)
		}
		outDirs[ed] = true
	}
	var dirs []string
	for d := range outDirs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs, nil
}
