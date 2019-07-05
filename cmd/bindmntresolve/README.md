<!-- __JSON: go list -json .
### `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

Install using [`gobin`](https://github.com/myitcv/gobin):

```
$ gobin {{.Out.ImportPath}}
```

-->
### `bindmntresolve`

bindmntresolve prints the real directory path on disk of a possibly bind-mounted path

Install using [`gobin`](https://github.com/myitcv/gobin):

```
$ gobin myitcv.io/cmd/bindmntresolve
```

<!-- END -->

<!-- __TEMPLATE: gobin -m -run . -h # NEGATE
### Usage

```
{{.Out -}}
```
-->
### Usage

```
bindmntresolve prints the real path on disk of a possibly bind-mounted path.

usage:
	bindmntresolve [path]

If not path argument is provided, bindmntresolve resolves $PWD (specifically
pwd -P).

```
<!-- END -->
