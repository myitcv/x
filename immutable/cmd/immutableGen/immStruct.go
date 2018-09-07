package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"myitcv.io/immutable"
	"myitcv.io/immutable/util"
)

type commonImm struct {
	fset *token.FileSet

	// the full package import path (not just the name)
	// declaring the type
	pkg string

	// the declaring file
	file *ast.File

	// the template declaration
	dec *ast.GenDecl
}

func (c *commonImm) isImmTmpl() {}

type immTmpl interface {
	isImmTmpl()
}

type immStruct struct {
	commonImm

	// the name of the type to generate; not the pointer version
	name string
	syn  *ast.StructType
	typ  *types.Struct

	special bool

	fields  []astField
	methods map[string]*field
}

type astField struct {
	anon  bool
	name  string
	field *ast.Field
}

func (o *output) genImmStructs(structs []*immStruct) {
	type genField struct {
		// the actual field name used in the generated struct
		Field string

		// The proper name of the field to be used on the method
		Name  string
		Type  string
		f     *ast.Field
		IsImm util.ImmType
	}

	for _, s := range structs {

		o.printCommentGroup(s.dec.Doc)
		o.printImmPreamble(s.name, s.syn)

		// start of struct
		o.pfln("type %v struct {", s.name)

		o.printLeadSpecCommsFor(s.syn)

		o.pln("")

		var fields []genField

		for _, f := range s.fields {

			name := fieldNamePrefix + fieldHidingPrefix + f.name

			if f.anon {
				name = fieldAnonPrefix + name
			}

			tag := ""
			if f.field.Tag != nil {
				tag = f.field.Tag.Value
			}
			typ := o.exprString(f.field.Type)

			isImm := o.isImm(o.info.TypeOf(f.field.Type), typ)

			fields = append(fields, genField{
				Field: name,
				Name:  f.name,
				Type:  typ,
				f:     f.field,
				IsImm: isImm,
			})

			o.pfln("%v %v %v", name, typ, tag)
		}

		o.pln("")
		o.pln("mutable bool")
		o.pfln("__tmpl *%v%v", immutable.ImmTypeTmplPrefix, s.name)

		// end of struct
		o.pfln("}")

		o.pln()

		o.pfln("var _ immutable.Immutable = new(%v)", s.name)
		o.pfln("var _ = new(%v).__tmpl", s.name)
		o.pln()

		exp := exporter(s.name)

		o.pt(`
		func (s *{{.}}) AsMutable() *{{.}} {
			if s.Mutable() {
				return s
			}

			res := *s
		`, exp, s.name)
		if s.special {
			o.pt(`
			res.field_Key.Version++
			`, exp, nil)
		}
		o.pt(`
			res.mutable = true
			return &res
		}

		func (s *{{.}}) AsImmutable(v *{{.}}) *{{.}} {
			if s == nil {
				return nil
			}

			if s == v {
				return s
			}

			s.mutable = false
			return s
		}

		func (s *{{.}}) Mutable() bool {
			return s.mutable
		}

		func (s *{{.}}) WithMutable(f func(si *{{.}})) *{{.}} {
			res := s.AsMutable()
			f(res)
			res = res.AsImmutable(s)

			return res
		}

		func (s *{{.}}) WithImmutable(f func(si *{{.}})) *{{.}} {
			prev := s.mutable
			s.mutable = false
			f(s)
			s.mutable = prev

			return s
		}

		func (s *{{.}}) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
			if s == nil {
				return true
			}

			if s.Mutable() {
				return false
			}

			if seen == nil {
				return s.IsDeeplyNonMutable(make(map[interface{}]bool))
			}

			if seen[s] {
				return true
			}

			seen[s] = true
		`, exp, s.name)

		for _, f := range fields {
			if f.IsImm == nil {
				continue
			}
			switch f.IsImm.(type) {
			case util.ImmTypeSlice, util.ImmTypeStruct, util.ImmTypeMap, util.ImmTypeImplsIntf, util.ImmTypeSimple:

				tmpl := struct {
					FieldName string
				}{
					FieldName: f.Field,
				}

				o.pt(`
				{
					v := s.{{.FieldName}}

					if v != nil && !v.IsDeeplyNonMutable(seen) {
						return false
					}
				}
				`, exp, tmpl)
			case util.ImmTypeBasic:
			}
		}

		o.pt(`
			return true
		}
		`, exp, s.name)

		var mns []string
		for n := range s.methods {
			mns = append(mns, n)
		}

		sort.Strings(mns)

		for _, n := range mns {
			f := s.methods[n]

			tmpl := struct {
				TypeName string
				Path     string
				Type     string
				Name     string
			}{
				TypeName: s.name,
				Path:     strings.Join(f.path, "."),
				Type:     f.typ,
				Name:     n,
			}

			exp := exporter(n)

			o.printCommentGroup(f.doc)

			o.pt(`
			func (s *{{.TypeName}}) {{.Name}}() {{.Type}} {
				return s.{{.Path}}
			}
			`, exp, tmpl)

			switch len(f.path) {
			case 0:
				panic(fmt.Errorf("We have zero path"))
			case 1:
				o.pt(`
				// {{Export "Set"}}{{Capitalise .Name}} is the setter for {{Capitalise .Name}}()
				func (s *{{.TypeName}}) {{Export "Set"}}{{Capitalise .Name}}(n {{.Type}}) *{{.TypeName}} {
					if s.mutable {
						s.{{.Path}} = n
						return s
					}

					res := *s
				`, exp, tmpl)
				if s.special {
					o.pt(`
					res.field_Key.Version++
					`, exp, tmpl)
				}
				o.pt(`
					res.{{.Path}} = n
					return &res
				}
				`, exp, tmpl)
			default:
				o.pt(`
				func (s *{{.TypeName}}) {{Export "Set"}}{{Capitalise .Name}}(n {{.Type}}) *{{.TypeName}} {
				`, exp, tmpl)
				last := "n"
				for i := len(f.path) - 1; i >= 0; i-- {
					v := fmt.Sprintf("v%v", i)
					p := f.path[i]

					rp := strings.Join(f.path[:i], ".")

					if strings.HasSuffix(p, "()") {
						if rp != "" {
							rp = rp + "."
						}
						sp := strings.TrimSuffix(p, "()")

						tmpl := struct {
							V    string
							Rp   string
							Sp   string
							Last string
						}{
							V:    v,
							Rp:   rp,
							Sp:   sp,
							Last: last,
						}
						exp := exporter(sp)
						o.pt(`{{.V}} := s.{{.Rp}}{{Export "Set"}}{{Capitalise .Sp}}({{.Last}})
						`, exp, tmpl)
					} else {
						o.pf("%v := s.%v\n", v, rp)
						o.pf("%v.%v = %v\n", v, p, last)
					}
					last = v
				}
				o.pf("return %v\n", last)
				o.pln("}")
			}
		}
	}
}

func (o *output) printLeadSpecCommsFor(st *ast.StructType) {

	var end token.Pos

	// we are looking for comments before the first field (if there is one)

	if f := st.Fields; f != nil && len(f.List) > 0 {
		end = f.List[0].End()
	} else {
		end = st.End()
	}

	for _, cg := range o.curFile.Comments {
		if cg.Pos() > st.Pos() && cg.End() < end {
			for _, c := range cg.List {
				if strings.HasPrefix(c.Text, "//") && !strings.HasPrefix(c.Text, "// ") {
					o.pln(c.Text)
				}
			}
		}
	}

}
