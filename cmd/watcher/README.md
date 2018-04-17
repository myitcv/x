<!-- __JSON: go list -json .
## `{{ filepathBase .ImportPath}}`

{{.Doc}}

```
go get -u {{.ImportPath}}
```
-->
## `watcher`

watcher is a Linux-based directory watcher for triggering commands

```
go get -u myitcv.io/cmd/watcher
```
<!-- END -->


<!-- __TEMPLATE: sh -c "${DOLLAR}(go list -f '{{.ImportPath}}' | xargs basename) -h"
### Usage

```
{{.}}
```
-->
### Usage

```
Flags:
  -I value
    	Paths to ignore. Absolute paths are absolute to the path; relative paths
    	can match anywhere in the tree
  -c	do not clear the screen before running the command
  -d	die on first notification; only consider -p and -f flags
  -debug
    	give debug output
  -f	whether to follow symlinks or not (recursively) [*]
  -i	don't run command at time zero; only applies when -d not supplied
  -k	don't kill the running command on a new notification
  -p string
    	the path to watch; default is CWD [*]
  -q duration
    	the duration of the 'quiet' window; format is 1s, 10us etc. Min 1
    	millisecond (default 100ms)
  -t duration
    	the timeout after which a process is killed; not valid with -k


```
<!-- END -->

### Status

Code is _very_ rough. It was some of my first Go code and so hasn't seen much love since then. Consider as WIP.
