package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
	"myitcv.io/gogenerate"
	"myitcv.io/immutable"
	"myitcv.io/immutable/util"
)

const (
	skipFileComment = "//" + immutable.CmdImmutableVet + ":skipFile"
)

var fset = token.NewFileSet()

type immutableVetter struct {
	pkgs []*packages.Package

	wd string

	skipFiles map[string]bool

	info *types.Info

	// immTmpls is the set of immutable template types in the package
	// being analysed
	immTmpls map[types.Type]bool

	// helper field used to hold Range() method calls on immutable types
	rngs map[*ast.Ident]bool

	// valid composite literals
	vcls map[*ast.CompositeLit]bool

	errlist []immErr
}

var typesCache = map[string]bool{
	"time.Time": true,
}

type immErr struct {
	pos token.Position
	msg string
}

type errors []immErr

var immIntf *types.Interface

func main() {
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		fatalf("could not get the working directory")
	}

	emsgs := vet(wd, flag.Args())

	for _, msg := range emsgs {
		fmt.Fprintf(os.Stderr, "%v\n", msg)
	}

	if len(emsgs) > 0 {
		os.Exit(1)
	}
}

func vet(wd string, args []string) []immErr {
	var emsgs []immErr

	// vetting phase: vet all packages packages passed in through the command line
	iv := newImmutableVetter(wd, args)

	emsgs = append(emsgs, iv.vetPackages()...)

	for i := range emsgs {
		rel, err := filepath.Rel(wd, emsgs[i].pos.Filename)
		if err != nil {
			fatalf("relative path error, %v", err)
		}

		emsgs[i].pos.Filename = rel
	}

	sort.Sort(errors(emsgs))

	return emsgs
}

func (iv *immutableVetter) ensurePointerTyp(n ast.Node, typ ast.Expr) {
	if ts, ok := n.(*ast.TypeSpec); ok {
		if ts.Assign.IsValid() {
			// we are an alias; this is fine in all cases
			return
		}
	}
	t := iv.info.Types[typ].Type
	if t == nil {
		return
	}
	p := types.NewPointer(t)
	switch util.IsImmType(p).(type) {
	case util.ImmTypeMap, util.ImmTypeSlice, util.ImmTypeStruct:
		iv.errorf(n.Pos(), "type should be %v", p)
	}
}

