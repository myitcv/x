## `mdreplace`

`mdreplace` is a tool to help you keep your markdown README/documentation current.

```
go get -u myitcv.io/cmd/mdreplace
```

_(will soon be available as a [`vgo` module](https://github.com/golang/go/issues/24301))_

A common problem with non `.go` documentation files is that their contents can easily become stale. For example with a
program it's common to include a "Help" section in the corresponding `README.md` which typically involves a discussion
of the program's flags. The contents of the `README.md` can, however, easily fall out of step with respect to the
_actual_ flags were we to run our program with `-h` (or equivalent) today.

`mdreplace` helps alleviate these problems by allowing you to insert special comment blocks in your markdown files that
are replaced with command output.

For example, were we to include the following special comment block:

```
<!-- __TEMPLATE: echo -n "hello world" today
{{.}}
-->
<!-- END -->
```

then this is what would result (see the [source of the
`README.md`](https://raw.githubusercontent.com/myitcv/x/master/cmd/mdreplace/README.md) you are currently reading):

---
<!-- __TEMPLATE: echo -n "hello world" today
{{.}}
-->
hello world today
<!-- END -->
---

The `__TEMPLATE` block provides the following template functions:

* `lines(string) []string` - split a string into lines


### Code fences

Code fences can appear within templates. Hence the following special template within a markdown file:

    <!-- __TEMPLATE: echo -n "hello world"
    ```go
    package main

    import "fmt"

    func main() {
            fmt.Println("{{.}}")
    }
    ```
    -->
    <!-- END -->

results in:

---
<!-- __TEMPLATE: echo -n "hello world"
```go
package main

import "fmt"

func main() {
	fmt.Println("{{.}}")
}
```
-->
```go
package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
```
<!-- END -->
---



The only place that special comment blocks are _not_ interpreted by `mdreplace` is within code blocks. Hence how we are
able to render the example special code blocks in this README.

_Note it is not possible to nest code fences._


### Variable expansion

Variable expansion also works; use the special `$DOLLAR` variable to get expand to the literal `$` sign:

```
<!-- __TEMPLATE: sh -c "BANANA=fruit; echo -n \"${DOLLAR}BANANA\""
{{.}}
-->
<!-- END -->
```

results in:

---
<!-- __TEMPLATE: sh -c "BANANA=fruit; echo -n \"${DOLLAR}BANANA\""
{{.}}
-->
fruit
<!-- END -->
---

### Implementation

This rather basic program is an implementation of the techniques proposed by [Rob Pike](https://twitter.com/rob_pike) in
his brilliant presentation [Lexical Scanning in Go](https://youtu.be/HxaD_trXwRE)
([slides](https://talks.golang.org/2011/lex.slide#1)).

