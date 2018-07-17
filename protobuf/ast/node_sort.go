// Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
// Use of this document is governed by a license found in the LICENSE document.

package ast

type NodeSort []Node

func (a NodeSort) Len() int      { return len(a) }
func (a NodeSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a NodeSort) Less(i, j int) bool {
	return a[i].Pos().Offset < a[j].Pos().Offset
}