func (iv *immutableVetter) Visit(node ast.Node) ast.Visitor {

	switch node := node.(type) {
	case *ast.File:
		for _, cg := range node.Comments {
			for _, c := range cg.List {
				if c.Text == skipFileComment {
					iv.skipFiles[fset.Position(node.Pos()).Filename] = true
					return nil
				}
			}
		}
	case *ast.ValueSpec:
		iv.ensurePointerTyp(node, node.Type)
	case *ast.ArrayType:
		iv.ensurePointerTyp(node, node.Elt)
	case *ast.MapType:
		iv.ensurePointerTyp(node, node.Key)
		iv.ensurePointerTyp(node, node.Value)
	case *ast.Field:
		iv.ensurePointerTyp(node, node.Type)
	case *ast.UnaryExpr:
		if node.Op != token.AND {
			break
		}

		cl, ok := node.X.(*ast.CompositeLit)
		if !ok {
			break
		}

		t := iv.info.Types[cl.Type].Type
		p := types.NewPointer(t)
		switch util.IsImmType(p).(type) {
		case util.ImmTypeMap, util.ImmTypeSlice, util.ImmTypeStruct:
			iv.errorf(node.Pos(), "construct using new() or generated constructors")
			iv.vcls[cl] = true
		}
	case *ast.CompositeLit:
		if ok := iv.vcls[node]; ok {
			break
		}

		iv.ensurePointerTyp(node, node.Type)
	case *ast.TypeSpec:
		iv.ensurePointerTyp(node, node.Type)
	case *ast.SelectorExpr:
		sel, ok := iv.info.Selections[node]
		if !ok {
			// this is fine... !ok implies a selector expression
			// that is a qualified identifier as opposed to a method
			// field selector
			break
		}

		if !isImmListOrMap(sel.Recv()) {
			break
		}

		switch node.Sel.Name {
		case "Range":
			if _, ok := iv.rngs[node.Sel]; !ok {
				iv.rngs[node.Sel] = false
			}
		}
	case *ast.RangeStmt:
		v := node.X
		ce, ok := v.(*ast.CallExpr)
		if !ok {
			break
		}

		e := ce.Fun
		se, ok := e.(*ast.SelectorExpr)
		if !ok {
			break
		}

		sel, ok := iv.info.Selections[se]
		if !ok {
			// then it must be a qualified identifier
			break
		}

		if !isImmListOrMap(sel.Recv()) {
			break
		}

		if sel.Kind() != types.MethodVal {
			break
		}

		ri := se.Sel
		if ri.Name != "Range" {
			break
		}
		iv.rngs[ri] = true
	case *ast.CallExpr:
		switch fun := node.Fun.(type) {
		case *ast.Ident:
			if fun.Name != "append" {
				break
			}

			if len(node.Args) != 2 {
				break
			}

			e := node.Args[1]
			ce, ok := e.(*ast.CallExpr)
			if !ok {
				break
			}

			se, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}

			sel, ok := iv.info.Selections[se]
			if !ok {
				break
			}

			if !isImmListOrMap(sel.Recv()) {
				break
			}

			ri := se.Sel
			if ri.Name != "Range" {
				break
			}

			if node.Ellipsis == node.Args[1].End() {
				iv.rngs[ri] = true
			}
		case *ast.SelectorExpr:
			sel, ok := iv.info.Selections[fun]
			if !ok {
				// this is fine... !ok implies a selector expression
				// that is a qualified identifier as opposed to a method
				// field selector
				break
			}

			if !isImmListOrMap(sel.Recv()) {
				break
			}

			if sel.Kind() != types.MethodVal {
				break
			}

			ri := fun.Sel
			if ri.Name != "Append" {
				break
			}

			if len(node.Args) != 1 {
				break
			}

			if node.Ellipsis == token.NoPos {
				break
			}

			ace, ok := node.Args[0].(*ast.CallExpr)
			if !ok {
				break
			}

			{
				se, ok := ace.Fun.(*ast.SelectorExpr)
				if !ok {
					break
				}

				sel, ok := iv.info.Selections[se]
				if !ok {
					// this is fine... !ok implies a selector expression
					// that is a qualified identifier as opposed to a method
					// field selector
					break
				}

				if !isImmListOrMap(sel.Recv()) {
					break
				}

				if sel.Kind() != types.MethodVal {
					break
				}

				ri := se.Sel
				if ri.Name == "Range" {
					iv.rngs[ri] = true
				}
			}
		}
	}
	return iv
}

func isImmListOrMap(t types.Type) bool {
	switch util.IsImmType(t).(type) {
	case util.ImmTypeMap, util.ImmTypeSlice:
		return true
	}

	return false
}

func newImmutableVetter(wd string, args []string) *immutableVetter {
	ip := "myitcv.io/immutable"
	allPkgs := append(args, ip)

	cfg := &packages.Config{
		Mode:  packages.LoadSyntax,
		Fset:  fset,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, allPkgs...)
	if err != nil {
		fatalf("could not load pacakages %v: %v", pkgs, err)
	}

	var immPkg *packages.Package

	vetPkgs := []*packages.Package{}
	for _, p := range pkgs {
		if p.PkgPath == ip {
			immPkg = p
			continue
		}
		vetPkgs = append(vetPkgs, p)
	}

	if immPkg == nil {
		fatalf("failed to load immutable package")
	}

	o := immPkg.Types.Scope().Lookup("Immutable")

	if o == nil {
		fatalf("failed to find anything called Immutable in pkg scope of %v", ip)
	}

	tn, ok := o.(*types.TypeName)
	if !ok {
		fatalf("Immutable is not a *types.TypeName: %T", o)
	}

	nmd, ok := tn.Type().(*types.Named)
	if !ok {
		fatalf("Immutable type is not a *types.Named: %T", tn.Type())
	}

	intf, ok := nmd.Underlying().(*types.Interface)
	if !ok {
		fatalf("Underlying type is not a *types.Interface: %T", nmd.Underlying())
	}

	immIntf = intf

	return &immutableVetter{
		pkgs:      vetPkgs,
		vcls:      make(map[*ast.CompositeLit]bool),
		wd:        wd,
		skipFiles: make(map[string]bool),
	}
}

func (iv *immutableVetter) isImmTmpl(t types.Type) bool {
	switch t := t.(type) {
	case *types.Pointer:
		return iv.isImmTmpl(t.Elem())
	}

	return iv.immTmpls[t]
}

