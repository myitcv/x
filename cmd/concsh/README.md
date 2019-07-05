<!-- __JSON: go list -json .
## `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

```
go get -u {{.Out.ImportPath}}
```
-->
## `concsh`

concsh allows you to concurrently run commands from your shell.

```
go get -u myitcv.io/cmd/concsh
```
<!-- END -->


<!-- __TEMPLATE: gobin -m -run . -h
### Usage

```
{{.Out -}}
```
-->
### Usage

```
concsh allows you to concurrently run commands from your shell

Usage:
	concsh -- comand1 arg1_1 arg1_2 ... --- command2 arg2_1 arg 2_2 ... --- ...
	concsh

In the case no arguments are provided, concsh will read the commands to execute
from stdin, one per line

Flags:
  -conc uint
    	define how many commands can be running at any given time; 0 = no limit;
    	default = 0
  -debug
    	debug output

```
<!-- END -->

All args after the first `--` are then considered as a `---`-separated (notice the extra `-`) list of commands to be run
concurrently. Output from each command (both stdout and stderr) is output to the `concsh`'s stdout and stderr when a
command finishes executing; output is not interleaved between commands, that is to say output is grouped by command
(although the distinction between stdout and stderr is retained)

The exit code from `concsh` is `0` if all commands succeed without error, or one of the non-zero exit codes otherwise

### Example

<!-- __TEMPLATE: cat _example/example.sh
```bash
{{.Out -}}
```
-->
```bash
set -eu

# ** SCRIPT START **
timer="go run $(go list -f '{{.Dir}}' myitcv.io/cmd/concsh)/_example/timer.go"
gobin -m -run myitcv.io/cmd/concsh -- $timer 1 --- $timer 2 --- $timer 3 --- $timer 4 --- $timer 5
```
<!-- END -->

which gives output similar to:

<!-- __TEMPLATE: sh _example/example.sh # SORTINVARIANT LONG
```
{{.Out -}}
```
-->
```
Instance 4 iteration loop 1
Instance 4 iteration loop 2
Instance 4 iteration loop 3
Instance 4 iteration loop 4
Instance 4 iteration loop 5
Instance 1 iteration loop 1
Instance 1 iteration loop 2
Instance 1 iteration loop 3
Instance 1 iteration loop 4
Instance 1 iteration loop 5
Instance 2 iteration loop 1
Instance 2 iteration loop 2
Instance 2 iteration loop 3
Instance 2 iteration loop 4
Instance 2 iteration loop 5
Instance 3 iteration loop 1
Instance 3 iteration loop 2
Instance 3 iteration loop 3
Instance 3 iteration loop 4
Instance 3 iteration loop 5
Instance 5 iteration loop 1
Instance 5 iteration loop 2
Instance 5 iteration loop 3
Instance 5 iteration loop 4
Instance 5 iteration loop 5
```
<!-- END -->

See how the output is grouped per command.
