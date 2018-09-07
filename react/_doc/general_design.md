## General Design

### Terminology

* **Component** - a type and corresponding method set definition; this is equivalent to a React component, e.g. the `HelloMessage` component is defined by the type [`HelloMessageDef`](https://godoc.org/myitcv.io/react/examples/hellomessage#HelloMessageDef) and corresponding method set defined on `*HelloMessageDef`
* **Element** - a component's pointer type value, e.g. `&HelloMessageDef{ ... }`, generally created via a call to a components exported helper function, e.g. [`HelloMessage`](https://godoc.org/myitcv.io/react/examples/hellomessage#HelloMessage)
* **Props** - a component's corresponding props type value, e.g. [`HelloMessageProps{ ... }`](https://godoc.org/myitcv.io/react/examples/hellomessage#HelloMessageProps) (which is the props type for the `HelloMessage` component)
* **State** - a component's corresponding state type value, e.g. [`TimerState{ ... }`](https://godoc.org/myitcv.io/react/examples/timer#TimerState) (which is the state type for the [`Timer`](https://godoc.org/myitcv.io/react/examples/timer#TimerDef) component)

### Code generation

The general approach being proposed is that components will be declared by the developer, followed by some element of code generation to do the grunt work of helper-method generation etc. More details to follow

See the docs for [code generator design](code_generator_design.md)

### Naming convention

What naming convention should we use for:

* packages
* types (components, props and state)
* functions (e.g. to create elements from components)

when it comes to defining GopherJS React components?

We need to consider:

* intrinsic components
* user-defined components

(the answers may be different)

```go
// ********
// Option 1

// myitcv.io/react
package react

type ADef struct { ... }   // the <a ...> component
type AProps struct { ... } // props type for <a ...> component
func A(...) *ADef { ... }  // create an <a ...> element


// myitcv.io/react/examples/hellomessage
package hellomessage

type HelloMessageDef struct { ... } // the <HelloMessage ... > component
type HelloMessageProps struct { ... } // props type for <HelloMessage ...> component
type HelloMessageState struct { ... } // state type for <HelloMessage ...> component

```

### PureRender

Whether to automatically `PureRender` for components whose corresponding state and props values are [comparable](https://golang.org/ref/spec#Comparison_operators)


### Intrinsic components

Assuming there is some conclusion to https://github.com/gopherjs/gopherjs/issues/236, we can do away with the existing helper-functions, e.g. `PProps`, rename the types, e.g. `PPropsDef -> PProps`, and use the types directly:

```go
x := P(&PProps{ClassName: "wide"}, ...)
```

For usability reasons, this however means we need to move away from the embedding of, for example `*react.BasicHTMLElement`, to explicit definition of fields:

```go
type PProps struct {
	o *js.Object

	Id        string `js:"id"`
	Key       string `js:"key"`
	ClassName string `js:"className"`

	// ...
}
```

This will clearly require some sort of code generation based on some higher-order definition of the props for each component.

### Unified state

Follow up with [neelance](https://github.com/neelance) about a different API for the `react` package that unifies state and props

