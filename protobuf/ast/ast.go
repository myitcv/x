// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

/*
Package ast defines the AST data structures used by gotoc.
*/
package ast

import (
	"fmt"
	"log"
	"sort"
)

// Node is implemented by concrete types that represent things appearing in a proto file.
type Node interface {
	FileOrNode
	Pos() Position
	File() *File
}

type FileOrNode interface {
	implFileOrNode()
}

type FileOrMessage interface {
	FileOrNode
	implFileOrMessage()
}

type MessageOrExtension interface {
	FileOrNode
	implMessageOrExtension()
}

type MessageOrField interface {
	FileOrNode
	implMessageOrField()
}

// FileSet describes a set of proto files.
type FileSet struct {
	Files []*File
}

// File represents a single proto file.
type File struct {
	Name    string // filename
	Syntax  string // "proto2" or "proto3"
	Package []string
	Options [][2]string // slice of key/value pairs

	Imports       []string
	PublicImports []int // list of indexes in the Imports slice

	Messages   []*Message   // top-level messages
	Enums      []*Enum      // top-level enums
	Services   []*Service   // services
	Extensions []*Extension // top-level extensions

	Comments []*Comment // all the comments for this file, sorted by position
}

var _ FileOrNode = &File{}
var _ FileOrMessage = &File{}

func (f *File) Nodes() []Node {
	var nodes []Node

	for _, v := range f.Messages {
		nodes = append(nodes, v)
	}
	for _, v := range f.Enums {
		nodes = append(nodes, v)
	}
	for _, v := range f.Services {
		nodes = append(nodes, v)
	}

	// TODO support for Extensions? Probably not, it's proto2

	sort.Stable(NodeSort(nodes))

	return nodes
}

func (f *File) implFileOrNode()    {}
func (f *File) implFileOrMessage() {}

// Message represents a proto message.
type Message struct {
	Position       Position // position of the "message" token
	Name           string
	Group          bool
	Fields         []*Field
	Extensions     []*Extension
	Oneofs         []*Oneof
	ReservedFields []Reserved
	Options        [][2]string

	Messages []*Message // includes groups
	Enums    []*Enum

	ExtensionRanges [][2]int // extension ranges (inclusive at both ends)

	Up FileOrMessage // either *File or *Message
}

var _ Node = &Message{}
var _ FileOrMessage = &Message{}
var _ MessageOrExtension = &Message{}
var _ MessageOrField = &Message{}

func (m *Message) implFileOrNode()         {}
func (m *Message) implFileOrMessage()      {}
func (m *Message) implMessageOrExtension() {}
func (m *Message) implMessageOrField()     {}

type Reserved struct {
	Name       string
	Start, End int
}

// Nodes returns a slice of the Nodes contained within this message definition
// i.e. all the fields, enums etc, sorted by their Position.Offset
func (m *Message) Nodes() []Node {
	var nodes []Node

	for _, v := range m.Fields {
		nodes = append(nodes, v)
	}

	// TODO support for Extensions? Probably not, it's proto2

	for _, v := range m.Oneofs {
		nodes = append(nodes, v)
	}

	// TODO support for ReservedFields

	for _, v := range m.Messages {
		nodes = append(nodes, v)
	}
	for _, v := range m.Enums {
		nodes = append(nodes, v)
	}

	sort.Stable(NodeSort(nodes))

	return nodes
}

func (m *Message) Pos() Position { return m.Position }
func (m *Message) File() *File {
	for x := m.Up; ; {
		switch up := x.(type) {
		case *File:
			return up
		case *Message:
			x = up.Up
		default:
			log.Panicf("internal error: Message.Up is a %T", up)
		}
	}
}

// Oneof represents a oneof bracketing a set of fields in a message.
type Oneof struct {
	Position Position // position of "oneof" token
	Name     string

	Up *Message
}

var _ Node = &Oneof{}

func (o *Oneof) implFileOrNode() {}

func (o *Oneof) Pos() Position { return o.Position }
func (o *Oneof) File() *File {
	return o.Up.File()
}

