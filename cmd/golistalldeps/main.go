// golistalldeps transitively lists all dependencies (package, test and xtest) of specified packages in a format
// identical to go list
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
)

var (
	fStd      = flag.Bool("std", false, "include std library packages in output")
	fJson     = flag.Bool("json", false, "output package data in JSON format")
	fTmplSpec = flag.String("f", "{{ .ImportPath }}", "template to use when outputting package data")
)

var pkgInfo = make(map[string]*Package)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	flag.Parse()

	if *fJson {
		log.Fatalln("-json not supported yet")
	}

	pkgs := flag.Args()

	for {
		diffs := goList(pkgs)

		if len(diffs) == 0 {
			break
		}

		pkgs = make([]string, 0)

		for _, p := range diffs {

			if p.Standard {
				continue
			}

			for _, d := range p.Deps {
				if _, ok := pkgInfo[d]; !ok {
					pkgs = append(pkgs, d)
				}
			}

			for _, d := range p.TestImports {
				if _, ok := pkgInfo[d]; !ok {
					pkgs = append(pkgs, d)
				}
			}

			for _, d := range p.XTestImports {
				if _, ok := pkgInfo[d]; !ok {
					pkgs = append(pkgs, d)
				}
			}
		}
	}

	keys := make([]string, 0, len(pkgInfo))

	for k := range pkgInfo {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	// TODO json support

	tSpec := strings.TrimRight(*fTmplSpec, "\n")

	tmpl := template.New("output")
	tmpl, err := tmpl.Parse(tSpec)
	if err != nil {
		log.Fatalf("could not parse template format:\n%v\n", err)
	}

	for _, k := range keys {
		p := pkgInfo[k]

		if !*fStd && p.Standard {
			continue
		}

		tmpl.Execute(os.Stdout, p)

		// having stripped any trailing newlines, we add one in here

		fmt.Println()
	}
}
