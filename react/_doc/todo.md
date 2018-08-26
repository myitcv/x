## TODO

### Next

* Create issues for the items in this rough TODO list
* Create automated tests to ensure the examples work
* Investigate https://material.io/components/web/catalog/
* Work out what to do in response to https://reactjs.org/blog/2018/03/27/update-on-async-rendering.html
* Upgrade to use Bootstrap 4
* Ensure that examples on blog and docs remain current/runnable
* Work out what to do with `_vendor`. Potentially move to another repo if they are only development deps?
* Get resolution on these core GopherJS issues (in order of criticalness/importance):
  * https://github.com/gopherjs/gopherjs/issues/617 - tighten up semantics of `$internalize`
  * https://github.com/gopherjs/gopherjs/issues/661 - struct value bug
  * https://github.com/gopherjs/gopherjs/issues/692 - function closure bug
  * https://github.com/gopherjs/gopherjs/issues/633 - explicit `*js.Object-special` struct types
  * https://github.com/gopherjs/gopherjs/issues/634 - implicit Object instantiation for `*js.Obect-special` struct types (so that we can avoid the need for [proxy types]
(https://github.com/myitcv/react/blob/c336a0f015a717172fe23f04ac441b982c9252db/gen_PProps_reactGen.go#L16-L36))
  * https://github.com/gopherjs/gopherjs/issues/186 - truly minimal JS output
* Add support for `data-*` and `aria-*` attributes on elements
* Support tab characters in syntax viewer
* Add support for deleting generated files for components that no longer exist (i.e. the situation that arises when we rename/delete a component)
* Improve the process by which we `webpack` our React dependencies
* Handle the fact that `componentWillReceiveProps` gets called _before_ `componentShouldUpdate`; i.e. regardless of whether the props have changed or not. Therefore we are likely triggering logs of pointless method calls (when the props haven't changed)
* (Potentially) Move HTML element to a separate `myitcv.io/react/html`
  * Will tidy up `myitcv.io/react`
  * More inline with the fact that we're not limited to only rendering HTML with React (e.g. Native)
* `present` example:
  * Fix `go-bindata` or use similar approach
  * Ensure license attribution etc is correct
  * Rewrite Javascript from template into Go
  * Switch from `<iframe>` approach to pure React approach
* Work out if/how we can integrate with http://gobuffalo.io/docs/getting-started
* Document (and at a later stage) vet that methods should be defined on a non-pointer receiver of a component. Check existing docs are accurate
* Create components for all the HTML 5 elements https://www.w3.org/TR/html5/index.html#elements-1
* Design a better pattern for inlining constant blocks of HTML: https://github.com/myitcv/react/issues/64
* Add tests for:
  * Lifecycle behaviour and ordering
  * That `SetState()` is synchronous everywhere it's valid to call `State()`
  * Expectations of when re-rendering should happen
  * ...
* Clarify component definitions details:
  * Document why they cannot have state (link to lifecycle explanation)
  * Update `reactGen` to ensure that there is no state defined on a component (it should effectively be bare)
* A first cut of `reactVet` (and `reactLint`), a tool to statically catch correctness problems in GopherJS React applications
  * Ensure that string constants to `jsx.*` functions can actually be parsed (avoids runtime errors)
  * Ensure that arguments to `DangerouslyInnerHTML` are constant strings - think this makes sense?
  * Include an example vet rule that errors on nested of `<p>` tags (for example) - base this rule (and related others) on [permitted content rules](https://developer.mozilla.org/en-US/docs/Web/HTML/Element/Heading_Elements) (there is probably a formal encoding of these somewhere?)
  * Verify that constructors return values of type `*XElem` (efficiency)
* Investigate alternative for CSS support: https://github.com/gu-io/gu/tree/master/trees/css
* Lifecycle explanation
  * Add another tab to the `examplesshowcase` that is a demonstration of the lifecycle of components (an outer and an inner component)
  * Create a sequence diagram that explains the lifecycle of components, props changes, state changes etc
* Evaluate whether component definition and props types can collapse into one in some way shape or form
* Demo use of [React Developer Tools](https://chrome.google.com/webstore/detail/react-developer-tools/fmkadmapgofadopljbjfkapdkoienihi?hl=en) extension
* Contribute to https://github.com/tastejs/todomvc-app-template/
* Incorporate some ideas from http://reagent-project.github.io/
* Support within `reactGen` for receiving a pointer to props in a constructor (effectively when props for a user-defined component are optional)
* Remove the _horrendous_ amounts of duplication in the examples showcase
* Document use of `github.com/cespare/reflex` to wrap `go generate`
* Examples:
  * Using local storage
  * ...
* Complete documentation of the code
* Complete static analysis of code:
  * `vet`
  * `lint`
  * `unused`
  * ...
* Complete definition of:
  * Intrinsic components and their props types (React props are subtly different to `honnef.co/go/js/dom`)
  * React types like `SyntheticEvent` etc
  * `CSS` type
* Further work on [code generator](code_generator_design.md)
* Extend `reactGen -init <template>` to understand more templates than simply `minimal`
* Reject any state or props types that are `*js.Object`-special until we are clear on the semantics of such types per https://github.com/gopherjs/gopherjs/issues/236
* When we are clear on the semantic of https://github.com/gopherjs/gopherjs/issues/236, remove/otherwise the [conversion methods](https://github.com/myitcv/gopherjs/blob/648bf1950ae20f0ad155e4faabc276252c7f3ff9/react/gen_DivProps_reactGen.go#L16-L36) we currently generate
* Ensure I have the proper license files included for React, Preact, https://github.com/simonwhitaker/github-fork-ribbon-css etc.
* `componentWillReceiveProps` is [documented](https://facebook.github.io/react/docs/react-component.html#componentwillreceiveprops) as follows: _"is invoked before a mounted component receives new props."_ The wording is [not, however, entirely accurate/precise](https://github.com/facebook/react/issues/3610). `componentWillReceiveProps` is called for a mounted component whenever the parent component re-renders (note this does not imply the component will actually `render`, just that the parent component itself has re-rendered), irrespective of whether the props have changed. This covers the case of no props, props not having changed and props having changed. The slightly loose documentation, plus [this gotcha](gotchas.md#no-requirement-for-shouldcomponentupdate), means that it's slightly weird for us to ever get a callback via `componentWillReceiveProps` if the props haven't actually changed according to struct value comparison. This probably needs to be addressed
* Within `stateGen`, ability to generate a state tree that persists somewhere. Could be browser-local storage, and/or a remote server

### MaybeNext

* [React Native](https://facebook.github.io/react-native/) package and examples

