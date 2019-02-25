package main

import (
	"fmt"
	"io"
)

func mainUsage(f io.Writer) {
	fmt.Fprint(f, mainHelp)
}

var mainHelp = `
The vartojson command writes the JSON marshaled value of a variable to a file.
`[1:]
