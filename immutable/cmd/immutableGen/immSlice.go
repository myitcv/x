package main

import (
	"go/ast"
	"go/types"
	"text/template"

	"myitcv.io/immutable"
	"myitcv.io/immutable/util"
)

type immSlice struct {
	commonImm

	// the name of the type to generate; not the pointer version
	name string
	syn  *ast.ArrayType
	typ  *types.Slice
}

func (o *output) genImmSlices(slices []*immSlice) {

	for _, s := range slices {
		blanks := struct {
			Name string
			Type string
		}{
			Name: s.name,
			Type: o.exprString(s.syn.Elt),
		}

		exp := exporter(s.name)

		o.printCommentGroup(s.dec.Doc)
		o.printImmPreamble(s.name, s.syn)

		// start of struct
		o.pfln("type %v struct {", s.name)
		o.pln("")

		o.pfln("theSlice []%v", blanks.Type)
		o.pln("mutable bool")
		o.pfln("__tmpl *%v%v", immutable.ImmTypeTmplPrefix, s.name)

		// end of struct
		o.pfln("}")

		tmpl := template.New("immslice")
		tmpl.Funcs(exp)
		_, err := tmpl.Parse(immSliceTmpl)
		if err != nil {
			fatalf("failed to parse immutable slice template: %v", err)
		}

		err = tmpl.Execute(o.output, blanks)
		if err != nil {
			fatalf("failed to execute immutable slice template: %v", err)
		}

		o.pt(`
		func (s *{{.}}) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
			if s == nil {
				return true
			}

			if s.Mutable() {
				return false
			}
		`, exp, s.name)

		valIsImm := o.isImm(s.typ.Elem(), o.exprString(s.syn.Elt))

		valIsImmOk := false

		switch valIsImm.(type) {
		case nil, util.ImmTypeBasic:
		default:
			valIsImmOk = true
		}

		if valIsImmOk {
			o.pt(`
			if s.Len() == 0 {
				return true
			}

			if seen == nil {
				return s.IsDeeplyNonMutable(make(map[interface{}]bool))
			}

			if seen[s] {
				return true
			}

			seen[s] = true

			for _, v := range s.theSlice {
			`, exp, s.name)

			o.pt(`
				if v != nil && !v.IsDeeplyNonMutable(seen) {
					return false
				}
			`, exp, s.name)

			o.pt(`
			}
			`, exp, s.name)
		}

		o.pt(`
			return true
		}
		`, exp, s.name)
	}
}
