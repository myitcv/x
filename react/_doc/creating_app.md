## Creating a GopherJS React app

This doc article has also been written up in a [blog post](https://blog.myitcv.io/2017/04/16/myitcv.io_react-gopherjs-bindings-for-react.html)

### Initial setup

First verify that you can install the `reactGen` generator and it is on your `PATH`:

```bash
go get -u github.com/gopherjs/gopherjs
go get -u myitcv.io/react myitcv.io/react/cmd/reactGen

# amend PATH for gopherjs and reactGen
export PATH="$(dirname $(go list -f '{{.Target}}' myitcv.io/react/cmd/reactGen)):$PATH"
```

Then:

```bash
reactGen -help
```

should show you the options to `reactGen`, something like:

```
Usage:
        reactGen [-init <template>]
        reactGen [-gglog <log_level>] [-licenseFile <filepath>] [-core]

  -core
        indicates we are generating for a core component (only do props expansion)
  -gglog string
        log level; one of info, warning, error, fatal (default "fatal")
  -init value
        create a GopherJS React application using the specified template (see below)
  -licenseFile string
        file that contains a license header to be inserted at the top of each generated file

...
```

### Create a minimal React app

Create yourself a new directory somewhere in your `GOPATH`; let's assume you call that directory `helloworld`. Within that directory now run:

```bash
mkdir -p $GOPATH/src/example.com/helloworld
cd $_

reactGen -init minimal
```

Let's serve the template app:

```
gopherjs serve
```

and then navigate to [http://localhost:8080/example.com/helloworld](http://localhost:8080/example.com/helloworld)

### Writing components

Now that we have a good starting point, let's assume we want to create a variant on the `HelloMessage` component. This component will have props and state:

```go
// hello_message.go

package main

import (
	"myitcv.io/react"
)

//go:generate reactGen

// Step 1
// Declare a type that has (at least) an anonymous embedded react.ComponentDef
// (it can have other fields); this type must have the suffix 'Def', which corresponds to
// 'Definition'
//
type HelloMessageDef struct {
	react.ComponentDef
}

// Step 2
// Optionally declare a props type; the naming convention is *Props
//
type HelloMessageProps struct {
	Name string
}

// Step 3
// Optionally declare a state type; the naming convention is *State
//
type HelloMessageState struct {
	count int
}
```

With those definitions in place, we can now use the code generator to generate lots of helper code. We do so in the same directory as the component itself:

```bash
go generate
```

This should have created the file `gen_HelloMessage_reactGen.go`.

Now we go ahead and continue adding to `hello_message.go`:

```go
// hello_message.go continued....

// Step 4
// Declare a function to create instances of the component, i.e. an element. If
// your component requires props to be specified, add this to the function
// signature. If the props are optional, use a props pointer type.
//
// buildHelloMessageElem is code generated to wrap a call to react.CreateElement.
//
// Convention is that this function is given the name of the component, HelloMessage
// in this instance. Because this component has props, we also accept these as part
// of the constructor.
//
func HelloMessage(p HelloMessageProps) *HelloMessageElem {
	return buildHelloMessageElem(p)
}

// Step 5
// Define a Render method on the component's non-pointer type
//
func (r HelloMessageDef) Render() react.Element {
	return react.Div(nil,
		react.S("Hello "+r.Props().Name),
	)
}
```

Now re-run `go generate` as before.

```bash
go generate
```

_There is incidentally no harm in having `go generate` run via a watcher on file changes. In fact it removes the manual step of running `go generate` so is a good idea. See below_

At this point you are done defining the component. Why not modify the `App` component that was generated in the `-init` step to render this component?

### `go generate` and `reactGen`

The `//go:generate` directives in the example above tell `go generate` to call `reactGen`. `reactGen` works on a package-by-package basis, finds all components you have declared in that package, and automatically generates the required helper methods/code for those components to work with `myitcv.io/react`. In many respects, you can consider the components you declare as templates that `reactGen` then completes for you. Take a look at the [generated `Timer` code for example](https://github.com/myitcv/react/blob/58cf02dd8ed23f6d62ca8c2470d7de69338d634b/examples/timer/gen_Timer_reactGen.go).

You therefore need to re-run `go generate` (which in turn calls `reactGen`) regularly to ensure the generated files are up-to-date.

Running `go generate` manually is painful, hence it's useful to use a "watcher" tool like [`reflex`](https://github.com/cespare/reflex) to run `go generate` whenever a file changes. The following command "watches" all files in and below the current directory and runs `go generate` when a change is detected:

```bash
reflex go generate ./...
```

See the [reflex documentation](https://github.com/cespare/reflex) for more information.

_**TODO**: optimise the use of `reflex` to only re-run `go generate` on the directory containing the changed file(s)._

### Other lifecycle methods

You can now optionally implement:

* `ComponentWillMount()`
* `ComponentDidMount()`
* `ComponentWillReceiveProps(...)`
* `GetInitialState() ...`
* `ComponentWillUnmount()`

See the [various examples](examples.md) for instances of these methods but take note of the [Gotchas](gotchas.md)

### React's `.js` files

By default, React's [production version `.js` files](https://facebook.github.io/react/docs/installation.html#development-and-production-versions) (`react.min.js` and `react-dom.min.js`) are bundled with a GopherJS build that references the `myitcv.io/react` package. This means you don't have to separately load React (see [the Examples Showcase `index.html` file](https://github.com/myitcv/x/blob/master/react/examples/sites/examplesshowcase/index.html) for example).

To bundle the development version `.js` files (which "includes many helpful warnings"), provide the build tag `debug`:

```bash
# bundle React development version .js files
gopherjs serve --tags debug
```

To prevent any bundling at all, use the `noReactBundle` build tag:

```bash
# do not bundle React
gopherjs serve --tags noReactBundle
```

Using this build tag obviously requires React to have been loaded separately.

### Using Preact instead of React

Initial support for [Preact](https://github.com/developit/preact) is provided via [`preact-compat`](https://github.com/developit/preact-compat) (for now). To use Preact instead of React, simply provide the `preact` build tag:

```bash
# bundle Preact instead of React
gopherjs serve --tags preact
```

Thanks to [@developit](https://github.com/developit) for the pointers on `preact-compat` and [@tj](https://github.com/tj) for the initial inspiration to look into Preact.

### Creating state trees with `stateGen`

_Notes on `stateGen` to follow_
