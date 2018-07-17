// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package parser

// This file implements the symbol resolution stage of parsing.
// TODO: make this more efficient if needed.

import (
	"fmt"
	"strings"

	"myitcv.io/protobuf/ast"
)

func resolveSymbols(fset *ast.FileSet) error {
	r := &resolver{fset: fset}
	s := new(scope)
	s.push(fset)
	for _, f := range fset.Files {
		if err := r.resolveFile(s, f); err != nil {
			return err
		}
	}
	return nil
}

// A scope represents the context of the traversal.
type scope struct {
	// Valid types: FileSet, File, Message, Enum
	objects []interface{}
}

func (s *scope) global() bool       { return len(s.objects) == 0 }
func (s *scope) push(o interface{}) { s.objects = append(s.objects, o) }
func (s *scope) pop()               { s.objects = s.objects[:len(s.objects)-1] }

func (s *scope) dup() *scope {
	sc := &scope{
		objects: make([]interface{}, len(s.objects)),
	}
	copy(sc.objects, s.objects)
	return sc
}

func (s *scope) last() interface{} {
	if s.global() {
		return nil
	}
	return s.objects[len(s.objects)-1]
}

// findName attemps to find the given name in the scope.
// Only immediate names are found; it does not recurse.
func (s *scope) findName(name []string) []interface{} {
	o := s.last()
	if o == nil {
		return nil
	}
	switch ov := o.(type) {
	case *ast.FileSet:
		ret := []interface{}{}
		for _, f := range ov.Files {
			if len(f.Package) == 0 {
				// No package; match on message/enum names
				fs := s.dup()
				fs.push(f)
				ret = append(ret, fs.findName(name)...)
			} else {
				// Match on package name
				// TODO: fix this for dotted package names
				if len(f.Package) == len(name) {
					match := true
					for i := range f.Package {
						match = match && (f.Package[i] == name[i])
						if !match {
							break
						}
					}

					if match {
						return []interface{}{f}
					}
				}
			}
		}
		return ret
	case *ast.File:
		for _, msg := range ov.Messages {
			if msg.Name == name[0] {
				return []interface{}{msg}
			}
		}
		for _, enum := range ov.Enums {
			if enum.Name == name[0] {
				return []interface{}{enum}
			}
		}
	case *ast.Message:
		for _, msg := range ov.Messages {
			if msg.Name == name[0] {
				return []interface{}{msg}
			}
		}
		for _, enum := range ov.Enums {
			if enum.Name == name[0] {
				return []interface{}{enum}
			}
		}
		// can't be *EnumDescriptorProto
	}
	return nil
}

func (s *scope) fullName() string {
	n := make([]string, 0, len(s.objects))
	for _, o := range s.objects {
		switch ov := o.(type) {
		case *ast.File:
			n = append(n, ov.Package...)
		case *ast.Message:
			n = append(n, ov.Name)
		case *ast.Enum:
			n = append(n, ov.Name)
		}
	}
	return "." + strings.Join(n, ".")
}

type resolver struct {
	fset *ast.FileSet
}

func (r *resolver) resolveFile(s *scope, f *ast.File) error {
	fs := s.dup()
	fs.push(f)

	// Resolve messages.
	for _, msg := range f.Messages {
		if err := r.resolveMessage(fs, msg); err != nil {
			return fmt.Errorf("(%v): %v", msg.Name, err)
		}
	}
	// Resolve messages in services.
	for _, srv := range f.Services {
		for _, mth := range srv.Methods {
			if err := r.resolveMethod(fs, mth); err != nil {
				return fmt.Errorf("(%s.%s): %v", srv.Name, mth.Name, err)
			}
		}
	}
	// Resolve types in extensions.
	for _, ext := range f.Extensions {
		if err := r.resolveExtension(fs, ext); err != nil {
			return fmt.Errorf("(ext %s): %v", ext.Extendee, err)
		}
	}

	// TODO: resolve other types.

	return nil
}

var fieldTypeInverseMap = make(map[string]ast.FieldType)

