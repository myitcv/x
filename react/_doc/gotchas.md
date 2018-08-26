## Gotchas

### `SetState(...)` updates `State()` synchronously

[`setState` in ReactJS](https://facebook.github.io/react/docs/react-component.html#setstate) is defined as follows:

```
void setState(
  function|object nextState,
  [function callback]
)
```
> _Performs a shallow merge of nextState into current state..._

> _`setState()` does not immediately mutate `this.state` but creates a pending state transition. Accessing `this.state` after calling this method can potentially return the existing value._

There are two significant differences to these semantics in `myitcv.io/react`:

* `SetState(...)` immediately reflects via `State()`
* There is no shallow merge as state is a struct value. Hence to only partially update state, you need only mutate a copy of the current value:

```go
type MyAppDef struct {
	ComponentDef
}

type MyAppState struct {
	name string
	age  int
}

// ...

func (p *MyAppDef) onNameChange(se *SyntheticEvent) {
	target := se.Target().(*dom.HTMLInputElement)

	ns := p.State()
	ns.name = target.Value

	p.SetState(ns)
}
```

### No requirement for `ShouldComponentUpdate()`

Both state and props are defined as struct types and struct values are used for current state or props. Hence we can generally rely on [comparison](https://golang.org/ref/spec#Comparison_operators) between new and old state/props values to determine whether or not a component should re-`Render`.

In case either state or props struct types are defined with slice, map, and function fields (this is incidentally not advised, docs on immutable values to follow), comparison between struct values cannot be used ([a simple example of this](https://play.golang.org/p/9JgaMsg4nV)). In this case you can define an `Equals` method:

```go
func (c TodoAppState) Equals(v TodoAppState) bool {
    // ...
}
```

### React 16 DOM attributes

React 16 introduced a [change in the way that unknown DOM attributes are handled](https://reactjs.org/blog/2017/09/08/dom-attributes-in-react-16.html). Taking the example from the linked React article, if previously (i.e. React 15 and earlier) you wrote JSX with an attribute that React doesn’t recognise, React would just skip it. For example, this:

```jsx
// Your code:
<div mycustomattribute="something" />
```

would render an empty div to the DOM with React 15:

```jsx
// React 15 output:
<div />
```

In React 16 unknown attributes will end up in the DOM:

```jsx
// React 16 output:
<div mycustomattribute="something" />
```

`myitcv.io/react` does **NOT** support this, ror the obvious reason that the props of the DOM elements are predefined (and hence as the caller you can't provided ad hoc, custom attributes else it would be a compile error).

Conceivably we could support this via a catch-all, `map[string]interface{}`-typed attribute on all DOM element prop types. But this feels like a bit of a back door and one that right now doesn't have anyone knocking at it. That is to say, we'll wait until this becomes a massively pressing need before changing the status quo.

### React 16

As of [commit `b382bf4`](https://github.com/myitcv/react/commit/b382bf4b89b3dcaa3e64eef8547f4abc02f81c7b) `myitcv.io/react` bundles React 16.

For React 15, please use the `react_15` branch.
