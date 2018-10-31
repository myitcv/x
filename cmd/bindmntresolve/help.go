package main

import (
	"fmt"
	"io"
	"text/template"
)

func mainUsage(f io.Writer) {
	t := template.Must(template.New("").Parse(mainHelpTemplate))
	if err := t.Execute(f, nil); err != nil {
		fmt.Fprintf(f, "cannot write usage output: %v", err)
	}
}

var mainHelpTemplate = `
bindmntresolve prints the real path on disk of a possibly bind-mounted path.

usage:
	bindmntresolve [path]

If not path argument is provided, bindmntresolve resolves $PWD (specifically
pwd -P).

`[1:]
