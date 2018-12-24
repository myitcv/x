package main

import (
	"fmt"
	"path"
)

type gobinModDep struct {
	pkg        *pkg
	importPath string

	hash [hashSize]byte

	depsMap
}

func (g *gobinModDep) Deps() depsMap {
	return g.depsMap
}

func (g *gobinModDep) Ready() bool {
	return g.pkg != nil && len(g.dirtydeps) == 0
}

func (g *gobinModDep) Done() bool {
	return g.Ready() && g.hash != nilHash
}

func (g *gobinModDep) Undo() {
	g.hash = nilHash
}

func (g *gobinModDep) String() string {
	rslvd := "unresolved"
	if g.pkg != nil {
		rslvd = g.pkg.ImportPath
	}
	return fmt.Sprintf("gobinModDep: %v (%v)", g.importPath, rslvd)
}

func (g *gobinModDep) HashString() string {
	return fmt.Sprintf("gobinModDep: %v [%x]", g.importPath, g.pkg.hash)
}

func (g *gobinModDep) DirectiveName() string {
	return path.Base(g.importPath)
}

var _ generator = (*gobinModDep)(nil)
