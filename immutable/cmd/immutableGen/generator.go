// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package main

import (
	"bytes"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"golang.org/x/tools/go/packages"
	"myitcv.io/gogenerate"
	"myitcv.io/immutable/util"
)

const (
	fieldHidingPrefix = "_"
	fieldNamePrefix   = "field"
	fieldAnonPrefix   = "anon"
)

func execute(dir string, envPkg string, licenseHeader string, cmds gogenCmds) {

	absDir, err := filepath.Abs(dir)
	if err != nil {
		fatalf("could not make absolute path from %v: %v", dir, err)
	}

	config := &packages.Config{
		Dir:   absDir,
		Mode:  packages.LoadSyntax,
		Fset:  token.NewFileSet(),
		Tests: false,
	}

	pkgs, err := packages.Load(config, absDir)
	if err != nil {
		fatalf("could not load packages from dir %v: %v", dir, err)
	}

	pkg := pkgs[0]

	if pkg.Name != envPkg {
		pps := make([]string, 0, len(pkgs))
		for _, k := range pkgs {
			pps = append(pps, k.Name)
		}
		fatalf("expected to have parsed %v, instead parsed %v", envPkg, pps)
	}

	var checkFiles []*ast.File
	var fns []string

	for indx, fn := range pkg.GoFiles {
		// skip files that we generated
		if gogenerate.FileGeneratedBy(fn, immutableGenCmd) {
			continue
		}
		f := pkg.Syntax[indx]
		checkFiles = append(checkFiles, f)
		fns = append(fns, fn)
	}

	out := &output{
		dir:       dir,
		info:      pkg.TypesInfo,
		fset:      pkg.Fset,
		pkgName:   pkg.Name,
		pkgPath:   pkg.PkgPath,
		license:   licenseHeader,
		goGenCmds: cmds,
		files:     make(map[*ast.File]*fileTmpls),
		commMaps:  make(map[*ast.File]ast.CommentMap),
		immTypes:  make(map[string]util.ImmType),
		immTmpls:  make(map[string]immTmpl),
		methods:   make(map[string]string),
	}

	for _, f := range checkFiles {
		out.curFile = f
		out.commMaps[f] = ast.NewCommentMap(pkg.Fset, f, f.Comments)
		out.gatherImmTypes()
	}

	// precompute struct methods
	out.calcMethodSets()

	out.genImmTypes()
}

type output struct {
	dir       string
	pkgName   string
	pkgPath   string
	fset      *token.FileSet
	license   string
	goGenCmds gogenCmds

	// type info about the package (and its deps) we are generating against
	info *types.Info

	output *bytes.Buffer

	immTmpls map[string]immTmpl

	// a convenience map of all the imm types we will be generating in this
	// package. The map key here is the pointer type of the generated type.
	immTypes map[string]util.ImmType

	// methods is a map of pointer type name to method name for any methods with
	// pointer receivers we visit
	methods map[string]string

	files map[*ast.File]*fileTmpls

	// a convenience for when we are gathering imm types and generating imm
	// types
	curFile  *ast.File
	commMaps map[*ast.File]ast.CommentMap
}

// fileTmpls are the immutable templates we encounter, along with any imports
// they require
type fileTmpls struct {
	imports map[*ast.ImportSpec]struct{}

	maps    []*immMap
	slices  []*immSlice
	structs []*immStruct
}

type embedded struct {
	typ  types.Type
	es   string
	path []string
}

type field struct {
	path []string
	typ  string
	doc  *ast.CommentGroup
}

func (o *output) isImm(t types.Type, exp string) util.ImmType {
	ct := t
	switch v := ct.(type) {
	case *types.Pointer:
		ct = v.Elem()
	case *types.Named:
		ct = v.Underlying()
	}

	// we might have an invalid type because it refers to a yet-to-be-generated
	// immutable type in this package. If that is the case we fall back to a
	// comparison of the string representation of the type (which will be a
	// pointer).
	if typeIsInvalid(ct) {
		return o.immTypes[exp]
	}

	return util.IsImmType(t)
}

func (o *output) gatherImmTypes() {
	file := o.curFile
	fset := o.fset
	pkgPath := o.pkgPath

	g := &fileTmpls{
		imports: make(map[*ast.ImportSpec]struct{}),
	}

	impf := &importFinder{
		imports: file.Imports,
		matches: g.imports,
	}

	// note the file we are looking in here has _not_ been generated
	// by immutableGen... so we won't walk into methods we generate

	for _, d := range file.Decls {

		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Recv == nil {
				continue
			}

			if len(fd.Recv.List) != 1 {
				continue
			}

			// this is more a sanity check than anything
			if len(fd.Recv.List[0].Names) != 1 {
				continue
			}

			var i *ast.Ident

			if se, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok {
				if id, ok := se.X.(*ast.Ident); ok {
					i = id
				}
			}

			if i == nil {
				continue
			}

			o.methods["*"+i.Name] = fd.Name.Name

			// we can't be a type decl
			continue
		}

		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, s := range gd.Specs {
			ts := s.(*ast.TypeSpec)

			name, ok := util.IsImmTmpl(ts)
			if !ok {
				continue
			}

			typ := o.info.Defs[ts.Name].Type().(*types.Named)

			infof("found immutable declaration at %v: %v", fset.Position(gd.Pos()), typ)

			comm := commonImm{
				fset: fset,
				file: file,
				pkg:  pkgPath,
				dec:  gd,
			}

			switch u := typ.Underlying().(type) {
			case *types.Map:
				m := &immMap{
					commonImm: comm,
					name:      name,
					typ:       u,
					syn:       ts.Type.(*ast.MapType),
				}
				g.maps = append(g.maps, m)
				o.immTypes["*"+name] = util.ImmTypeMap{}
				o.immTmpls["*"+name] = m

				ast.Walk(impf, ts.Type)

			case *types.Slice:
				// TODO support for arrays
				s := &immSlice{
					commonImm: comm,
					name:      name,
					typ:       u,
					syn:       ts.Type.(*ast.ArrayType),
				}

				g.slices = append(g.slices, s)
				o.immTypes["*"+name] = util.ImmTypeSlice{}
				o.immTmpls["*"+name] = s

				ast.Walk(impf, ts.Type)

			case *types.Struct:
				astst := ts.Type.(*ast.StructType)

				var fields []astField

				for _, f := range astst.Fields.List {
					if len(f.Names) == 0 {
						fields = append(fields, astField{
							anon:  true,
							name:  fieldTypeToIdent(f.Type).Name,
							field: f,
						})
					} else {
						for _, n := range f.Names {
							fields = append(fields, astField{
								name:  n.Name,
								field: f,
							})
						}
					}
				}

				s := &immStruct{
					commonImm: comm,
					name:      name,
					typ:       u,
					syn:       ts.Type.(*ast.StructType),
					special:   isSpecialStruct(name, u),
					fields:    fields,
				}

				g.structs = append(g.structs, s)
				o.immTypes["*"+name] = util.ImmTypeStruct{}
				o.immTmpls["*"+name] = s

				ast.Walk(impf, ts.Type)
			}

		}
	}

	o.files[o.curFile] = g
}
