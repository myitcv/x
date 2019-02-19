<!-- __JSON: go list -json
### `{{.Out.ImportPath}}`

-->
### `myitcv.io/cmd/gg`

<!-- END -->

<!-- __TEMPLATE: gobin -m -run myitcv.io/cmd/gg -h # NEGATE
```
{{.Out}}
```
-->
```
gg is a cached-based wrapper around go generate directives.

Usage:
        gg [-p n] [-r n] [-trace] [-tags 'tag list'] [packages]

gg runs go generate directives found in packages according to the reverse
dependency graph implied by those packages' imports, and the dependencies of
the go generate directives. gg works in both GOPATH and modules modes.

The packages argument is similar to the packages argument for the go command;
see 'go help packages' for more information. In module mode, it is an error if
packages resolves to packages outside of the main module.

The -tags flag is similar to the build flag that can be passed to the go
command. It takes a space-separated list of build tags to consider satisfied as
gg runs, and can appear multiple times.

The -p flag controls the concurrency level of gg. By default will assume a -p
value of GOMAXPROCS. go generate directives only ever run in serial and never
concurrently with other work (this may be relaxed in the future to allow
concurrent execution of go generate directives). A -p value of 1 implies serial
execution of work in a well defined order.

The -trace flag outputs a log of work being executed by gg. It is most useful
when specified along with -p 1 (else the order of execution of work is not well
defined).

Note: at present, gg does not understand the GOFLAGS environment variable.
Neither does it pass the effective build tags via GOFLAGS to each go generate
directive. For more details see:

https://github.com/golang/go/issues/26849#issuecomment-460301061

go generate directives can take three forms:

  //go:generate command ...
  //go:generate gobin -run main_pkg[@version] ...
  //go:generate gobin -m -run main_pkg[@version] ...

The first form, a simple command-based directive, is a relative or absolute
PATH-resolved command.

The second form similar to the command-based directive, except that the path of
the resulting command is resolved via gobin's semantics (see gobin -help).

The third form uses the main module for resolution of the generator and its
dependencies. Those dependencies take part in the reverse dependency graph. Use
of this form is, by definition, only possible in module mode.

gg also understands a special form of directive:

  //go:generate:gg [cond] break

This special directives include a [cond] prefix. At present, a single command
is understood: break. If the preceding conditions are satisfied, no further
go:generate directives are exectuted in this iteration. Note, if spaces are
required in [cond] it must be double-quoted.

The predefined conditions are:

- [exists:file] for whether the (relative) file path exists
- [exec:prog] for whether prog is available for execution (found by
  exec.LookPath)

Where the third form of go generate directive is used, it may be necessary to
declare tool dependencies in your main module. For more information on how to
do this see:

https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

By default, gg uses the directory gg-artefacts under your user cache directory.
See the documentation for os.UserCacheDir for OS-specific details on how to
configure its location. Setting GGCACHE overrides the default.


TODO
====
The following is a rough list of TODOs for gg:

* add support for parsing of GOFLAGS
* add support for setting of GOFLAGS for go generate directives
* consider supporting concurrent execution of go generate directives
* define semantics for when generated files are removed by a generator
* add full tests for cgo

exit status 1

```
<!-- END -->
