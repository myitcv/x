// foo_bar.go

package main

import (
	"fmt"

	"myitcv.io/react"
)

//go:generate reactGen

// FooBarDef is the definition of the FooBar component. All components are
// declared with a *Def suffix and an embedded myitcv.io/react.ComponentDef
// field
//
type FooBarDef struct {
	react.ComponentDef
}

// FooBarProps is the props type for the FooBar component. All props types are
// declared as a struct type with a *Props suffix
//
type FooBarProps struct {
	Name string
}

// FooBarState is the state type for the FooBar component. All state types are
// declared as a struct type with a *State suffix
//
type FooBarState struct {
	Age int
}

// FooBar is the constructor for a FooBar component. Given that this component
// can take props (can, not must), we add a parameter of type FooBarProps
//
func FooBar(p FooBarProps) *FooBarElem {
	// every component constructor must call this function
	return buildFooBarElem(p)
}

// Render is a required method on all React components. Notice that the method
// is declared on the type FooBarDef.
//
func (f FooBarDef) Render() react.Element {

	name := f.Props().Name
	age := f.State().Age

	details := fmt.Sprintf("My name is %v. My age is %v", name, age)

	// all React components must render under a single root. This is typically
	// achieved by rendering everything within a <div> elememt
	//
	return react.Div(nil,
		react.P(nil,
			react.S(details),
		),
		react.Button(
			&react.ButtonProps{
				OnClick: ageClick{f},
			},
			react.S("Bump age"),
		),
	)
}

// ageClick implements the react.OnClick interface to handle when the "Bump
// age" button is clicked
//
type ageClick struct{ FooBarDef }

// OnClick is the ageClick implementation of the react.OnClick interface
//
func (a ageClick) OnClick(e *react.SyntheticMouseEvent) {
	s := a.State()
	s.Age++
	a.SetState(s)
}
