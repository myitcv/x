package util

import (
	"go/ast"
	"strings"

	"myitcv.io/immutable"
)

const (
	debug = false
)

// IsImmTmpl determines whether the supplied type spec is an immutable template type (either a struct,
// slice or map), returning the name of the type with the ImmTypeTmplPrefix removed in that case
func IsImmTmpl(ts *ast.TypeSpec) (string, bool) {
	typName := ts.Name.Name

	if !strings.HasPrefix(typName, immutable.ImmTypeTmplPrefix) {
		return "", false
	}

	valid := false

	switch typ := ts.Type.(type) {
	case *ast.MapType:
		valid = true
	case *ast.ArrayType:
		if typ.Len == nil {
			valid = true
		}
	case *ast.StructType:
		valid = true
	}

	if !valid {
		return "", false
	}

	name := strings.TrimPrefix(typName, immutable.ImmTypeTmplPrefix)

	return name, true
}
