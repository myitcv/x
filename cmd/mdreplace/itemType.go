package main

import "fmt"

//go:generate stringer -type=itemType -output=gen_itemType.go

type itemType int

const (
	itemError itemType = iota

	itemEOF

	itemText

	itemCodeFence
	itemCode

	itemTmplBlockStart
	itemJsonBlockStart

	itemBlockEnd
	itemCommEnd

	itemArg
	itemQuoteArg
)

type item struct {
	typ itemType
	val string
}

func (i item) String() string {
	return fmt.Sprintf("{typ: %v, val: %q}", i.typ, i.val)
}
