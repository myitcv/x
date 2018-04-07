package main

import (
	"strings"
	"text/template"
)

var tmplFuncMap = template.FuncMap{
	"lines": func(s string) []string {
		return strings.Split(s, "\n")
	},
}

func (p *processor) processTmplBlock() procFn {
	return p.processCommonBlock(tmplBlock, func(out []byte) interface{} {
		return string(out)
	})
}
