package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"myitcv.io/internal/golist"
)

const (
	ImmPrefix = "_Imm_"
)

type ignorePaths struct {
	vals []string
}

func (i *ignorePaths) Set(value string) error {
	i.vals = append(i.vals, value)
	return nil
}

func (i *ignorePaths) String() string {
	return fmt.Sprintf("%v", i.vals)
}

var fImm = flag.Bool("imm", false, "filter out _Imm_ types but use their position for the corresponding generated type")
var fIgnorePaths ignorePaths

func init() {
	flag.Var(&fIgnorePaths, "I", "Package path to ignore (can appear multiple times)")
}

type match struct {
	path string
	name string
}

func main() {
	flag.Parse()
	pkgs, err := golist.List(append(fIgnorePaths.vals, "./..."))
	if err != nil {
		panic(err)
	}

	mods, err := golist.ListM(nil)
	if err != nil {
		panic(err)
	}
	if len(mods) == 0 {
		panic(fmt.Errorf("only works in module mode"))
	}

	mainMod := mods[0]

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rel, err := filepath.Rel(mainMod.Dir, wd)
	if err != nil {
		panic(err)
	}
	prefix := path.Join(mainMod.Path, filepath.ToSlash(rel))

	fset := token.NewFileSet()

	matches := make(map[match]string)

	for _, p := range pkgs {
		switch len(p.Match) {
		case 1:
			if p.Match[0] != "./..." {
				continue
			}
		default:
			continue
		}
		pkgs, err := parser.ParseDir(fset, p.Dir, nil, 0)
		if err != nil {
			panic(err)
		}

		base := strings.TrimPrefix(path.Dir(p.ImportPath), prefix)
		if base == "" {
			base = "/"
		}

		for pn, pkg := range pkgs {
			for _, f := range pkg.Files {
				for _, d := range f.Decls {
					switch d := d.(type) {
					case *ast.GenDecl:
						if d.Tok != token.TYPE {
							continue
						}

						for _, s := range d.Specs {
							s := s.(*ast.TypeSpec)

							path := filepath.Join(base, pn)

							key := match{
								path: path,
								name: s.Name.Name,
							}

							matches[key] = fset.Position(s.Pos()).String()
						}
					}
				}
			}
		}
	}

	var out []string

	for k, v := range matches {
		pos := v

		if *fImm {
			if strings.HasPrefix(k.name, ImmPrefix) {
				continue
			}

			imm, ok := matches[match{
				path: k.path,
				name: ImmPrefix + k.name,
			}]

			if ok {
				pos = imm
			}
		}

		out = append(out, fmt.Sprintf("%v: .%v.%v", pos, k.path, k.name))
	}

	sort.Strings(out)

	for _, v := range out {
		fmt.Println(v)
	}
}
