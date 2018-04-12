### `myitcv.io/...` mono-repo

<!-- __TEMPLATE: go list -f "{{${DOLLAR}ip := .ImportPath}}{{range .Deps}}{{if (eq \"myitcv.io/vgo\" .)}}{{${DOLLAR}ip}}{{end}}{{end}}" ./...
{{ with . }}
Please note the following packages current rely on `vgo` with https://go-review.googlesource.com/c/vgo/+/105855 applied:

```
{{. -}}
```
{{end -}}
-->

Please note the following packages current rely on `vgo` with https://go-review.googlesource.com/c/vgo/+/105855 applied:

```
myitcv.io/immutable/cmd/immutableVet
```
<!-- END -->

