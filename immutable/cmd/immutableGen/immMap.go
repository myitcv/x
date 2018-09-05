package main

import (
	"go/ast"
	"go/types"
	"text/template"

	"myitcv.io/immutable"
	"myitcv.io/immutable/util"
)

type immMap struct {
	commonImm

	// the name of the type to generate; not the pointer version
	name string
	syn  *ast.MapType
	typ  *types.Map
}

func (o *output) genImmMaps(maps []*immMap) {
	for _, m := range maps {
		blanks := struct {
			Name    string
			VarName string
			KeyType string
			ValType string
		}{
			Name:    m.name,
			VarName: genVarName(m.name),
			KeyType: o.exprString(m.syn.Key),
			ValType: o.exprString(m.syn.Value),
		}

		exp := exporter(m.name)

		o.printCommentGroup(m.dec.Doc)
		o.printImmPreamble(m.name, m.syn)

		// start of struct
		o.pfln("type %v struct {", m.name)
		o.pln("")

		o.pfln("theMap map[%v]%v", blanks.KeyType, blanks.ValType)
		o.pln("mutable bool")
		o.pfln("__tmpl *%v%v", immutable.ImmTypeTmplPrefix, m.name)

		// end of struct
		o.pfln("}")

		tmpl := template.New("immmap")
		tmpl.Funcs(exp)
		_, err := tmpl.Parse(immMapTmpl)
		if err != nil {
			fatalf("failed to parse immutable map template: %v", err)
		}

		err = tmpl.Execute(o.output, blanks)
		if err != nil {
			fatalf("failed to execute immutable map template: %v", err)
		}

		o.pt(`
		func (s *{{.}}) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
			if s == nil {
				return true
			}

			if s.Mutable() {
				return false
			}
		`, exp, m.name)

		// we don't vet here; we just do what we are told
		// immutableVet will catch bad stuff later
		keyIsImm := o.isImm(m.typ.Key(), o.exprString(m.syn.Key))
		valIsImm := o.isImm(m.typ.Elem(), o.exprString(m.syn.Value))

		keyIsImmOk := false
		valIsImmOk := false

		switch keyIsImm.(type) {
		case nil, util.ImmTypeBasic:
		default:
			keyIsImmOk = true
		}

		switch valIsImm.(type) {
		case nil, util.ImmTypeBasic:
		default:
			valIsImmOk = true
		}

		if keyIsImmOk || valIsImmOk {
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

			`, exp, m.name)

			switch {
			case keyIsImmOk && valIsImmOk:
				o.pt(`
				for k, v := range s.theMap {
				`, exp, m.name)
			case keyIsImmOk:
				o.pt(`
				for k := range s.theMap {
				`, exp, m.name)
			case valIsImmOk:
				o.pt(`
				for _, v := range s.theMap {
				`, exp, m.name)
			}

			if keyIsImmOk {
				o.pt(`
				if k != nil && !k.IsDeeplyNonMutable(seen) {
					return false
				}
				`, exp, m.name)
			}

			if valIsImmOk {
				o.pt(`
				if v != nil && !v.IsDeeplyNonMutable(seen) {
					return false
				}
				`, exp, m.name)
			}

			o.pt(`
			}
			`, exp, m.name)
		}

		o.pt(`
			return true
		}
		`, exp, m.name)
	}
}
