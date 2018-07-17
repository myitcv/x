<!-- __JSON: go list -json .
## `{{ filepathBase .ImportPath}}`

{{.Doc}}

```
go get -u {{.ImportPath}}
```
-->
## `gitgodoc`

gitgodoc allows you to view `godoc` documentation for different branches of a Git repository

```
go get -u myitcv.io/cmd/gitgodoc
```
<!-- END -->

**Work in progress; see implementation notes below**

`gitgodoc` allows you to view `godoc` documentation for different branches of a Git repository. It is effectively a proxy
that sits in front of multiple `godoc` instances (that `gitgodoc` controls) and relays requests based on the branch you
are interested in.  This is most useful in situations where you either have a private repository or a vendored copy of packages
you depend on.

For example consider the package myitcv.io/immutable. Here we vendor a copy of
`myitcv.io/gogenerate` into the `_vendor` directory. This `_vendor` directory is
then prepended to our `GOPATH` when working with the `myitcv.io/immutable` package
(so our `GOPATH` is effectively `/path/to/gopath/src/myitcv.io/immutable/_vendor:/path/to/gopath`)

If we run `gitgodoc` on the https://github.com/myitcv/immutable repo:

```bash
gitgodoc -serveFrom /tmp/immutable -repo https://github.com/myitcv/immutable.git \
  -pkg myitcv.io/immutable -gopath src/myitcv.io/immutable/_vendor:
```

we can do things like:

```bash
# the documentation for myitcv.io/immutable on the master branch
http://localhost:8080/master/pkg/myitcv.io/immutable

# the documentation for the vendored myitcv.io/gogenerate on the master branch
http://localhost:8080/master/pkg/myitcv.io/gogenerate

# the documentation for myitcv.io/immutable on the feature branch
http://localhost:8080/feature/pkg/myitcv.io/immutable

...

```

The `gitgodoc` server comes to learn about new branches as a result of a webhook call to a refresh
endpoint. [Gitlab](https://about.gitlab.com/) is currently the only supported platform; on pushes to
branches this endpoint gets called:

```
http://localhost:8080/?refresh=gitlab
```

The body of the POST request to this endpoint [gives details of the branch that has been
updated](https://gitlab.com/gitlab-org/gitlab-ce/blob/master/doc/web_hooks/web_hooks.md#push-events).

For now (this is clearly a security hole) you can cheat and tell `gitgodoc` about branches manually:

```bash
echo '{ "ref": "refs/heads/master" }' | \
  curl -X POST -d @- http://localhost:8080.com/?refresh=gitlab --header "Content-Type:application/json"
```

Adding support for other webhook sources should be trivial (see TODO below).

### Security/Implementation

There are many aspects of the implementation which are very insecure and inefficient (for example at
one stage we end up doing rewriting of `hrefs` in HTML); please consider this very much WIP, use at your
own risk etc.

### TODO

See the [TODO in the wiki](https://github.com/myitcv/gitgodoc/wiki/TODO)
