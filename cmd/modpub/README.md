<!-- __JSON: go list -json .
## `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

```
go get -u {{.Out.ImportPath}}
```
-->
## `modpub`

modpub is a tool to help create a directory of vgo modules from a git respository.

```
go get -u myitcv.io/cmd/modpub
```
<!-- END -->


<!-- __TEMPLATE: bash -c "${DOLLAR}(go list -f '{{.ImportPath}}' | xargs basename) -h"
### Usage

```
{{.Out -}}
```
-->
### Usage

```
Flags:
  -target string
    	target directory for publishing
  -v	give verbose output

```
<!-- END -->

### Status

Very much work in progress. There are some notable TODOs:

* currently only works with commits (and not tags) and hence produces only pre-release versions
* currently we assume the remote is `origin/master`; make this configurable
