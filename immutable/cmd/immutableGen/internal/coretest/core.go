package coretest

import (
	"time"

	"myitcv.io/immutable"
	"myitcv.io/immutable/cmd/immutableGen/internal/coretest/pkga"
)

//go:generate immutableGen -licenseFile license.txt -G "echo \"hello world\""

// a comment about MyMap
type _Imm_MyMap map[string]int

// a comment about Slice
type _Imm_MySlice []string

type MyStructUuid uint64

type MyStructKey struct {
	Uuid    MyStructUuid
	Version uint64
}

// a comment about myStruct
type _Imm_MyStruct struct {
	Key MyStructKey

	// my field comment
	//somethingspecial
	/*

		Heelo

	*/
	Name, surname string `tag:"value"`
	age           int    `tag:"age"`

	string

	fieldWithoutTag bool
}

type _Imm_A struct {
	Name string
	A    *A

	Blah
}

type _Imm_AS []*A

type _Imm_AM map[*A]*A

type Blah interface {
	immutable.Immutable
}

type _Imm_BlahUse struct {
	Blah
}

type BlahMutable struct{}

var _ Blah = BlahMutable{}

func (b BlahMutable) Mutable() bool {
	return true
}

func (b BlahMutable) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
	return false
}

type BlahNonMutable struct{}

var _ Blah = BlahNonMutable{}

func (b BlahNonMutable) Mutable() bool {
	return false
}

func (b BlahNonMutable) IsDeeplyNonMutable(seen map[interface{}]bool) bool {
	return true
}

type _Imm_Clash1 struct {
	Clash    string
	NoClash1 string
}

// types for testing embedding
type _Imm_Embed1 struct {
	Name string
	*Embed2
	*pkga.PkgA
	*Clash1
	*pkga.Clash2
	NonImmStruct
	pkga.NonImmStructA
}

type _Imm_Embed2 struct {
	Age int
}

type NonImmStruct struct {
	Now time.Time
	*Other
}

type _Imm_Other struct {
	OtherName string
}

func main() {
}
