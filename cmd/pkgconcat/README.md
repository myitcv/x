<!-- __JSON: go list -json .
## `{{ filepathBase .ImportPath}}`

{{.Doc}}

```
go get -u {{.ImportPath}}
```
-->
## `pkgconcat`

pkgconcat is a simple tool that concatenates the contents of a package into a single file

```
go get -u myitcv.io/cmd/pkgconcat
```
<!-- END -->

Used with its `-out` flag, `pkgconcat` can effectively act as a code generator, where the directory/import path is a
"template":

<!-- __TEMPLATE: sh -c "cat ${DOLLAR}(go list -f '{{.Dir}}' myitcv.io/cmd/modpub)/modpub.go | grep \"go:generate pkgconcat\""
```
{{. -}}
```
-->
```
//go:generate pkgconcat -out gen_cliflag.go myitcv.io/_tmpls/cliflag
```
<!-- END -->


<!-- __TEMPLATE: sh -c "${DOLLAR}(go list -f '{{.ImportPath}}' | xargs basename) -h"
### Usage

```
{{. -}}
```
-->
### Usage

```
Usage:

  pkgconcat [directory/import path]

pkgconcat takes an optional single argument that is a directory or an import
path. A directory is indicated by being an absolute path; anything else is
treated as an import path.

Flags:
  -out string
    	path to which to write output; if not specified STDOUT is used
  -outpkgname string
    	name of package to output; if not specified take the package name of the 
    	input directory/import path

```
<!-- END -->
