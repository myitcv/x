package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"

	"github.com/rogpeppe/go-internal/imports"
)

//go:generate gobin -m -run myitcv.io/cmd/helpflagtopkgdoc

type tagsFlag struct {
	vals []string
}

func (e *tagsFlag) String() string {
	return fmt.Sprintf("%v", e.vals)
}

func (e *tagsFlag) Set(v string) error {
	e.vals = append(e.vals, v)
	return nil
}

func main() {
	os.Exit(main1())
}

func main1() int {
	switch err := mainerr(); err {
	case nil:
		return 0
	case flag.ErrHelp:
		return 2
	default:
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
}

func mainerr() (retErr error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		mainUsage(os.Stderr)
	}
	var tagsVals tagsFlag
	fs.Var(&tagsVals, "tags", "tags for build list")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if len(fs.Args()) != 1 {
		return fmt.Errorf("expected a single arg; the variable to marshal")
	}
	varName := fs.Arg(0)

	goos := os.Getenv("GOOS")
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	tags := map[string]bool{
		goos:   true,
		goarch: true,
	}
	for _, v := range tagsVals.vals {
		for _, vv := range strings.Fields(v) {
			tags[vv] = true
		}
	}

	fset := token.NewFileSet()
	matchFile := func(fi os.FileInfo) bool {
		return imports.MatchFile(fi.Name(), tags)
	}
	pkgs, err := parser.ParseDir(fset, ".", matchFile, 0)
	if err != nil {
		return fmt.Errorf("failed to parse current directory: %v", err)
	}

	pkgName := os.Getenv("GOPACKAGE")
	pkg := pkgs[pkgName]

	if pkg == nil {
		return fmt.Errorf("failed to find package for package name %v", pkgName)
	}

	type match struct {
		file *ast.File
		expr ast.Expr
	}

	var matches []match
	typeDecls := make(map[string]*ast.TypeSpec)

	for _, f := range pkg.Files {
		var comments bytes.Buffer
		for _, cg := range f.Comments {
			if cg == f.Doc || cg.Pos() > f.Package {
				break
			}

			for _, cm := range cg.List {
				comments.WriteString(cm.Text + "\n")
			}

			comments.WriteString("\n")
		}

		if !imports.ShouldBuild(comments.Bytes(), tags) {
			continue
		}

		for _, gd := range f.Decls {
			switch gd := gd.(type) {
			case *ast.GenDecl:
				for _, s := range gd.Specs {
					switch gd.Tok {
					case token.VAR:
						vs := s.(*ast.ValueSpec)
						if len(vs.Values) == 0 {
							// no value; nothing to do
							continue
						}
						for i, name := range vs.Names {
							expr := vs.Values[i]
							if varName == name.Name {
								matches = append(matches, match{
									file: f,
									expr: expr,
								})
							}
						}
					case token.TYPE:
						ts := s.(*ast.TypeSpec)
						typeDecls[ts.Name.Name] = ts
					}
				}
			}
		}
	}

	switch len(matches) {
	case 0:
		return fmt.Errorf("failed to find declaration of %v", varName)
	case 1:
	default:
		var dups []string
		for _, m := range matches {
			dups = append(dups, fmt.Sprintf("found declaration of %v at %v", varName, fset.Position(m.expr.Pos())))
		}
		return fmt.Errorf("%v", strings.Join(dups, "\n"))
	}

	theMatch := matches[0]

	imports := make(map[*ast.ImportSpec]bool)
	usedTypes := make(map[*ast.TypeSpec]bool)

	work := []ast.Node{theMatch.expr}

	visitType := func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.SelectorExpr:
			if x, ok := node.X.(*ast.Ident); ok {
				for _, imp := range pkg.Files[fset.File(node.Pos()).Name()].Imports {
					if imp.Name != nil {
						if x.Name == imp.Name.Name {
							imports[imp] = true
						}
					} else {
						cleanPath := strings.Trim(imp.Path.Value, "\"")
						parts := strings.Split(cleanPath, "/")
						if x.Name == parts[len(parts)-1] {
							imports[imp] = true
						}
					}
				}
			}
			// we have handled the qualified identifier; do no inspect its Idents
			return false
		case *ast.Ident:
			typ := typeDecls[node.Name]
			if typ != nil {
				usedTypes[typ] = true
			} else if types.Universe.Lookup(node.Name) == nil {
				panic(fmt.Errorf("failed to find type declaration for %v", node.Name))
			}
			work = append(work, node)
		}
		return true
	}

	visitVal := func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CompositeLit:
			if node.Type != nil {
				ast.Inspect(node.Type, visitType)
			}
		case *ast.BasicLit:
			ast.Inspect(node, visitType)
		}
		return true
	}

	err = func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = r.(error)
			}
		}()
		for len(work) > 0 {
			w := work[0]
			work = work[1:]
			ast.Inspect(w, visitVal)
		}
		return
	}()

	if err != nil {
		return fmt.Errorf("failed to walk AST: %v", err)
	}

	var tempFile string
	var lock sync.Mutex

	ctrlc := make(chan os.Signal)
	signal.Notify(ctrlc, os.Interrupt)
	go func() {
		<-ctrlc
		lock.Lock()
		if tempFile != "" {
			os.Remove(tempFile)
		}
		os.Exit(1)
	}()

	lock.Lock()
	tf, err := ioutil.TempFile(".", "vartojson.*.go")
	if err != nil {
		lock.Unlock()
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tempFile = tf.Name()
	defer func() {
		lock.Lock()
		defer lock.Unlock()
		os.Remove(tempFile)
	}()
	lock.Unlock()

	var buf bytes.Buffer
	p := func(format string, args ...interface{}) {
		fmt.Fprintf(&buf, format, args...)
	}
	pp := func(node interface{}) string {
		var sb strings.Builder
		printer.Fprint(&sb, fset, node)
		return sb.String()
	}

	p(`
package main

import (
	"fmt"
	"encoding/json"
)
`[1:])
	for i := range imports {
		p("import %v\n", pp(i))
	}

	for t := range usedTypes {
		p("type %v\n", pp(t))
	}

	p(`
func main() {
	v := %v

	byts, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		 panic(err)
	}
	fmt.Println(string(byts))
}
`[1:], pp(theMatch.expr))

	if err := ioutil.WriteFile(tempFile, buf.Bytes(), 0666); err != nil {
		return fmt.Errorf("failed to write temp main")
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", tempFile)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to %v: %v\n%s", strings.Join(cmd.Args, " "), err, stderr.Bytes())
	}

	// reparse into an interface{} and write out again... so that we get consistently formatted JSON
	// output
	var i interface{}
	if err := json.Unmarshal(stdout.Bytes(), &i); err != nil {
		return fmt.Errorf("failed to Unmarshal JSON: %v", err)
	}

	toWrite, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to re-Marshal JSON: %v", err)
	}

	fn := "gen_" + varName + "_vartojson.json"
	if err := ioutil.WriteFile(fn, append(toWrite, '\n'), 0666); err != nil {
		return fmt.Errorf("failed to write %v: %v", fn, err)
	}

	return nil
}
