// Code generated by myitcv.io/react/cmd/reactGen. DO NOT EDIT.

package immtodoapp

import "myitcv.io/react"

type EntryElem struct {
	react.Element
}

func (e *EntryElem) RendersLi(*react.LiElem) {}

func (e *EntryElem) noop() {
	var v EntryDef
	r := v.Render()

	v.RendersLi(r)
}

func buildEntry(cd react.ComponentDef) react.Component {
	return EntryDef{ComponentDef: cd}
}

func buildEntryElem(props EntryProps, children ...react.Element) *EntryElem {
	return &EntryElem{
		Element: react.CreateElement(buildEntry, props, children...),
	}
}

func (e EntryDef) RendersElement() react.Element {
	return e.Render()
}

// IsProps is an auto-generated definition so that EntryProps implements
// the myitcv.io/react.Props interface.
func (e EntryProps) IsProps() {}

// Props is an auto-generated proxy to the current props of Entry
func (e EntryDef) Props() EntryProps {
	uprops := e.ComponentDef.Props()
	return uprops.(EntryProps)
}

func (e EntryProps) EqualsIntf(val react.Props) bool {
	return e == val.(EntryProps)
}

var _ react.Props = EntryProps{}
