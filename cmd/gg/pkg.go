package main

import (
	"fmt"
)

type pkg struct {
	// generate indicates whether we should run generate on this package
	generate bool

	// generated indicates that the package has been generated
	generated bool

	*Package

	hash [hashSize]byte

	depsMap

	dirs []directive

	isXTest bool
	x       *pkg

	genCount int
}

func (p *pkg) Deps() depsMap {
	return p.depsMap
}

func (p *pkg) Ready() bool {
	return p.Package != nil && len(p.dirtydeps) == 0
}

func (p *pkg) Done() bool {
	return p.Ready() && (!p.generate || p.generated) && p.hash != nilHash
}

func (p *pkg) Undo() {
	p.generated = false
	p.hash = nilHash
}

func (p *pkg) String() string {
	var generate string
	if p.generate {
		generate = " [G]"
	}
	return fmt.Sprintf("{Pkg: %v%v}", p.Package.ImportPath, generate)
}

func (p *pkg) HashString() string {
	return fmt.Sprintf("pkg: %v [%x]", p.ImportPath, p.hash)
}

var _ dep = (*pkg)(nil)