func init() {
	for ft, name := range ast.FieldTypeMap {
		fieldTypeInverseMap[name] = ft
	}
}

var validMapKeyTypes = map[string]bool{
	"int64":    true,
	"uint64":   true,
	"int32":    true,
	"fixed64":  true,
	"fixed32":  true,
	"bool":     true,
	"string":   true,
	"uint32":   true,
	"sfixed32": true,
	"sfixed64": true,
	"sint32":   true,
	"sint64":   true,
}

func (r *resolver) resolveMessage(s *scope, msg *ast.Message) error {
	ms := s.dup()
	ms.push(msg)

	// Resolve fields.
	for _, field := range msg.Fields {
		ft, ok := r.resolveFieldTypeName(ms, field.TypeName)
		if !ok {
			return fmt.Errorf("failed to resolve name %q", field.TypeName)
		}
		field.Type = ft

		if ktn := field.KeyTypeName; ktn != "" {
			if !validMapKeyTypes[ktn] {
				return fmt.Errorf("invalid map key type %q", ktn)
			}
			field.KeyType = fieldTypeInverseMap[ktn]
		}
	}
	// Resolve types in extensions.
	for _, ext := range msg.Extensions {
		if err := r.resolveExtension(ms, ext); err != nil {
			return err
		}
	}
	// Resolve nested types.
	for _, nmsg := range msg.Messages {
		if err := r.resolveMessage(ms, nmsg); err != nil {
			return err
		}
	}
	return nil
}

func (r *resolver) resolveFieldTypeName(s *scope, name string) (interface{}, bool) {
	if ft, ok := fieldTypeInverseMap[name]; ok {
		// field is a primitive type
		return ft, true
	}
	// field must be a named type, message or enum
	o := r.resolveName(s, name)
	if o != nil {
		//log.Printf("(resolved %q to %q)", name, o.fullName())
		return o.last(), true
	}
	return nil, false
}

func (r *resolver) resolveMethod(s *scope, mth *ast.Method) error {
	o := r.resolveName(s, mth.InTypeName)
	if o == nil {
		return fmt.Errorf("failed to resolve name %q", mth.InTypeName)
	}
	mth.InType = o.last()

	o = r.resolveName(s, mth.OutTypeName)
	if o == nil {
		return fmt.Errorf("failed to resolve name %q", mth.OutTypeName)
	}
	mth.OutType = o.last()

	return nil
}

func (r *resolver) resolveExtension(s *scope, ext *ast.Extension) error {
	o := r.resolveName(s, ext.Extendee)
	if o == nil {
		return fmt.Errorf("failed to resolve name %q", ext.Extendee)
	}
	m, ok := o.last().(*ast.Message)
	if !ok {
		return fmt.Errorf("extendee %q resolved to non-message %T", ext.Extendee, o.last())
	}
	ext.ExtendeeType = m
	// Resolve fields.
	for _, field := range ext.Fields {
		ft, ok := r.resolveFieldTypeName(s, field.TypeName)
		if !ok {
			return fmt.Errorf("failed to resolve name %q", field.TypeName)
		}
		field.Type = ft

		// TODO: Map fields should be forbidden?
	}
	return nil
}

func (r *resolver) resolveName(s *scope, name string) *scope {
	parts := strings.Split(name, ".")

	// Move up the scope, finding a place where the name makes sense.
	for ws := s.dup(); !ws.global(); ws.pop() {
		//log.Printf("Trying to resolve %q in %q", name, ws.fullName())
		if os := matchNameComponents(ws, parts); os != nil {
			return os
		}
	}

	return nil // failed
}

func matchNameComponents(s *scope, parts []string) *scope {
	for i := 1; i <= len(parts); i++ {
		first, rem := parts[:i], parts[i:]
		for _, o := range s.findName(first) {
			os := s.dup()
			os.push(o)
			if len(rem) == 0 {
				return os
			}
			// TODO: catch ambiguous names here
			if is := matchNameComponents(os, rem); is != nil {
				return is
			}
		}
	}
	return nil
}