// Field represents a field in a message.
type Field struct {
	Position Position // position of "required"/"optional"/"repeated"/type

	// TypeName is the raw name parsed from the input.
	// Type is set during resolution; it will be a FieldType, *Message or *Enum.
	TypeName string
	Type     interface{}

	// For a map field, the TypeName/Type fields are the value type,
	// and KeyTypeName/KeyType will be set.
	KeyTypeName string
	KeyType     FieldType

	// At most one of {required,repeated} is set.
	Required bool
	Repeated bool
	Name     string
	Tag      int

	HasDefault bool
	Default    string // e.g. "foo", 7, true

	HasPacked bool
	Packed    bool

	HasDeprecated bool
	Deprecated    bool

	Options [][2]string // slice of key/value pairs

	Oneof *Oneof

	Up MessageOrExtension // either *Message or *Extension
}

var _ Node = &Field{}
var _ MessageOrField = &Field{}

func (f *Field) implFileOrNode()     {}
func (f *Field) implMessageOrField() {}

func (f *Field) Pos() Position { return f.Position }
func (f *Field) File() *File {
	switch up := f.Up.(type) {
	case *Message:
		return up.File()
	case *Extension:
		return up.File()
	default:
		log.Panicf("internal error: Field.Up is a %T", up)
		return nil
	}
}

type FieldType int8

const (
	min FieldType = iota
	Double
	Float
	Int64
	Uint64
	Int32
	Fixed64
	Fixed32
	Bool
	String
	Bytes
	Uint32
	Sfixed32
	Sfixed64
	Sint32
	Sint64
	max
)

func (ft FieldType) IsValid() bool { return min < ft && ft < max }

var FieldTypeMap = map[FieldType]string{
	Double:   "double",
	Float:    "float",
	Int64:    "int64",
	Uint64:   "uint64",
	Int32:    "int32",
	Fixed64:  "fixed64",
	Fixed32:  "fixed32",
	Bool:     "bool",
	String:   "string",
	Bytes:    "bytes",
	Uint32:   "uint32",
	Sfixed32: "sfixed32",
	Sfixed64: "sfixed64",
	Sint32:   "sint32",
	Sint64:   "sint64",
}

func (ft FieldType) String() string {
	if s, ok := FieldTypeMap[ft]; ok {
		return s
	}
	return "UNKNOWN"
}

type Enum struct {
	Position Position // position of "enum" token
	Name     string
	Values   []*EnumValue

	Up FileOrMessage // either *File or *Message
}

var _ Node = &Enum{}

func (e *Enum) implFileOrNode() {}

func (enum *Enum) Pos() Position { return enum.Position }
func (enum *Enum) File() *File {
	for x := enum.Up; ; {
		switch up := x.(type) {
		case *File:
			return up
		case *Message:
			x = up.Up
		default:
			log.Panicf("internal error: Enum.Up is a %T", up)
		}
	}
}

type EnumValue struct {
	Position Position // position of Name
	Name     string
	Number   int32

	Up *Enum
}

var _ Node = &EnumValue{}

func (e *EnumValue) implFileOrNode() {}

func (ev *EnumValue) Pos() Position { return ev.Position }
func (ev *EnumValue) File() *File   { return ev.Up.File() }

// Service represents an RPC service.
type Service struct {
	Position Position // position of the "service" token
	Name     string

	Methods []*Method

	Up *File
}

var _ Node = &Service{}

func (s *Service) implFileOrNode() {}

func (s *Service) Pos() Position { return s.Position }
func (s *Service) File() *File   { return s.Up }

// Method represents an RPC method.
type Method struct {
	Position Position // position of the "rpc" token
	Name     string

	// InTypeName/OutTypeName are the raw names parsed from the input.
	// InType/OutType is set during resolution; it will be a *Message.
	InTypeName, OutTypeName string
	InType, OutType         interface{}

	// TODO: support streaming methods
	Options [][2]string // slice of key/value pairs

	Up *Service
}

var _ Node = &Method{}

func (m *Method) implFileOrNode() {}

