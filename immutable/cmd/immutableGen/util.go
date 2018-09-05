package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/types"
	"strings"
	"text/template"
)

func isSpecialStruct(name string, st *types.Struct) bool {
	// work out whether this is a special struct with a Key field
	// pattern is:
	//
	// 1. struct field has a field called Key of type {{.StructName}}Key (non pointer)
	//
	// later checks will include:
	//
	// 2. said type has two fields, Uuid and Version, of type {{.StructName}}Uuid and uint64 respectively
	// 3. the underlying type of {{.StructName}}Uuid is uint64 (we might be able to relax these two
	// two underlying type restrictions)

	if st.NumFields() == 0 {
		return false
	}

	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)

		if f.Name() != "Key" {
			continue
		}

		kst, ok := f.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}

		if kst.NumFields() != 2 {
			continue
		}

		uuid := kst.Field(0)
		if uuid.Name() != "Uuid" {
			continue
		}

		ver := kst.Field(1)
		if ver.Name() != "Version" {
			continue
		}

		// we found it
		return true
	}

	return false
}

func typeIsInvalid(t types.Type) bool {
	switch t := t.(type) {
	case *types.Basic:
		return t.Kind() == types.Invalid
	case *types.Pointer:
		return typeIsInvalid(t.Elem())
	case nil:
		return true
	}

	return false
}

func fieldTypeToIdent(e ast.Expr) *ast.Ident {
	switch e := e.(type) {
	case *ast.Ident:
		return e
	case *ast.StarExpr:
		return fieldTypeToIdent(e.X)
	case *ast.SelectorExpr:
		return e.Sel
	default:
		panic(fmt.Errorf("don't know how to handle %T", e))
	}
}

func (o *output) exprString(e ast.Expr) string {
	var buf bytes.Buffer

	err := printer.Fprint(&buf, o.fset, e)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func (o *output) printCommentGroup(d *ast.CommentGroup) {
	if d != nil {
		for _, c := range d.List {
			o.pfln("%v", c.Text)
		}
	}
}

func (o *output) pln(i ...interface{}) {
	fmt.Fprintln(o.output, i...)
}

func (o *output) pf(format string, i ...interface{}) {
	fmt.Fprintf(o.output, format, i...)
}

func (o *output) pfln(format string, i ...interface{}) {
	o.pf(format+"\n", i...)
}

func (o *output) pt(tmpl string, fm template.FuncMap, val interface{}) {

	// on the basis most templates are for convenience define inline
	// as raw string literals which start the ` on one line but then start
	// the template on the next (for readability) we strip the first leading
	// \n if one exists
	tmpl = strings.TrimPrefix(tmpl, "\n")

	t := template.New("tmp")
	t.Funcs(fm)

	_, err := t.Parse(tmpl)
	if err != nil {
		panic(err)
	}

	err = t.Execute(o.output, val)
	if err != nil {
		panic(err)
	}
}
