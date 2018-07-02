package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kisielk/gotool"
)

const (
	ImmPrefix = "_Imm_"
)

type ignorePaths []string

func (i *ignorePaths) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *ignorePaths) String() string {
	return fmt.Sprint(*i)
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
	pkgs := gotool.ImportPaths([]string{"./..."})

	fset := token.NewFileSet()

	matches := make(map[match]string)

Parse:
	for _, dir := range pkgs {
		for _, p := range fIgnorePaths {
			if dir == p {
				continue Parse
			}
		}
		pkgs, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			panic(err)
		}

		base := filepath.Dir(dir)

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

		out = append(out, fmt.Sprintf("%v: ./%v.%v", pos, k.path, k.name))
	}

	sort.Strings(out)

	for _, v := range out {
		fmt.Println(v)
	}
}
