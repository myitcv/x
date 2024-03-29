// Code generated by myitcv.io/react/cmd/reactGen. DO NOT EDIT.

package hellomessage

import "myitcv.io/react"

type HelloMessageElem struct {
	react.Element
}

func (h *HelloMessageElem) RendersDiv(*react.DivElem) {}

func (h *HelloMessageElem) noop() {
	var v HelloMessageDef
	r := v.Render()

	v.RendersDiv(r)
}

func buildHelloMessage(cd react.ComponentDef) react.Component {
	return HelloMessageDef{ComponentDef: cd}
}

func buildHelloMessageElem(props HelloMessageProps, children ...react.Element) *HelloMessageElem {
	return &HelloMessageElem{
		Element: react.CreateElement(buildHelloMessage, props, children...),
	}
}

func (h HelloMessageDef) RendersElement() react.Element {
	return h.Render()
}

// IsProps is an auto-generated definition so that HelloMessageProps implements
// the myitcv.io/react.Props interface.
func (h HelloMessageProps) IsProps() {}

// Props is an auto-generated proxy to the current props of HelloMessage
func (h HelloMessageDef) Props() HelloMessageProps {
	uprops := h.ComponentDef.Props()
	return uprops.(HelloMessageProps)
}

func (h HelloMessageProps) EqualsIntf(val react.Props) bool {
	return h == val.(HelloMessageProps)
}

var _ react.Props = HelloMessageProps{}
