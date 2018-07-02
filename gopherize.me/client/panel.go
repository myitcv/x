package main

import (
	"path/filepath"

	r "myitcv.io/react"
)

var blank = filepath.Join("artwork", "whitebox_thumbnail.png")

type PanelProps struct {
	Category *Category
	Open     bool
	Part     int
	Selected string
	Update   UpdateGopher
	Expand   ExpandPanel
}

func Panel(p PanelProps) *PanelElem {
	return buildPanelElem(p)
}

type PanelDef struct {
	r.ComponentDef
}

func (pa PanelDef) Render() r.Element {
	props := pa.Props()

	collapse := " collapse"

	if props.Open {
		collapse = ""
	}

	var imgs []r.Element

	if props.Open {
		for _, o := range props.Category.Options {
			var src string
			class := "item"

			if o == props.Selected {
				class += " selected"
			} else if o == "" {
				class += " none"
			}

			if o == "" {
				src = blank
			} else {
				src = filepath.Join("artwork", o+"_thumbnail.png")
			}

			imgs = append(imgs,
				r.Label(
					&r.LabelProps{
						ClassName: class,
						OnClick: chooseItemClick{
							U:  props.Update,
							ci: props.Part,
							v:  o,
						},
					},
					r.Img(
						&r.ImgProps{Src: src},
					),
				),
			)
		}
	}

	return r.Div(&r.DivProps{ClassName: "panel panel-default"},
		r.Div(&r.DivProps{ClassName: "panel-heading", Role: "tab"},
			r.H4(
				&r.H4Props{ClassName: "panel-title"},
				r.A(
					&r.AProps{
						OnClick: expandClick{
							E: props.Expand,
							i: props.Part,
						},
					},
					r.S(props.Category.Name),
				),
			),
		),
		r.Div(
			&r.DivProps{
				ID:        "Body",
				ClassName: "panel-collapse collapse in",
				Role:      "tabpanel",
			},
			r.Div(
				&r.DivProps{ClassName: "panel-body" + collapse},
				r.Div(nil, imgs...),
			),
		),
	)

}

type expandClick struct {
	E ExpandPanel
	i int
}

func (ex expandClick) OnClick(e *r.SyntheticMouseEvent) {
	ex.E.Expand(ex.i)
	e.PreventDefault()
}

type chooseItemClick struct {
	U  UpdateGopher
	ci int
	v  string
}

func (c chooseItemClick) OnClick(e *r.SyntheticMouseEvent) {
	c.U.UpdateGopher(c.ci, c.v)
	e.PreventDefault()
}
