package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"myitcv.io/immutable/util"
)

func (o *output) calcMethodSets() {

	for _, fts := range o.files {

		typeToString := func(t types.Type) string {
			return types.TypeString(t, func(p *types.Package) string {
				if p.Path() == o.pkgPath {
					return ""
				}

				for i := range fts.imports {
					ip := strings.Trim(i.Path.Value, "\"")
					if p.Path() == ip {
						if i.Name != nil {
							return i.Name.Name
						}
						return p.Name()
					}
				}

				newImport := &ast.ImportSpec{
					Path: &ast.BasicLit{Value: fmt.Sprintf(`"%v"`, p.Path())},
				}
				fts.imports[newImport] = struct{}{}

				return p.Name()
			})
		}
		varTypeString := func(v *types.Var) string {

			if !typeIsInvalid(v.Type()) {
				return typeToString(v.Type())
			}

			f, err := o.findFieldFromVar(v)
			if err != nil {
				panic(fmt.Errorf("failed to findFieldFromVar: %v", err))
			}
			return o.exprString(f.Type)
		}

		for _, is := range fts.structs {
			debugf(">> calculating %v\n", is.name)

			seen := make(map[interface{}]bool)
			set := make(map[string]*field)
			possSet := make(map[string]*field)

			work := []embedded{{es: "*" + is.name}}
			var next []embedded
			var h embedded

			addPoss := func(name string, f field) {
				if _, ok := set[name]; !ok {
					if _, ok := possSet[name]; ok {
						possSet[name] = nil
					} else {
						f.path = append(append([]string(nil), h.path...), f.path...)
						possSet[name] = &f
					}
				}
			}

			for len(work) > 0 {
				h, work = work[0], work[1:]
				debugf(" - examining %v\n", h.es)

				// what do we have?
				if typeIsInvalid(h.typ) {
					if seen[h.es] {
						continue
					}
					seen[h.es] = true
					debugf("using es check\n")
					it, ok := o.immTmpls[h.es]
					if !ok {
						panic(fmt.Errorf("failed to find generated imm type for %v", h.es))
					}

					switch it := it.(type) {
					case *immStruct:
						// here the fields do _not_ have a prefix

						impf := &importFinder{
							imports: it.file.Imports,
							matches: fts.imports,
						}

						for _, f := range it.fields {
							if h.typ == nil {
								// we are at the first level of a struct
								// so the paths must be the prefixed field names
								fname := fieldNamePrefix + fieldHidingPrefix + f.name

								if f.anon {
									fname = fieldAnonPrefix + fname
								}
								addPoss(f.name, field{
									path: []string{fname},
									typ:  o.exprString(f.field.Type),
									doc:  f.field.Doc,
								})
							} else {
								// typeToString adds required imports; for ast-walked
								// types we need to use importFinder
								impf.Visit(f.field.Type)
								addPoss(f.name, field{
									path: []string{f.name + "()"},
									typ:  o.exprString(f.field.Type),
								})
							}

							if f.anon {
								next = append(next, embedded{
									es:   o.exprString(f.field.Type),
									path: append(append([]string(nil), h.path...), fieldTypeToIdent(f.field.Type).Name+"()"),
									typ:  o.info.TypeOf(f.field.Type),
								})
							}

							debugf(")) %v %v %v\n", f.name, f.field.Type, o.exprString(f.field.Type))
						}
					}
				} else {
					type ptr struct {
						types.Type
					}
					kt := h.typ
					if pt, ok := kt.(*types.Pointer); ok {
						kt = ptr{pt.Elem()}
					}
					if seen[kt] {
						continue
					}
					seen[kt] = true
					debugf("using type check on %T %v\n", h.typ, h.typ)
					if v, ok := util.IsImmType(h.typ).(util.ImmTypeStruct); ok {
						is := v.Struct
						for i := 0; i < is.NumFields(); i++ {
							f := is.Field(i)
							name := f.Name()
							isAnon := false
							if strings.HasPrefix(name, "anon") {
								isAnon = true
								name = strings.TrimPrefix(name, "anon")
							}
							if !strings.HasPrefix(name, "field_") {
								continue
							}
							name = strings.TrimPrefix(name, "field_")
							// we can only consider exported fields
							if r, _ := utf8.DecodeRuneInString(name); unicode.IsLower(r) {
								continue
							}
							typStr := varTypeString(f)
							addPoss(name, field{
								typ:  typStr,
								path: []string{name + "()"},
							})

							if isAnon {
								next = append(next, embedded{
									path: append(append([]string(nil), h.path...), name+"()"),
									typ:  f.Type(),
									es:   typStr,
								})
							}
						}
					} else if v, ok := h.typ.Underlying().(*types.Struct); ok {
						for i := 0; i < v.NumFields(); i++ {
							f := v.Field(i)
							if !f.Exported() {
								continue
							}
							typStr := varTypeString(f)
							name := f.Name()
							addPoss(name, field{
								typ:  typStr,
								path: []string{name},
							})
							if f.Anonymous() {
								next = append(next, embedded{
									path: append(append([]string(nil), h.path...), name),
									typ:  f.Type(),
									es:   typStr,
								})
							}
						}
					}
				}

				if len(work) == 0 {
					for n, f := range possSet {
						if f == nil {
							continue
						}
						set[n] = f
					}
					possSet = make(map[string]*field)
					work = next
					next = nil
				}
			}

			is.methods = set

			debugf("-----------\n")
			for n, f := range set {
				if f.path == nil {
					continue
				}

				debugf("%v() %v => %v\n", n, f.typ, strings.Join(f.path, "."))
			}
			debugf("===============\n")
		}
	}
}

func (o *output) findFieldFromVar(v *types.Var) (field *ast.Field, err error) {
	defer func() {
		if v := recover(); v != nil {
			if vf, ok := v.(*ast.Field); ok {
				field = vf
			} else {
				err = fmt.Errorf("unknown error: %v", v)
			}
		}
	}()

	var file *ast.File

	pos := o.fset.Position(v.Pos())

	for af := range o.files {
		if o.fset.Position(af.Pos()).Filename == pos.Filename {
			file = af
		}
	}

	if file == nil {
		return nil, fmt.Errorf("failed to resolve %v to an *ast.File", pos)
	}

	ff := &fieldFinder{
		pos: v.Pos(),
	}

	ast.Walk(ff, file)

	return
}

type fieldFinder struct {
	pos token.Pos
	f   *ast.Field
}

func (f *fieldFinder) Visit(n ast.Node) ast.Visitor {
	if n.Pos() == f.pos {
		f := n.(*ast.Field)
		panic(f)
	}
	if n.Pos() <= f.pos && f.pos <= n.End() {
		return f
	}
	return nil
}
