<!-- __JSON: go list -json .
## `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

```
go get -u {{.Out.ImportPath}}
```
-->
## `hybridimporter`

Package hybridimporter is an implementation of go/types.ImporterFrom that uses depdency export information where it can, falling back to a source-file based importer otherwise.

```
go get -u myitcv.io/hybridimporter
```
<!-- END -->
### `myitcv.io/hybridimporter`

This is essentially a work-in-progress and will become obsolete when
[`go/packages`](https://github.com/golang/go/issues/14120#issuecomment-383994980) lands. The importer discussed in that
thread will be able to take advantage of the build cache and also be go module-aware.

Currently relies on Go tip as of
[baf399b02e](https://go.googlesource.com/go/+/f7248f05946c1804b5519d0b3eb0db054dc9c5d6), which is due to be part of Go
1.11.
