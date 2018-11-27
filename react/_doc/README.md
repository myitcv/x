### About

[`myitcv.io/react`](https://godoc.org/myitcv.io/react) is a set of [GopherJS](http://www.gopherjs.org/) bindings for Facebook's [React](https://facebook.github.io/react/), a Javascript library for building user interfaces.

```
go get -u myitcv.io/react
```

See the [Changelog](changelog.md) for notices of significant updates.

### Current state

* All key aspects of React are covered in these bindings
* You can create a GopherJS React-based web app, write components, run and distribute your web app
* GopherJS React components can be integrated with existing React applications, or vice versa
* Examples to get you started, and `reactGen -init` to create a skeleton web app
* Beta; API surface _may_ change

Please [raise issues](https://github.com/myitcv/x/issues/new?title=react:) if you find problems

### Links

For consumers of the package:

* [Examples](examples.md)
* [Creating a GopherJS React app](creating_app.md)
* [Golang UK talk: _"Creating interactive frontend apps with GopherJS and React."_](https://youtu.be/emoUiK-GHkE)
 ([slides](https://blog.myitcv.io/gopherjs_examples_sites/present/?url=https://raw.githubusercontent.com/myitcv/x/master/react/_talks/2017/golang_uk.slide&hideAddressBar=true))
* [Gotchas](gotchas.md) (including significant differences to the React API)

For developers of this package:

* [General design](general_design.md) - random thoughts about the design of this package
* [Code generator design](code_generator_design.md) - brainstorming the workings of the code generation step
* [TODO](todo.md) - a rough list of things to work on next

### Credits

With thanks to the following for their previous work on GopherJS and related projects:

* [@neelance](https://github.com/neelance)
* [@shurcooL](https://github.com/shurcooL)
* [@dominikh](https://github.com/dominikh)
* [@mpl](https://github.com/mpl)
* [@johanbrandhorst](https://github.com/johanbrandhorst) - in particular see Johan's [work on using a gRPC backend](https://github.com/johanbrandhorst/grpcweb-example)
