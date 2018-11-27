## Old Changelog

This changelog was created when the code used to live at https://github.com/myitcv/react

### [2018-05-06](https://github.com/myitcv/react/tree/ef032a4be917529efafec8b459ca4192de13a503) - newborn

* The ability to write `Render()` methods with specific return types (as opposed to `react.Element`). This then allows us to constraint the types of children a component can accept, e.g. `Ul` require that its children implement [`RendersLi`](https://godoc.org/myitcv.io/react#Ul).
* Support for `data-*` and `aria-*` attributes
* Added the [Syntax Viewer](https://blog.myitcv.io/gopherjs_examples_sites/syntaxviewer/) example
* Initial cut of `coreGen` to automate the coding of the `myitcv.io/react` core elements

### [2017-12-08](https://github.com/myitcv/react/tree/bcaf55421745acd10f22033c3dbe6faa2215b5b5) - early Christmas present

* **Breaking change**: tidy up of `myitcv.io/react` package; rename some elements (`<hr>`, `<br>`) for consistent naming
* Added a basic [Bootstrap](https://getbootstrap.com/docs/3.3/)-based template to `reactGen`
* Basic first cut of GopherJS-version of [`present`](https://godoc.org/golang.org/x/tools/cmd/present)
* Upgraded to pin to GopherJS for Go 1.9 support
* Support for `<table>` elements
* Use our own "version" of [`setState`](https://github.com/myitcv/react/commit/a527d183c28be28afb4e41659b639bf0dcaec51e)
* React 16 bundled by default
* [Fragment support](https://reactjs.org/docs/fragments.html)

### [2017-05-02](https://github.com/myitcv/react/tree/2b435e4552cdb6a5dceaa7db9da952c871630c7e) - creating gophers for fun

* **Breaking change**: introduction of element types to complement component definitions. See https://github.com/myitcv/react/pull/73 for more detail
* Add various missing HTML element: `<h4>`, `<i>`, `<footer>`, `<img>`
* Add blog examples to repo so we can be sure they compile
* Another fun example available: https://blog.myitcv.io/gopherize.me_site/

### [2017-05-02](https://github.com/myitcv/react/tree/890c91fce3c81cc2fec2d58a78d20d8a44ff9e67) - CSS, `stateGen` and JSX goodies

* Initial cut of [CSS](https://godoc.org/myitcv.io/react#CSS) support for core HTML components
* **Breaking change**: refactor events to be interface based: [#53](https://github.com/myitcv/react/pull/53)
* By kind permission of [@tjholowaychuk](https://twitter.com/tjholowaychuk), included a basic component-based version of his [Latency Checker](https://blog.myitcv.io/gopherjs_examples_sites/latency/) as an example
* Include Github source ribbon links on all example pages for convenience
* First cut of an [global state example app](https://blog.myitcv.io/gopherjs_examples_sites/globalstate/) that uses [`stateGen`](https://github.com/myitcv/x/tree/master/react/cmd/stateGen): [#61](https://github.com/myitcv/react/pull/61)
* First cut of JSX-like support. All components needing constant blocks of HTML updated to use `jsx.HTML`, `jsx.HTMLElem` or `jsx.Markdown` (see the [latency checker for example](https://github.com/myitcv/react/blob/890c91fce3c81cc2fec2d58a78d20d8a44ff9e67/examples/sites/latency/latency.go#L78-L83)). Also includes a first cut of `reactVet` to ensure correct usage of `myitcv.io/react/jsx`:  [#65](https://github.com/myitcv/react/pull/65)

### [2017-04-19](https://github.com/myitcv/react/tree/827b0efd23aab5fb50b528f6204d5d89e2db7272) - moved to `myitcv.io/react`

* **Breaking change:** moved package to [`myitcv.io/react`](https://myitcv.io/react)
* Use [Highlight.js](https://highlightjs.org/) to highlight code in the [examples showcase](https://blog.myitcv.io/gopherjs_examples_sites/examplesshowcase/)
* Initial cut of support for [Preact](https://github.com/developit/preact) - thanks to [@developit](https://github.com/developit) for the pointers on `preact-compat` and [@tjholowaychuk](https://twitter.com/tjholowaychuk) for the initial inspiration to look into Preact.

### [2017-04-13](https://github.com/myitcv/react/tree/648bf1950ae20f0ad155e4faabc276252c7f3ff9) - proper props

* **Breaking change:** reimplementation of core React component props types. See description in https://github.com/myitcv/react/pull/38 for more details
* Initial cut of `reactGen -init <template>` to help start a new GopherJS React web app - [docs](creating_app.md) updated

### [2017-03-27](https://github.com/myitcv/react/tree/c6a4a02106a183348900b52e1b869146fe88f9f1) - more power to `go generate`

* Bundle React by default - [more details](creating_app.md#creating-a-new-gopherjs-react-app). This enabled us to remove the dependency on NodeJS
* Initial cut of `stateGen`, a `go generate` program that helps with the creation of state trees for GopherJS React applications - [docs to follow](creating_app.md#creating-state-trees-with-stategen)
* Fixed bug whereby [`Render` was called unconditionally](https://github.com/myitcv/react/pull/34) for state-only components when re-rendered by their parent components
* Fixed docs around use of `gopherjs serve`
* Examples now hosted via Github Pages
* `<span>` and `<nav>` HTML elements added
* Proper package docs so that snippets in on https://godoc.org/ are sane
* Fix `.gitattributes` of project for proper detection of language etc.


### [2017-03-15](https://github.com/myitcv/react/tree/9fe41b550ac2299624ad50aa0e90b446b198e772) - a spring in my step

* Generated files now contain comments consistent with https://github.com/golang/go/issues/13560#issuecomment-277804473
* Initial cut of an example component, `immtodoapp.TodoApp`, that uses `github.com/myitcv/immutable` to generate immutable data structures. `Examples` component also re-written to be immutable-based
* Separated example "sites" (i.e. web apps) from the components themselves. This is now reflected in the [Examples](examples.md) docs. Also makes it easier to copy/paste/hack any of the sites and the example components they reference.
* Tidy up `_vendor` directory

### [2017-03-14](https://github.com/myitcv/react/tree/c19b110f5f7b154dd37d753b44f485146fa417f7) - spring clean

* Code cleanup and various bug fixes
* All the [examples](https://github.com/myitcv/x/tree/master/react/examples) updated to latest best practice; updated the live examples site
* More code-level documentation, `go doc`-level comment updates, docs updates (including improved instructions on [getting the examples running locally](examples.md))
* Initial cut of `reactGen`, a `go generate` tool to help with component development - see [Creating a GopherJS React App](creating_app.md)
* Support for `ComponentWillReceiveProps` lifecycle events
* **Breaking change**: synchronous `SetState(...)` (with respect to `State()`) - see [Gotchas](gotchas.md)
* Support for `Equals` methods on state and props types - see [Gotchas](gotchas.md)

### [2016-10-16](https://github.com/myitcv/react/tree/2944fcd25f18439d6e7db90ff71e703cd2faabe7) - Initial Release

* Initial cut of React bindings demonstrated via examples that mirror the [React JS](https://facebook.github.io/react/) homepage examples
* The beginnings of docs, some examples
* No real code-level documentation

---

### WIP - Next "release"

Written based on [this diff](https://github.com/myitcv/react/compare/ef032a4be917529efafec8b459ca4192de13a503...master)


