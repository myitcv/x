package gogenerate

import "fmt"

type directive struct {
	pkgName     string
	file        string
	line        int
	args        []string
	gen         generator
	outDirs     []string
	inFilePatts []string
}

func (d directive) String() string {
	return fmt.Sprintf("{pkgName: %v, pos: %v:%v, args: %v, gen: [%v], outDirs: %v, inFilePatts: %v}", d.pkgName, d.file, d.line, d.args, d.gen, d.outDirs, d.inFilePatts)
}

func (d directive) HashString() string {
	return d.String()
}
