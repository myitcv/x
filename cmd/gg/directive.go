package main

import "fmt"

type directive struct {
	pkgName string
	file    string
	line    int
	args    []string
	gen     generator
	outDirs []string
}

func (d directive) String() string {
	return fmt.Sprintf("{pkgName: %v, pos: %v:%v, args: %v, gen: [%v], outDirs: %v}", d.pkgName, d.file, d.line, d.args, d.gen, d.outDirs)
}

func (d directive) HashString() string {
	return d.String()
}
