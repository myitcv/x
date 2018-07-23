// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

// Package immutable is a helper package for the immutable data structures
// generated by myitcv.io/immutable/cmd/immutableGen.
//
package immutable

const (
	CmdImmutableGen = "immutableGen"
	CmdImmutableVet = "immutableVet"
)

const (
	// ImmTypeTmplPrefix is the prefix used to identify immutable type templates
	ImmTypeTmplPrefix = "_Imm_"

	// Pkg is the import path of this package
	PkgImportPath = "myitcv.io/immutable"
)

// Immutable is the interface implemented by all immutable types. If Go had generics the interface would
// be defined, assuming a generic type parameter T, as follows:
//
// 	type Immutable<T> interface {
// 		AsMutable() T
// 		AsImmutable() T
// 		WithMutations(f func(v T)) T
// 		Mutable() bool
// 	}
//
// Because we do not have such a type parameter we can only define the Mutable() method in the interface
type Immutable interface {
	Mutable() bool
	IsDeeplyNonMutable(seen map[interface{}]bool) bool
}