func (m *Method) Pos() Position { return m.Position }
func (m *Method) File() *File   { return m.Up.Up }

// Extension represents an extension definition.
type Extension struct {
	Position Position // position of the "extend" token

	Extendee     string   // the thing being extended
	ExtendeeType *Message // set during resolution

	Fields []*Field

	Up FileOrMessage // either *File or *Message or ...
}

var _ Node = &Extension{}
var _ MessageOrExtension = &Extension{}

func (e *Extension) implFileOrNode()         {}
func (e *Extension) implMessageOrExtension() {}

func (e *Extension) Pos() Position { return e.Position }
func (e *Extension) File() *File {
	switch up := e.Up.(type) {
	case *File:
		return up
	case *Message:
		return up.File()
	default:
		log.Panicf("internal error: Extension.Up is a %T", up)
	}
	panic("unreachable")
}

// Comment represents a comment.
type Comment struct {
	Start, End Position // position of first and last "//"
	Text       []string
}

func (c *Comment) implFileOrNode() {}

func (c *Comment) Pos() Position { return c.Start }

// LeadingComment returns the comment that immediately precedes a node,
// or nil if there's no such comment.
func LeadingComment(n Node) *Comment {
	f := n.File()
	// Get the comment whose End position is on the previous line.
	lineEnd := n.Pos().Line - 1
	ci := sort.Search(len(f.Comments), func(i int) bool {
		return f.Comments[i].End.Line >= lineEnd
	})
	if ci >= len(f.Comments) || f.Comments[ci].End.Line != lineEnd {
		return nil
	}
	return f.Comments[ci]
}

// InlineComment returns the comment on the same line as a node,
// or nil if there's no inline comment.
// The returned comment is guaranteed to be a single line.
func InlineComment(n Node) *Comment {
	// TODO: Do we care about comments line this?
	// 	string name = 1; /* foo
	// 	bar */

	f := n.File()
	pos := n.Pos()
	ci := sort.Search(len(f.Comments), func(i int) bool {
		return f.Comments[i].Start.Line >= pos.Line
	})
	if ci >= len(f.Comments) || f.Comments[ci].Start.Line != pos.Line {
		return nil
	}
	c := f.Comments[ci]
	// Sanity check; it should only be one line.
	if c.Start != c.End || len(c.Text) != 1 {
		log.Panicf("internal error: bad inline comment: %+v", c)
	}
	return c
}

// Position describes a source position in an input file.
// It is only valid if the line number is positive.
type Position struct {
	Line   int // 1-based line number
	Offset int // 0-based byte offset
}

func (pos Position) IsValid() bool              { return pos.Line > 0 }
func (pos Position) Before(other Position) bool { return pos.Offset < other.Offset }
func (pos Position) String() string {
	if pos.Line == 0 {
		return ":<invalid>"
	}
	return fmt.Sprintf(":%d", pos.Line)
}

type Visitor interface {
	Visit(node Node) (w Visitor)
}

func WalkFile(v Visitor, f *File) {
	var nodes []Node
	for _, m := range f.Messages {
		nodes = append(nodes, m)
	}
	for _, e := range f.Enums {
		nodes = append(nodes, e)
	}
	for _, s := range f.Services {
		nodes = append(nodes, s)
	}

	sort.Sort(NodeSort(nodes))

	for _, n := range nodes {
		Walk(v, n)
	}
}

func Walk(v Visitor, n Node) {
	if v = v.Visit(n); v == nil {
		return
	}

	switch n := n.(type) {
	case *Message:
		var nodes []Node
		for _, m := range n.Messages {
			nodes = append(nodes, m)
		}
		for _, e := range n.Enums {
			nodes = append(nodes, e)
		}
		for _, f := range n.Fields {
			nodes = append(nodes, f)
		}

		sort.Sort(NodeSort(nodes))

		for _, n := range nodes {
			Walk(v, n)
		}
	case *Enum:
		for _, val := range n.Values {
			Walk(v, val)
		}
	case *Service:
		for _, meth := range n.Methods {
			Walk(v, meth)
		}
	}
}
