// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package fmt

import (
	"strings"

	"myitcv.io/protobuf/ast"
)

// TODO
//
// We lose spacing (or no spacing) between fields in a message; it should be at most 1 space
// not an enforced 1 space;

func (f *Formatter) FmtFile(file *ast.File) {
	f.fmtSyntax(file.Syntax)
	f.fmtPackage(file.Package)
	f.fmtOptions(file.Options)
	f.fmtImports(file.Imports)

	for _, n := range file.Nodes() {
		switch n := n.(type) {
		case *ast.Message:
			f.fmtMessage(n)
		case *ast.Enum:
			f.fmtEnum(n)
		case *ast.Service:
			f.fmtService(n)
		}
	}
}

func (f *Formatter) fmtSyntax(syntax string) {
	f.printf("syntax = \"%v\";\n", syntax)
	f.println()
}

func (f *Formatter) fmtPackage(pkg []string) {
	f.printf("package %v;\n", strings.Join(pkg, "."))

	if len(pkg) > 0 {
		f.println()
	}
}

func (f *Formatter) fmtOptions(options [][2]string) {
	for _, o := range options {
		f.printf("option %v = %v;\n", o[0], o[1])
	}

	if len(options) > 0 {
		f.println()
	}
}

func (f *Formatter) fmtImports(imports []string) {
	for _, i := range imports {
		f.printf("import \"%v\";\n", i)
	}

	if len(imports) > 0 {
		f.println()
	}
}

func (f *Formatter) fmtService(svc *ast.Service) {
	f.printf("service %v {\n", svc.Name)
	f.indent++

	for _, m := range svc.Methods {
		f.fmtMethod(m)
	}

	f.indent--
	f.println("}")
}

func (f *Formatter) fmtMethod(meth *ast.Method) {
	f.printf("rpc %v (%v) returns (%v)", meth.Name, meth.InTypeName, meth.OutTypeName)
	if len(meth.Options) > 0 {
		f.noIndentPrintf(" {\n")
		f.indent++

		for _, o := range meth.Options {
			f.printf("option (%v) = %v;\n", o[0], o[1])
		}

		f.indent--
		f.println("}")
	} else {
		f.noIndentPrintf(";\n")
	}
}

func (f *Formatter) fmtMessage(message *ast.Message) {
	f.printf("message %v {\n", message.Name)
	f.indent++

	for _, o := range message.Options {
		f.printf("option (%v) = %v;\n", o[0], o[1])

	}

	for _, n := range message.Nodes() {
		switch n := n.(type) {
		case *ast.Message:
			f.fmtMessage(n)
		case *ast.Enum:
			f.fmtEnum(n)
		case *ast.Field:
			f.fmtField(n)
		}
	}

	// TODO: hack; if a one-of field was the last field in a message
	// we need to close out the one-of group
	if f.oneOf != nil {
		f.oneOf = nil
		f.println("}")
	}

	f.indent--
	f.println("}")
}

func (f *Formatter) fmtEnum(enum *ast.Enum) {
	f.printf("enum %v {\n", enum.Name)
	f.indent++

	for _, v := range enum.Values {
		f.printf("%v = %v;\n", v.Name, v.Number)
	}

	f.indent--
	f.println("}")
}

func (f *Formatter) fmtField(field *ast.Field) {
	if field.Oneof != nil && f.oneOf == nil {
		f.oneOf = field.Oneof
		f.printf("oneof %v {\n", field.Oneof.Name)
	} else if field.Oneof == nil && f.oneOf != nil {
		f.oneOf = nil
		f.println("}")
	}

	if field.Oneof != nil {
		f.indent++
	}

	if field.KeyTypeName != "" {
		f.printf("map<%v, %v> %v = %v", field.KeyTypeName, field.TypeName, field.Name, field.Tag)
	} else if field.Repeated {
		f.printf("repeated %v %v = %v", field.TypeName, field.Name, field.Tag)
	} else {
		f.printf("%v %v = %v", field.TypeName, field.Name, field.Tag)
	}

	if len(field.Options) > 0 {
		f.noIndentPrintf(" [")
		for i, o := range field.Options {
			if i > 0 {
				f.noIndentPrintf(", ")
			}
			f.noIndentPrintf("(%v)=%v", o[0], o[1])
		}
		f.noIndentPrintf("];\n")
	} else {
		f.noIndentPrintf(";\n")
	}

	if field.Oneof != nil {
		f.indent--
	}
}
