### `modpub`

`modpub` is a tool to help create a directory of [`vgo`](https://github.com/golang/go/issues/24301) modules from a git
repository.

```
go get -u myitcv.io/cmd/modpub
```

### Usage

<!-- __TEMPLATE: sh -c "modpub -h || true"
```
{{. -}}
```
-->
```
Usage of modpub:
  -target string
    	target directory for publishing
  -v	give verbose output
```
<!-- END -->

### Example

The [`myitcv.io/...` mono-repo](https://github.com/myitcv/x) has its `vgo` modules published to
https://github.com/myitcv/pubx. Once we get a resolution on https://github.com/golang/go/issues/24751, `pubx` will serve
as the `go-import` `mod` target.

`pubx` is effectively built using the following commands:

<!-- __TEMPLATE: sh -c "sh _scripts/readme_example > /dev/null 2>&1 && cat _scripts/readme_example"
```bash
{{ trimLinePrefixWhitespace . "# ** SCRIPT START **" }}
```
-->
```bash
# clone the mono repo
git clone -q https://github.com/myitcv/x src/myitcv.io

# get modpub
go get -u myitcv.io/cmd/modpub

# create our target directory
mkdir pubx

cd src/myitcv.io
git checkout -qf c57b27668caebdef755844c84016f8bf1cf618fe

modpub -target ../../pubx

```
<!-- END -->

The resulting directory structure can be seen in the https://github.com/myitcv/pubx repository.

### Status

Very much work in progress. There are some notable TODOs:

* currently only works with commits (and not tags) and hence produces only pre-release versions
* currently we assume the remote is `origin/master`; make this configurable