func (iv *immutableVetter) vetPackages() []immErr {
	for _, pkg := range iv.pkgs {
		info := pkg.TypesInfo

		iv.rngs = make(map[*ast.Ident]bool)
		iv.immTmpls = make(map[types.Type]bool)
		iv.info = info

		for _, f := range pkg.Syntax {
			for _, d := range f.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok {
					continue
				}

				if gd.Tok != token.TYPE {
					continue
				}

				for _, s := range gd.Specs {
					ts := s.(*ast.TypeSpec)

					_, ok := util.IsImmTmpl(ts)
					if !ok {
						continue
					}

					o := info.ObjectOf(ts.Name)
					iv.immTmpls[o.Type()] = true

					st, ok := o.Type().(*types.Named).Underlying().(*types.Struct)
					if !ok {
						continue
					}

					for i := 0; i < st.NumFields(); i++ {
						f := st.Field(i)
						if !isImmType(f.Type()) {
							msg := fmt.Sprintf("immutable struct field must be immutable type; %v is not", f.Type())
							iv.errorf(f.Pos(), msg)
						}
					}
				}
			}
		}

		for _, fnode := range pkg.Syntax {
			ast.Walk(iv, fnode)
		}

		for exp, t := range info.Types {
			out := bytes.NewBuffer(nil)
			printer.Fprint(out, fset, exp)

			switch {
			case t.IsType():
				typ := t.Type

				if !iv.isImmTmpl(typ) {
					continue
				}

				fn := fset.Position(exp.Pos()).Filename

				if !gogenerate.FileGeneratedBy(fn, immutable.CmdImmutableGen) {
					iv.errorf(exp.Pos(), "template type %v should never get used", typ)
				}

			case t.IsValue():
				p := types.NewPointer(t.Type)
				switch util.IsImmType(p).(type) {
				case util.ImmTypeMap:
				case util.ImmTypeSlice:
				case util.ImmTypeStruct:
				default:
					continue
				}

				fn := fset.Position(exp.Pos()).Filename

				if !iv.skipFiles[fn] {
					iv.errorf(exp.Pos(), "non-pointer value of immutable type %v found", p)
				}
			}

		}

		// find selector exprs which access properties of Immutable types
		for exp, sel := range info.Selections {
			isField := sel.Kind() == types.FieldVal
			if !isField {
				continue
			}

			if util.IsImmType(sel.Recv()) == nil {
				continue
			}

			if iv.skipFiles[fset.Position(exp.X.Pos()).Filename] {
				continue
			}

			oname := sel.Obj().Name()
			iv.errorf(exp.X.Pos(), "should not be using %v of %v immutable type", oname, sel.Recv())
		}

		for k, v := range iv.rngs {
			if v == false {
				msg := fmt.Sprintf("Range() of immutable type must appear in a range statement or used with an ellipsis as the second argument to append")
				iv.errorf(k.NamePos, msg)
			}
		}
	}

	return iv.errlist
}

func isImmType(t types.Type) bool {
	if v, ok := typesCache[t.String()]; ok {
		return v
	}

	switch t := t.(type) {
	case *types.Named:

		typesCache[t.String()] = true

		v := isImmType(t.Underlying())
		typesCache[t.String()] = v

		return v
	case *types.Basic:
		return true
	case *types.Map, *types.Slice:
		return false
	case *types.Pointer:
		return util.IsImmType(t) != nil
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			f := t.Field(i)
			if !isImmType(f.Type()) {
				return false
			}
		}

		return true
	case *types.Interface:
		return types.Implements(t, immIntf)
	case *types.Signature:
		return false
	default:
		fatalf("unable to handle type %T %v", t, t)
	}

	return false
}

func (iv *immutableVetter) errorf(pos token.Pos, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	iv.errlist = append(iv.errlist, immErr{
		pos: fset.Position(pos),
		msg: msg,
	})
}

func fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}

func (r immErr) String() string {
	return fmt.Sprintf("%v:%v:%v: %v", r.pos.Filename, r.pos.Line, r.pos.Column, r.msg)
}

func (e errors) Len() int {
	return len(e)
}

func (e errors) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e errors) Less(i, j int) bool {
	l, r := e[i].pos, e[j].pos

	if v := strings.Compare(l.Filename, r.Filename); v != 0 {
		return v < 0
	}

	if l.Line != r.Line {
		return l.Line < r.Line
	}

	if l.Column != r.Column {
		return l.Column < r.Column
	}

	return e[i].msg < e[j].msg
}
